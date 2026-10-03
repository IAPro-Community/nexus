package nexus

import (
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
