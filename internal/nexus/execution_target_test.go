package nexus

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestResolveExecutionTargetDefaultsToLocal(t *testing.T) {
	project := store.Project{CanonicalPath: "/workspace"}
	target := ResolveExecutionTarget(project, AgentConfig{})
	if target.ID != "local" || target.Kind != runner.ExecutionTargetLocal || target.Transport != "ipc" {
		t.Fatalf("unexpected local target: %+v", target)
	}
}

func TestResolveExecutionTargetProjectsExistingSSHCommandTemplate(t *testing.T) {
	project := store.Project{CanonicalPath: "/host/project"}
	cfg := AgentConfig{Options: map[string]any{
		"execution_adapter":  "command_template",
		"command_template":   "ssh devbox {provider} {args}",
		"target_kind":        "ssh",
		"target_workspace":   "/srv/project",
		"target_location":    "devbox",
		"target_machine_id":  "machine-devbox",
	}}
	target := ResolveExecutionTarget(project, cfg)
	if target.Kind != runner.ExecutionTargetSSH || target.Transport != "ssh" {
		t.Fatalf("existing SSH target was not projected: %+v", target)
	}
	if target.Workspace != "/srv/project" || target.Location != "devbox" || target.MachineID != "machine-devbox" {
		t.Fatalf("SSH target metadata lost: %+v", target)
	}
	if target.ID == "" || target.ID == "local" {
		t.Fatalf("SSH target requires a stable non-local id: %+v", target)
	}
}

func TestResolveExecutionTargetIDIsStable(t *testing.T) {
	project := store.Project{CanonicalPath: "/host/project"}
	cfg := AgentConfig{Options: map[string]any{
		"execution_adapter": "command_template",
		"command_template":  "ssh devbox {provider} {args}",
		"target_kind":       "ssh",
	}}
	first := ResolveExecutionTarget(project, cfg)
	second := ResolveExecutionTarget(project, cfg)
	if first.ID != second.ID {
		t.Fatalf("target id must be stable: %q != %q", first.ID, second.ID)
	}
}


