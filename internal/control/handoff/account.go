package handoff

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/host"
	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/profile"
)

// HandoffState represents the phase in the transactional account handoff process.
type HandoffState string

const (
	HandoffRequested       HandoffState = "REQUESTED"
	HandoffPreflight       HandoffState = "PREFLIGHT"
	HandoffTargetValidated HandoffState = "TARGET_VALIDATED"
	HandoffCheckpointed    HandoffState = "CHECKPOINTED"
	HandoffSourceQuiesced  HandoffState = "SOURCE_QUIESCED"
	HandoffTargetStarting  HandoffState = "TARGET_STARTING"
	HandoffTargetResumed   HandoffState = "TARGET_RESUMED"
	// HandoffContinuityUnverified means the runtime and resume arguments were
	// verified locally, but the provider did not expose a confirmation channel.
	// HandoffVerified is reserved for a future provider-level acknowledgement.
	HandoffContinuityUnverified HandoffState = "NATIVE_RESUME_UNVERIFIED"
	HandoffVerified             HandoffState = "VERIFIED"
	HandoffCompleted            HandoffState = "COMPLETED"
	HandoffRollback             HandoffState = "ROLLBACK_REQUIRED"
	HandoffRollingBack          HandoffState = "ROLLING_BACK"
	HandoffRolledBack           HandoffState = "ROLLED_BACK"
	HandoffFailedSafe           HandoffState = "FAILED_SAFE"
	HandoffFailedUnsafe         HandoffState = "FAILED_UNSAFE"
)

// Transaction encapsulates the context and state of an account handoff attempt.
type Transaction struct {
	State          HandoffState
	SourceSession  registry.RuntimeSession
	TargetProvider string
	TargetProfile  string
	Checkpoint     WorkCheckpoint
	TargetSession  *registry.RuntimeSession
	TargetHost     *host.SessionHost
	Error          error
}

// PerformAccountHandoff executes a safe, transactional account handoff to another profile of the SAME provider.
func PerformAccountHandoff(ctx context.Context, sourceRuntimeID, targetSpec string) (*registry.RuntimeSession, error) {
	return defaultService().PerformAccountHandoff(ctx, sourceRuntimeID, targetSpec)
}

