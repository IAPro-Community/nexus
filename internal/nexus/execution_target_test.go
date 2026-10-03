package nexus

import (
	"os"
	"path/filepath"
	"testing"

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
	n := newIsolatedNexus(t)
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

	n := newIsolatedNexus(t)
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
      commandTemplate: ssh -tt devbox {provider} {args}
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

	n := newIsolatedNexus(t)
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
      commandTemplate: ssh devbox {provider} {args}
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
	if remote.Selectable || remote.Effective || remote.ActivationStatus != "PREVIEW_ONLY" {
		t.Fatalf("SSH must remain preview-only until approval exists: %+v", remote)
	}
}