func TestPreviewProjectExecutionTargetsDefaultsToLocal(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	project, err := st.CreateProject(store.Project{Name: "Preview", CanonicalPath: projectDir})
	if err != nil {
		t.Fatal(err)
	}

	catalog, err := n.PreviewProjectExecutionTargets(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.ManifestPresent || catalog.DeclaredDefaultTarget != "local" || catalog.EffectiveDefaultTarget != "local" {
		t.Fatalf("unexpected local catalog: %+v", catalog)
	}
	if len(catalog.Targets) != 1 || catalog.Targets[0].ID != "local" || !catalog.Targets[0].Effective || !catalog.Targets[0].Selectable {
		t.Fatalf("local target must be ready and effective: %+v", catalog.Targets)
	}
}

func TestPreviewProjectExecutionTargetsKeepsSSHNonEffective(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_REMOTE_EXECUTION", "")
	t.Setenv("NEXUS_DISABLE_REMOTE_EXECUTION", "")

	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	project, err := st.CreateProject(store.Project{Name: "Remote Preview", CanonicalPath: projectDir})
	if err != nil {
		t.Fatal(err)
	}
	manifest := `version: 1
execution:
  defaultTarget: devbox
  targets:
    - id: devbox
      kind: ssh
      location: devbox
      machineId: machine-devbox
      workspace: /srv/project
`
	if err := os.WriteFile(filepath.Join(projectDir, "nexus.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	catalog, err := n.PreviewProjectExecutionTargets(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.ManifestPresent || catalog.DeclaredDefaultTarget != "devbox" || catalog.EffectiveDefaultTarget != "local" {
		t.Fatalf("remote declaration must not change effective target: %+v", catalog)
	}
	if catalog.RemoteExecutionEnabled {
		t.Fatal("remote execution must default off")
	}
	if len(catalog.Targets) != 2 {
		t.Fatalf("expected local + devbox targets, got %+v", catalog.Targets)
	}
	remote := catalog.Targets[1]
	if remote.ID != "devbox" || remote.Kind != runner.ExecutionTargetSSH || remote.Selectable || remote.Effective || remote.ActivationStatus != "FEATURE_DISABLED" {
		t.Fatalf("remote target must remain disabled: %+v", remote)
	}
}

func TestPreviewProjectExecutionTargetsFeatureFlagStillDoesNotActivateSSH(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_REMOTE_EXECUTION", "1")

	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	project, err := st.CreateProject(store.Project{Name: "Remote Preview", CanonicalPath: projectDir})
	if err != nil {
		t.Fatal(err)
	}
	manifest := `version: 1
execution:
  defaultTarget: devbox
  targets:
    - id: devbox
      kind: ssh
      location: devbox
      workspace: /srv/project
`
	if err := os.WriteFile(filepath.Join(projectDir, "nexus.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	catalog, err := n.PreviewProjectExecutionTargets(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.RemoteExecutionEnabled || catalog.EffectiveDefaultTarget != "local" {
		t.Fatalf("feature flag must expose capability without activation: %+v", catalog)
	}
	remote := catalog.Targets[1]
	if !remote.Selectable || remote.Effective || remote.ActivationStatus != "PROBE_REQUIRED" {
		t.Fatalf("SSH must require an explicit probe/activation before becoming effective: %+v", remote)
	}
}


type fakeProjectSSHBridge struct {
	closed bool
}

func (f *fakeProjectSSHBridge) Close() error {
	f.closed = true
	return nil
}

func withProjectSSHBridgeOpener(
	t *testing.T,
	opener func(context.Context, string, string) (projectSSHBridge, protocol.RemoteBridgeHelloData, error),
) {
	t.Helper()
	previous := openProjectSSHBridge
	openProjectSSHBridge = opener
	t.Cleanup(func() { openProjectSSHBridge = previous })
}

func TestProbeProjectExecutionTargetRequiresLocalFeatureFlag(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_REMOTE_EXECUTION", "")
	t.Setenv("NEXUS_DISABLE_REMOTE_EXECUTION", "")

	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	project, err := st.CreateProject(store.Project{Name: "Probe", CanonicalPath: projectDir})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "nexus.yaml"), []byte(`version: 1
execution:
  targets:
    - id: devbox
      kind: ssh
      location: devbox
      workspace: /srv/project
`), 0o600); err != nil {
		t.Fatal(err)
	}

	called := false
	withProjectSSHBridgeOpener(t, func(context.Context, string, string) (projectSSHBridge, protocol.RemoteBridgeHelloData, error) {
		called = true
		return nil, protocol.RemoteBridgeHelloData{}, errors.New("must not dial")
	})

	if _, err := n.ProbeProjectExecutionTarget(context.Background(), project.ID, "devbox"); err == nil {
		t.Fatal("probe must fail when local remote_execution is disabled")
	}
	if called {
		t.Fatal("disabled probe attempted SSH")
	}
}

func TestProbeProjectExecutionTargetUsesOnlyManifestTargetAndChecksMachineIdentity(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_REMOTE_EXECUTION", "1")

	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	project, err := st.CreateProject(store.Project{Name: "Probe", CanonicalPath: projectDir})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "nexus.yaml"), []byte(`version: 1
execution:
  defaultTarget: devbox
  targets:
    - id: devbox
      kind: ssh
      location: ssh-devbox
      machineId: machine-devbox
      workspace: /srv/project
`), 0o600); err != nil {
		t.Fatal(err)
	}

	fake := &fakeProjectSSHBridge{}
	var gotTarget, gotLocation string
	withProjectSSHBridgeOpener(t, func(_ context.Context, targetID, location string) (projectSSHBridge, protocol.RemoteBridgeHelloData, error) {
		gotTarget, gotLocation = targetID, location
		return fake, protocol.RemoteBridgeHelloData{
			ProtocolVersion:        protocol.ProtocolVersion,
			MachineID:              "machine-devbox",
			OS:                     "linux",
			Arch:                   "amd64",
			NexusVersion:           "1.1.0-test",
			RemoteExecutionEnabled: true,
			Capabilities: []protocol.RemoteBridgeCapability{
				protocol.RemoteBridgeCapabilitySpawn,
				protocol.RemoteBridgeCapabilityCWD,
				protocol.RemoteBridgeCapabilityPTY,
				protocol.RemoteBridgeCapabilityResize,
				protocol.RemoteBridgeCapabilityLiveness,
			},
		}, nil
	})

	probe, err := n.ProbeProjectExecutionTarget(context.Background(), project.ID, "devbox")
	if err != nil {
		t.Fatal(err)
	}
	if gotTarget != "devbox" || gotLocation != "ssh-devbox" {
		t.Fatalf("probe did not resolve manifest target: target=%q location=%q", gotTarget, gotLocation)
	}
	if !fake.closed {
		t.Fatal("probe did not close SSH bridge")
	}
	if !probe.Ready || probe.ApprovalToken == "" || probe.TargetID != "devbox" || probe.MachineID != "machine-devbox" ||
		probe.Location != "ssh-devbox" || probe.Workspace != "/srv/project" ||
		probe.OS != "linux" || probe.Arch != "amd64" || probe.NexusVersion != "1.1.0-test" {
		t.Fatalf("unexpected probe: %+v", probe)
	}
}

