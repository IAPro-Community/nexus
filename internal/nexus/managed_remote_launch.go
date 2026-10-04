package nexus

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	remotecontrol "github.com/IAPro-Community/nexus/internal/control/remote"
	"github.com/IAPro-Community/nexus/internal/core/flags"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

type managedRemoteBridge interface {
	remotecontrol.ManagedBridgeControl
	Spawn(context.Context, protocol.RemoteBridgeSpawnPayload) (protocol.RemoteBridgeSpawnData, error)
}

var openManagedRemoteBridge = func(
	ctx context.Context,
	targetID string,
	location string,
) (managedRemoteBridge, protocol.RemoteBridgeHelloData, error) {
	client, hello, err := remotecontrol.OpenSSHBridge(ctx, targetID, location)
	if err != nil {
		return nil, protocol.RemoteBridgeHelloData{}, err
	}
	return client, hello, nil
}

func validateManagedRemoteAgentConfig(project store.Project, cfg AgentConfig) error {
	effectiveIsolation := strings.ToLower(strings.TrimSpace(cfg.Isolation))
	if effectiveIsolation == "" {
		effectiveIsolation = strings.ToLower(strings.TrimSpace(project.DefaultIsolation))
	}
	if effectiveIsolation == "worktree" {
		return fmt.Errorf("managed SSH does not yet support remote worktree isolation")
	}
	if strings.TrimSpace(cfg.Workspace) != "" {
		return fmt.Errorf("managed SSH uses the target workspace; Agent workspace override is not supported yet")
	}
	if len(cfg.Environment) > 0 {
		return fmt.Errorf("managed SSH does not transfer Agent environment values; configure the remote host instead")
	}
	for key := range cfg.Options {
		switch key {
		case "mode", "origin", "role_type":
			// UI/identity metadata only; it does not alter the provider process.
		default:
			return fmt.Errorf("managed SSH does not transfer Agent option %q yet", key)
		}
	}
	return nil
}

func (n *Nexus) launchAgentOnManagedTarget(
	ctx context.Context,
	project store.Project,
	agent store.Agent,
	cfg AgentConfig,
	continuity continuityLaunch,
	accountScope model.AccountScope,
	runtimeID string,
	binding ManagedExecutionTargetBinding,
) (*registry.RuntimeSession, error) {
	if !flags.Enabled(flags.RemoteExecution) {
		return nil, fmt.Errorf("%w: remote_execution feature flag is off", ErrManagedRemoteExecutionDisabled)
	}
	if err := validateManagedRemoteAgentConfig(project, cfg); err != nil {
		return nil, err
	}
	manifest, _, err := LoadProjectManifest(project.CanonicalPath)
	if err != nil {
		return nil, err
	}
	target, ok := manifest.ProjectExecutionTarget(binding.TargetID)
	if !ok {
		return nil, fmt.Errorf("%w: bound target %q is no longer declared", ErrExecutionTargetApprovalStale, binding.TargetID)
	}
	if target.Kind != "ssh" {
		return nil, fmt.Errorf("bound execution target %q kind %q is not supported", target.ID, target.Kind)
	}

	bridge, hello, err := openManagedRemoteBridge(ctx, target.ID, target.Location)
	if err != nil {
		return nil, fmt.Errorf("open managed SSH target %q: %w", target.ID, err)
	}
	cleanupBridge := true
	defer func() {
		if cleanupBridge {
			_ = bridge.Close()
		}
	}()

	if !hello.RemoteExecutionEnabled {
		return nil, fmt.Errorf("remote_execution is disabled on target %q", target.ID)
	}
	if target.MachineID != "" && hello.MachineID != target.MachineID {
		return nil, fmt.Errorf(
			"%w: target %q expected machine %q but remote bridge reported %q",
			ErrExecutionTargetIdentityMismatch,
			target.ID,
			target.MachineID,
			hello.MachineID,
		)
	}
	if binding.MachineID != hello.MachineID {
		return nil, fmt.Errorf(
			"%w: approved machine %q changed to %q",
			ErrExecutionTargetApprovalStale,
			binding.MachineID,
			hello.MachineID,
		)
	}
	currentToken := projectExecutionTargetApprovalToken(project, target, hello)
	if subtle.ConstantTimeCompare([]byte(binding.ApprovalToken), []byte(currentToken)) != 1 {
		return nil, fmt.Errorf("%w: target fingerprint changed; test and activate it again", ErrExecutionTargetApprovalStale)
	}

	spawn, err := bridge.Spawn(ctx, protocol.RemoteBridgeSpawnPayload{
		RuntimeID: runtimeID,
		AgentID:   agent.ID,
		ProjectID: project.ID,
		Provider:  cfg.Provider,
		Profile:   cfg.Profile,
		Model:     cfg.Model,
		Workspace: target.Workspace,
		Args:      append([]string(nil), continuity.Args...),
		Rows:      24,
		Cols:      80,
	})
	if err != nil {
		return nil, fmt.Errorf("spawn remote runtime: %w", err)
	}

	status, err := bridge.Status(ctx, runtimeID)
	if err != nil {
		_, _ = bridge.Control(context.Background(), runtimeID, protocol.CmdTerminate, nil)
		return nil, fmt.Errorf("verify remote runtime status: %w", err)
	}
	if status.RuntimeID != runtimeID ||
		status.MachineID != hello.MachineID ||
		status.Provider != cfg.Provider ||
		status.Profile != cfg.Profile ||
		status.Workspace != target.Workspace {
		_, _ = bridge.Control(context.Background(), runtimeID, protocol.CmdTerminate, nil)
		return nil, fmt.Errorf("remote runtime identity did not match the approved launch")
	}

	session := registry.RuntimeSession{
		RuntimeID:     runtimeID,
		AgentID:       agent.ID,
		ProjectID:     project.ID,
		ProjectName:   project.Name,
		Title:         strings.TrimSpace(agent.Name),
		ProviderID:        cfg.Provider,
		ProfileID:         cfg.Profile,
		AccountScope:      accountScope,
		ProviderSessionID: continuity.ProviderSessionID,
		Model:             cfg.Model,
		Workspace:     target.Workspace,
		PID:           spawn.ProviderPID,
		State:         registry.StateRunning,
		ControlLevel:  registry.ControlLevel(status.ControlLevel),
		StartedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
		MachineID:     hello.MachineID,
		Location:      target.Location,
		Continuity:    continuity.Status,
		Transport:     registry.TransportManagedSSHProxy,
		Labels: map[string]string{
			"nexus.remote_target_id": target.ID,
			"nexus.remote_transport": "ssh",
		},
	}

	proxy, localSession, err := remotecontrol.StartManagedRuntimeProxy(ctx, remotecontrol.ManagedRuntimeProxyConfig{
		Session:  session,
		TargetID: target.ID,
		Location: target.Location,
		Bridge:   bridge,
		Registry: n.runtimeRegistry(),
	})
	if err != nil {
		_, _ = bridge.Control(context.Background(), runtimeID, protocol.CmdTerminate, nil)
		return nil, fmt.Errorf("start local remote-runtime proxy: %w", err)
	}
	if err := n.registerRemoteProxy(runtimeID, proxy); err != nil {
		_, _ = bridge.Control(context.Background(), runtimeID, protocol.CmdTerminate, nil)
		_ = proxy.Close()
		return nil, err
	}

	cleanupBridge = false
	return &localSession, nil
}