// PerformAccountHandoff executes a safe account handoff using owned
// control-plane dependencies.
func (s *Service) PerformAccountHandoff(ctx context.Context, sourceRuntimeID, targetSpec string) (*registry.RuntimeSession, error) {
	tx := &Transaction{
		State: HandoffRequested,
	}

	reg := s.registry
	source, ok := reg.Get(sourceRuntimeID)
	if !ok {
		return nil, fmt.Errorf("source runtime %q not found", sourceRuntimeID)
	}
	tx.SourceSession = source

	// 1. Parse target "provider:profile" or "profile"
	targetProvider := source.ProviderID
	targetProfile := targetSpec
	if idx := strings.Index(targetSpec, ":"); idx != -1 {
		targetProvider = targetSpec[:idx]
		targetProfile = targetSpec[idx+1:]
	}
	tx.TargetProvider = targetProvider
	tx.TargetProfile = targetProfile

	// 2. Strict Provider match validation
	if targetProvider != source.ProviderID {
		return nil, fmt.Errorf("account handoff requires matching provider (source: %s, target: %s). Use cross-provider context handoff instead", source.ProviderID, targetProvider)
	}

	// 3. Strict Profile difference validation
	if targetProfile == source.ProfileID {
		return nil, fmt.Errorf("target profile is identical to current active profile %q", source.ProfileID)
	}

	// 4. Strict Session ID presence validation
	if strings.TrimSpace(source.ProviderSessionID) == "" {
		return nil, fmt.Errorf("handoff unavailable: source provider session ID is unknown. Cannot guarantee session continuity")
	}

	tx.State = HandoffPreflight

	// 5. Preflight: Target Profile Authentication & Driver Capabilities
	info := profile.GetAccountInfo(targetProvider, targetProfile)
	if !info.Authenticated {
		return nil, fmt.Errorf("target profile %s:%s is not authenticated", targetProvider, targetProfile)
	}

	d, err := s.drivers.Get(targetProvider)
	if err != nil {
		return nil, fmt.Errorf("target driver error: %w", err)
	}

	canResume, reason := d.CanResume(ctx, model.Profile{Name: targetProfile, Provider: targetProvider}, source.ProviderSessionID)
	if !canResume {
		return nil, fmt.Errorf("resume unavailable for %s:%s: %s", targetProvider, targetProfile, reason)
	}

	resumeArgs, err := d.BuildResumeArgs(ctx, model.Profile{Name: targetProfile, Provider: targetProvider}, source.ProviderSessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to construct resume arguments: %w", err)
	}

	tx.State = HandoffTargetValidated

	// 6. Capture work checkpoint
	tx.Checkpoint = CaptureWorkCheckpoint(source.Workspace, source.RuntimeID, source.ProviderID, source.ProfileID, source.ProviderSessionID, source.Model, "")
	if _, err := SaveCheckpoint(tx.Checkpoint); err != nil {
		tx.State = HandoffFailedSafe
		return nil, fmt.Errorf("account handoff aborted: failed to persist mandatory work checkpoint: %w (FAILED_SAFE)", err)
	}
	tx.State = HandoffCheckpointed

	// 7. Quiesce and gracefully stop source runtime
	client, err := protocol.NewClient(sourceRuntimeID)
	if err == nil {
		_ = client.Stop()
		_ = client.Close()
	}
	// Wait up to 2 seconds for source process to stop, while honoring caller
	// cancellation before starting the target runtime.
	waitForProcessStop(ctx, source.PID, 2*time.Second)
	if err := ctx.Err(); err != nil {
		tx.State = HandoffFailedSafe
		_ = reg.UpdateState(sourceRuntimeID, registry.StateRunning)
		return nil, fmt.Errorf("account handoff canceled before target start: %w", err)
	}
	// Source stop is a hard barrier (H3): never start the target while the source
	// may still be a writer for the same provider session.
	if registry.IsProcessAlive(source.PID) {
		tx.State = HandoffFailedSafe
		_ = reg.UpdateState(sourceRuntimeID, registry.StateRunning)
		return nil, fmt.Errorf("account handoff aborted (FAILED_SAFE): source runtime %s did not stop within policy; refusing to start target while source may still write the session", sourceRuntimeID)
	}
	_ = reg.UpdateState(sourceRuntimeID, registry.StateHandoff)
	tx.State = HandoffSourceQuiesced

	// 8. Start target runtime using unified launcher
	tx.State = HandoffTargetStarting
	newRuntimeID := fmt.Sprintf("%s-handoff-%s", targetProvider, ids.NewRuntimeID())
	lineageID := fmt.Sprintf("lin-%s-%s", targetProvider, ids.NewRuntimeID())

	targetSession, err := s.launcher.Launch(ctx, launcher.LaunchOptions{
		RuntimeID:         newRuntimeID,
		AgentID:           source.AgentID,
		ProjectID:         source.ProjectID,
		ProjectName:       source.ProjectName,
		ProviderID:        targetProvider,
		ProfileID:         targetProfile,
		ProviderSessionID: source.ProviderSessionID,
		Workspace:         source.Workspace,
		Args:              resumeArgs,
		Model:             source.Model,
		Standalone:        false,
		Labels:            cloneRuntimeLabels(source.Labels),
	})
	if err != nil {
		tx.State = HandoffRollback
		return nil, tx.rollback(ctx, d, source, s, fmt.Errorf("failed to start target runtime: %w", err))
	}

	tx.TargetSession = targetSession
	tx.State = HandoffTargetResumed

	// Verify continuity before claiming VERIFIED: the target must be running and
	// its resume command must actually reference the source session ID.
	if ok, reason := VerifyResumeContinuity(targetSession, resumeArgs, source.ProviderSessionID); !ok {
		tx.State = HandoffRollback
		// Never orphan the freshly launched target before rolling the source back.
		stopRuntime(targetSession.RuntimeID)
		return nil, tx.rollback(ctx, d, source, s, fmt.Errorf("resume continuity verification failed: %s", reason))
	}

	targetSession.ProviderSessionID = source.ProviderSessionID
	targetSession.State = registry.StateRunning
	targetSession.ParentRuntimeID = source.RuntimeID
	targetSession.HandoffType = "account"
	targetSession.LineageID = lineageID
	if targetSession.Labels == nil {
		targetSession.Labels = map[string]string{}
	}
	targetSession.Labels["continuity"] = string(HandoffContinuityUnverified)
	_ = reg.Register(*targetSession)
	tx.State = HandoffContinuityUnverified

	// 10. Record Lineage
	if err := RecordLineage(LineageRecord{
		LineageID:               lineageID,
		SourceRuntimeID:         source.RuntimeID,
		SourceProviderSessionID: source.ProviderSessionID,
		TargetRuntimeID:         targetSession.RuntimeID,
		TargetProviderSessionID: targetSession.ProviderSessionID,
		Type:                    "ACCOUNT_HANDOFF",
		CreatedAt:               time.Now(),
		CheckpointID:            tx.Checkpoint.CheckpointID,
	}); err != nil {
		slog.Warn("Failed to record lineage during account handoff", "err", err)
	}

	tx.State = HandoffCompleted

	events.DefaultBus().Publish(events.NewEventWithCorrelation(
		lineageID,
		targetSession.RuntimeID,
		targetSession.ProviderID,
		targetSession.ProfileID,
		events.EventHandoffCompleted,
		fmt.Sprintf("Account handoff completed from %s to %s", source.RuntimeID, targetSession.RuntimeID),
		map[string]any{"source_id": source.RuntimeID, "target_id": targetSession.RuntimeID, "session_id": targetSession.ProviderSessionID, "continuity": string(HandoffContinuityUnverified)},
	))

	return targetSession, nil
}

