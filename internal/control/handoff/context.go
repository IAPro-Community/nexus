package handoff

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/cooldown"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/core/scheduler"
	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/profile"
)

// FormatKickoffPrompt produces a clean, honest initial prompt for the target provider from a WorkCheckpoint.
func FormatKickoffPrompt(cp WorkCheckpoint) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "=== IAPro Nexus Context Handoff (from %s:%s) ===\n", strings.ToUpper(cp.SourceProvider), cp.SourceProfile)
	fmt.Fprintf(&sb, "Workspace: %s\n", cp.Workspace)
	if cp.SourceModel != "" {
		fmt.Fprintf(&sb, "Previous Model: %s\n", cp.SourceModel)
	}
	if cp.GitBranch != "" {
		fmt.Fprintf(&sb, "Active Git Branch: %s\n", cp.GitBranch)
	}
	if len(cp.ChangedFiles) > 0 {
		fmt.Fprintf(&sb, "Modified Files (%d):\n", len(cp.ChangedFiles))
		for _, f := range cp.ChangedFiles {
			fmt.Fprintf(&sb, " - %s\n", f)
		}
	}
	if cp.GitDiffStat != "" {
		sb.WriteString("Git Diff Summary:\n" + cp.GitDiffStat + "\n")
	}
	if cp.Goal != "" {
		fmt.Fprintf(&sb, "Current Goal / Task: %s\n", cp.Goal)
	}
	if len(cp.MaestroContext.ReadOrder) > 0 {
		sb.WriteString("Maestro continuity contract:\n")
		sb.WriteString("Read these workspace files in order before changing anything:\n")
		for _, path := range cp.MaestroContext.ReadOrder {
			status := ""
			for _, file := range cp.MaestroContext.Files {
				if file.Path == path {
					status = " [" + file.Status + "]"
					break
				}
			}
			fmt.Fprintf(&sb, " - %s%s\n", path, status)
		}
		sb.WriteString("Reconcile Git status and the checkpoint before repeating work.\n")
	}
	sb.WriteString("==================================================\n")
	sb.WriteString("Please inspect the modified files and continue the ongoing task in this workspace.")
	return security.Redact(sb.String())
}

// PerformContextHandoff creates a new session on a DIFFERENT provider using a captured WorkCheckpoint.
func PerformContextHandoff(ctx context.Context, sourceRuntimeID, targetProvider, targetProfile string) (*registry.RuntimeSession, error) {
	return defaultService().PerformContextHandoff(ctx, sourceRuntimeID, targetProvider, targetProfile)
}