func TestProbeProjectExecutionTargetReportsReachableButDisabledRemoteHost(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_REMOTE_EXECUTION", "1")

	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	project, err := st.CreateProject(store.Project{Name: "Probe", CanonicalPath: projectDir})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "nexus.yaml"), []byte(`version: 1
execution:
  targets:
    - id: devbox
      kind: ssh
      location: devbox
      workspace: /srv/project
`), 0o600); err != nil {
		t.Fatal(err)
	}

	withProjectSSHBridgeOpener(t, func(context.Context, string, string) (projectSSHBridge, protocol.RemoteBridgeHelloData, error) {
		return &fakeProjectSSHBridge{}, protocol.RemoteBridgeHelloData{
			ProtocolVersion:        protocol.ProtocolVersion,
			MachineID:              "machine-devbox",
			OS:                     "linux",
			Arch:                   "amd64",
			NexusVersion:           "1.1.0-test",
			RemoteExecutionEnabled: false,
			Capabilities: []protocol.RemoteBridgeCapability{
				protocol.RemoteBridgeCapabilitySpawn,
				protocol.RemoteBridgeCapabilityCWD,
				protocol.RemoteBridgeCapabilityPTY,
				protocol.RemoteBridgeCapabilityResize,
				protocol.RemoteBridgeCapabilityLiveness,
			},
		}, nil
	})

	probe, err := n.ProbeProjectExecutionTarget(context.Background(), project.ID, "devbox")
	if err != nil {
		t.Fatal(err)
	}
	if probe.Ready || probe.RemoteExecutionEnabled || probe.Reason == "" {
		t.Fatalf("reachable but disabled host must not be ready: %+v", probe)
	}
}

func TestProbeProjectExecutionTargetRejectsMachineDriftAndUnknownTarget(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_REMOTE_EXECUTION", "1")

	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	project, err := st.CreateProject(store.Project{Name: "Probe", CanonicalPath: projectDir})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "nexus.yaml"), []byte(`version: 1
execution:
  targets:
    - id: devbox
      kind: ssh
      location: devbox
      machineId: expected-machine
      workspace: /srv/project
`), 0o600); err != nil {
		t.Fatal(err)
	}

	withProjectSSHBridgeOpener(t, func(context.Context, string, string) (projectSSHBridge, protocol.RemoteBridgeHelloData, error) {
		return &fakeProjectSSHBridge{}, protocol.RemoteBridgeHelloData{
			ProtocolVersion:        protocol.ProtocolVersion,
			MachineID:              "different-machine",
			OS:                     "linux",
			Arch:                   "amd64",
			NexusVersion:           "1.1.0-test",
			RemoteExecutionEnabled: true,
			Capabilities: []protocol.RemoteBridgeCapability{
				protocol.RemoteBridgeCapabilitySpawn,
				protocol.RemoteBridgeCapabilityCWD,
				protocol.RemoteBridgeCapabilityPTY,
				protocol.RemoteBridgeCapabilityResize,
				protocol.RemoteBridgeCapabilityLiveness,
			},
		}, nil
	})

	if _, err := n.ProbeProjectExecutionTarget(context.Background(), project.ID, "devbox"); err == nil {
		t.Fatal("machine identity drift must fail closed")
	}
	if _, err := n.ProbeProjectExecutionTarget(context.Background(), project.ID, "missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown target error=%v, want store.ErrNotFound", err)
	}
}