func (tx *Transaction) rollback(ctx context.Context, d driver.ControlDriver, source registry.RuntimeSession, s *Service, cause error) error {
	tx.State = HandoffRollingBack
	reg := s.registry

	if registry.IsProcessAlive(source.PID) {
		_ = reg.UpdateState(source.RuntimeID, registry.StateRunning)
		tx.State = HandoffRolledBack
		return fmt.Errorf("handoff failed (%w); source session remained alive and was restored to RUNNING (FAILED_SAFE)", cause)
	}

	// Try to restore source session
	resumeArgs, err := d.BuildResumeArgs(ctx, model.Profile{Name: source.ProfileID, Provider: source.ProviderID}, source.ProviderSessionID)
	if err != nil {
		tx.State = HandoffFailedUnsafe
		return fmt.Errorf("handoff failed (%w) and rollback command build failed: %v (FAILED_UNSAFE)", cause, err)
	}

	recoverRuntimeID := fmt.Sprintf("%s-recovered-%s", source.ProviderID, ids.NewRuntimeID())
	recoverSession, err := s.launcher.Launch(ctx, launcher.LaunchOptions{
		RuntimeID:         recoverRuntimeID,
		ProviderID:        source.ProviderID,
		ProfileID:         source.ProfileID,
		ProviderSessionID: source.ProviderSessionID,
		Workspace:         source.Workspace,
		Args:              resumeArgs,
		Standalone:        false,
	})
	if err != nil {
		tx.State = HandoffFailedUnsafe
		return fmt.Errorf("handoff failed (%w) and rollback execution failed: %v (FAILED_UNSAFE)", cause, err)
	}

	recoverSession.ParentRuntimeID = source.RuntimeID
	recoverSession.HandoffType = "rollback"
	_ = reg.Register(*recoverSession)

	tx.State = HandoffRolledBack
	return fmt.Errorf("handoff failed (%w); successfully restored source session on recovery runtime %s (ROLLED_BACK)", cause, recoverRuntimeID)
}

// stopRuntime forcefully terminates a freshly started runtime so a failed
// handoff never orphans a live target holding the provider session.
func stopRuntime(runtimeID string) {
	if runtimeID == "" {
		return
	}
	if c, err := protocol.NewClient(runtimeID); err == nil {
		_, _ = c.Send(protocol.CmdTerminate, nil)
		_ = c.Close()
	}
}