// PerformContextHandoff creates a new session using owned control-plane
// dependencies.
func (s *Service) PerformContextHandoff(ctx context.Context, sourceRuntimeID, targetProvider, targetProfile string) (*registry.RuntimeSession, error) {
	reg := s.registry
	source, ok := reg.Get(sourceRuntimeID)
	if !ok {
		return nil, fmt.Errorf("source runtime %q not found", sourceRuntimeID)
	}

	// 1. Resolve Target Profile using Smart Account Selector if unspecified
	if targetProfile == "" {
		cfg, _ := config.LoadConfig()
		qEng := quota.NewEngine(5 * time.Minute)
		cdTracker := cooldown.NewTracker()
		sel := scheduler.NewSelector(cfg, qEng, cdTracker)

		allProfiles, _ := profile.List()
		var candidates []model.Profile
		accounts := make(map[string]model.AccountInfo)
		for _, p := range allProfiles {
			if p.Provider == targetProvider {
				candidates = append(candidates, p)
				accounts[p.Name] = profile.GetAccountInfo(targetProvider, p.Name)
			}
		}

		if len(candidates) > 0 {
			res, _ := sel.SelectBestProfile(ctx, targetProvider, source.Workspace, candidates, accounts, nil)
			if res != nil && res.SelectedProfile != nil && res.SelectedProfile.Name != "" {
				targetProfile = res.SelectedProfile.Name
			} else {
				targetProfile = candidates[0].Name
			}
		} else {
			targetProfile = "default"
		}
	}

	// 2. Validate Target Authentication
	info := profile.GetAccountInfo(targetProvider, targetProfile)
	if !info.Authenticated {
		return nil, fmt.Errorf("target provider profile %s:%s is not authenticated", targetProvider, targetProfile)
	}

	// 3. Capture Safe Work Checkpoint
	cp := CaptureWorkCheckpoint(source.Workspace, source.RuntimeID, source.ProviderID, source.ProfileID, source.ProviderSessionID, source.Model, "")
	if _, err := SaveCheckpoint(cp); err != nil {
		return nil, fmt.Errorf("context handoff aborted: failed to persist work checkpoint: %w", err)
	}

	kickoffPrompt := FormatKickoffPrompt(cp)

	// 4. Get Target Driver & Build Kickoff Command
	d, err := s.drivers.Get(targetProvider)
	if err != nil {
		return nil, fmt.Errorf("target driver error: %w", err)
	}

	extraArgs, err := d.BuildKickoffArgs(ctx, model.Profile{Name: targetProfile, Provider: targetProvider}, kickoffPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to construct kickoff arguments: %w", err)
	}

	// 4b. Resolve model for target provider
	targetModel := ResolveTargetModel(source.Model, source.ProviderID, targetProvider)

	// 5. Quiesce and gracefully stop source runtime before launching target
	// to prevent concurrent writers on the same workspace.
	client, err := protocol.NewClient(sourceRuntimeID)
	if err == nil {
		_ = client.Stop()
		_ = client.Close()
	}
	waitForProcessStop(ctx, source.PID, 2*time.Second)
	if err := ctx.Err(); err != nil {
		_ = reg.UpdateState(sourceRuntimeID, registry.StateRunning)
		return nil, fmt.Errorf("context handoff canceled before target start: %w", err)
	}
	_ = reg.UpdateState(sourceRuntimeID, registry.StateHandoff)

	// 6. Launch Target Supervised Runtime via unified launcher (persistent host)
	newRuntimeID := fmt.Sprintf("%s-continue-%s", targetProvider, ids.NewRuntimeID())
	lineageID := fmt.Sprintf("lin-ctx-%s", ids.NewRuntimeID())

	newSession, err := s.launcher.Launch(ctx, launcher.LaunchOptions{
		RuntimeID:   newRuntimeID,
		AgentID:     source.AgentID,
		ProjectID:   source.ProjectID,
		ProjectName: source.ProjectName,
		ProviderID:  targetProvider,
		ProfileID:   targetProfile,
		Workspace:   source.Workspace,
		Args:        extraArgs,
		Model:       targetModel,
		Standalone:  false,
	})
	if err != nil {
		// Rollback source state on target launch failure
		_ = reg.UpdateState(sourceRuntimeID, registry.StateRunning)
		return nil, fmt.Errorf("failed to start context handoff target runtime: %w", err)
	}

	newSession.ParentRuntimeID = source.RuntimeID
	newSession.HandoffType = "context"
	newSession.LineageID = lineageID
	if newSession.Labels == nil {
		newSession.Labels = map[string]string{}
	}
	newSession.Labels["continuity"] = "CONTEXT_HANDOFF"
	_ = reg.Register(*newSession)

	// 7. Record Lineage
	if err := RecordLineage(LineageRecord{
		LineageID:               lineageID,
		SourceRuntimeID:         source.RuntimeID,
		SourceProviderSessionID: source.ProviderSessionID,
		TargetRuntimeID:         newSession.RuntimeID,
		Type:                    "CONTEXT_HANDOFF",
		CreatedAt:               time.Now(),
		CheckpointID:            cp.CheckpointID,
	}); err != nil {
		slog.Warn("Failed to record lineage during context handoff", "err", err)
	}

	events.DefaultBus().Publish(newContextHandoffCompletedEvent(lineageID, source, *newSession, cp.CheckpointID))

	return newSession, nil
}

func newContextHandoffCompletedEvent(lineageID string, source, target registry.RuntimeSession, checkpointID string) events.Event {
	return events.NewEventWithCorrelation(
		lineageID,
		target.RuntimeID,
		target.ProviderID,
		target.ProfileID,
		events.EventHandoffCompleted,
		fmt.Sprintf("Context handoff completed from %s to %s", source.RuntimeID, target.RuntimeID),
		map[string]any{"source_id": source.RuntimeID, "target_id": target.RuntimeID, "checkpoint_id": checkpointID, "continuity": "CONTEXT_HANDOFF"},
	)
}

// ResolveTargetModel determines whether the source model can be safely used on the target provider.
// If both providers are identical, the model is preserved unconditionally.
// If providers differ, we test whether the target provider's driver supports model overrides
// via ApplyLaunchConfiguration dry-run. If unsupported, returns "" to fall back cleanly to target default.
func ResolveTargetModel(sourceModel, sourceProvider, targetProvider string) string {
	sourceModel = strings.TrimSpace(sourceModel)
	if sourceModel == "" {
		return ""
	}
	if strings.EqualFold(sourceProvider, targetProvider) {
		return sourceModel
	}
	if _, err := driver.ApplyLaunchConfiguration(targetProvider, sourceModel, nil, nil); err != nil {
		return ""
	}
	return sourceModel
}