func TestActivateManagedExecutionTargetRejectsStaleApprovalAndPersistsFreshProbe(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_REMOTE_EXECUTION", "1")

	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	project, err := st.CreateProject(store.Project{
		Name:          "Activate target",
		CanonicalPath: projectDir,
		Settings:      `{"appearance":{"density":"compact"}}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "nexus.yaml"), []byte(`version: 1
execution:
  defaultTarget: devbox
  targets:
    - id: devbox
      kind: ssh
      location: ssh-devbox
      machineId: machine-devbox
      workspace: /srv/project
`), 0o600); err != nil {
		t.Fatal(err)
	}

	openCount := 0
	withProjectSSHBridgeOpener(t, func(_ context.Context, targetID, location string) (projectSSHBridge, protocol.RemoteBridgeHelloData, error) {
		openCount++
		if targetID != "devbox" || location != "ssh-devbox" {
			t.Fatalf("unexpected target resolution: %q %q", targetID, location)
		}
		return &fakeProjectSSHBridge{}, protocol.RemoteBridgeHelloData{
			ProtocolVersion:        protocol.ProtocolVersion,
			MachineID:              "machine-devbox",
			OS:                     "linux",
			Arch:                   "amd64",
			NexusVersion:           "1.1.0-test",
			RemoteExecutionEnabled: true,
			Capabilities: []protocol.RemoteBridgeCapability{
				protocol.RemoteBridgeCapabilitySpawn,
				protocol.RemoteBridgeCapabilityCWD,
				protocol.RemoteBridgeCapabilityPTY,
				protocol.RemoteBridgeCapabilityResize,
				protocol.RemoteBridgeCapabilityLiveness,
			},
		}, nil
	})

	probe, err := n.ProbeProjectExecutionTarget(context.Background(), project.ID, "devbox")
	if err != nil {
		t.Fatal(err)
	}
	if probe.ApprovalToken == "" {
		t.Fatal("ready probe must produce approval token")
	}

	if _, err := n.ActivateManagedExecutionTarget(context.Background(), project.ID, "devbox", "stale-token"); !errors.Is(err, ErrExecutionTargetApprovalStale) {
		t.Fatalf("stale approval error=%v", err)
	}
	if binding, err := n.ManagedExecutionTargetBinding(project.ID); err != nil || binding != nil {
		t.Fatalf("stale approval persisted binding=%+v err=%v", binding, err)
	}

	binding, err := n.ActivateManagedExecutionTarget(context.Background(), project.ID, "devbox", probe.ApprovalToken)
	if err != nil {
		t.Fatal(err)
	}
	if binding.TargetID != "devbox" || binding.MachineID != "machine-devbox" || binding.ApprovalToken != probe.ApprovalToken || binding.ActivatedAt.IsZero() {
		t.Fatalf("unexpected binding: %+v", binding)
	}
	persisted, err := n.ManagedExecutionTargetBinding(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted == nil || persisted.TargetID != binding.TargetID || persisted.ApprovalToken != binding.ApprovalToken {
		t.Fatalf("binding was not persisted: %+v", persisted)
	}

	projectAfter, err := st.GetProject(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(projectAfter.Settings, "appearance") || !strings.Contains(projectAfter.Settings, "compact") {
		t.Fatalf("activation replaced unrelated project settings: %s", projectAfter.Settings)
	}

	if err := n.DeactivateManagedExecutionTarget(project.ID); err != nil {
		t.Fatal(err)
	}
	if persisted, err := n.ManagedExecutionTargetBinding(project.ID); err != nil || persisted != nil {
		t.Fatalf("deactivation left binding=%+v err=%v", persisted, err)
	}
	if openCount < 3 {
		t.Fatalf("activation must re-probe instead of trusting old response, bridge opens=%d", openCount)
	}
}


func TestPreviewProjectExecutionTargetsReflectsActiveBindingWithoutNetworkProbe(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_REMOTE_EXECUTION", "1")

	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "nexus.yaml"), []byte("version: 1\nexecution:\n  defaultTarget: devbox\n  targets:\n    - id: devbox\n      kind: ssh\n      location: devbox\n      machineId: machine-devbox\n      workspace: /srv/project\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Bound", CanonicalPath: projectDir})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetProjectSetting(project.ID, managedExecutionTargetSettingKey, ManagedExecutionTargetBinding{
		TargetID:      "devbox",
		ApprovalToken: "approved-fingerprint",
		MachineID:     "machine-devbox",
	}); err != nil {
		t.Fatal(err)
	}

	catalog, err := n.PreviewProjectExecutionTargets(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.EffectiveDefaultTarget != "devbox" || len(catalog.Targets) != 2 {
		t.Fatalf("unexpected bound catalog: %+v", catalog)
	}
	if catalog.Targets[0].Effective {
		t.Fatal("local target must stop being effective when an explicit remote binding exists")
	}
	remote := catalog.Targets[1]
	if !remote.Effective || !remote.Selectable || remote.ActivationStatus != "ACTIVE" {
		t.Fatalf("active binding not reflected: %+v", remote)
	}
}

func TestPreviewProjectExecutionTargetsReportsStaleBinding(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_REMOTE_EXECUTION", "1")

	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "nexus.yaml"), []byte("version: 1\nexecution:\n  defaultTarget: local\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Stale", CanonicalPath: projectDir})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetProjectSetting(project.ID, managedExecutionTargetSettingKey, ManagedExecutionTargetBinding{
		TargetID:      "removed-target",
		ApprovalToken: "old-fingerprint",
		MachineID:     "old-machine",
	}); err != nil {
		t.Fatal(err)
	}

	catalog, err := n.PreviewProjectExecutionTargets(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.EffectiveDefaultTarget != "removed-target" {
		t.Fatalf("stale binding identity was hidden: %+v", catalog)
	}
	if len(catalog.Targets) != 2 {
		t.Fatalf("expected local + stale binding row, got %+v", catalog.Targets)
	}
	stale := catalog.Targets[1]
	if !stale.Effective || stale.Selectable || stale.ActivationStatus != "STALE_BINDING" {
		t.Fatalf("stale binding must be visible and non-selectable: %+v", stale)
	}
}
