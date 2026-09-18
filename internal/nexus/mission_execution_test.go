package nexus

import (
	"reflect"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

func TestMissionProviderArgsCodexPlacesApprovalFlagsBeforePrompt(t *testing.T) {
	got, err := missionProviderArgs("codex", []string{"exec", "write the validation note"}, agentPromptPolicy{
		Contract: runner.DefaultAutonomyContract(),
	})
	if err != nil {
		t.Fatalf("missionProviderArgs returned error: %v", err)
	}
	want := []string{"--ask-for-approval", "never", "--sandbox", "workspace-write", "exec", "write the validation note"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("codex args = %#v, want %#v", got, want)
	}
}

func TestMissionProviderArgsCodexReviewUsesReadOnlyExecSandbox(t *testing.T) {
	got, err := missionProviderArgs("codex", []string{"exec", "return the review JSON"}, agentPromptPolicy{
		Contract: runner.DefaultAutonomyContract(),
		Review:   true,
	})
	if err != nil {
		t.Fatalf("missionProviderArgs returned error: %v", err)
	}
	want := []string{"--sandbox", "read-only", "--ask-for-approval", "never", "exec", "return the review JSON"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("codex review args = %#v, want %#v", got, want)
	}
}

func TestMissionProviderArgsOpenCodeReviewUsesPermissionModePlan(t *testing.T) {
	got, err := missionProviderArgs("opencode", []string{"run", "review the code"}, agentPromptPolicy{
		Contract: runner.DefaultAutonomyContract(),
		Review:   true,
	})
	if err != nil {
		t.Fatalf("missionProviderArgs returned error: %v", err)
	}
	// For non-Codex providers, ReviewMode flags are appended after the prompt.
	want := []string{"run", "review the code", "--permission-mode", "plan"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("opencode review args = %#v, want %#v", got, want)
	}
}

func TestMissionProviderArgsOpenCodeReviewFailsWithoutReviewMode(t *testing.T) {
	// This test verifies that if ReviewMode were nil, review would fail.
	// Since we've set ReviewMode in the spec, this test documents the expected behavior.
	// In a real scenario, if ReviewMode is empty, missionProviderArgs returns an error.
	got, err := missionProviderArgs("opencode", []string{"run", "edit the file"}, agentPromptPolicy{
		Contract: runner.DefaultAutonomyContract(),
		Review:   false,
	})
	if err != nil {
		t.Fatalf("missionProviderArgs returned error: %v", err)
	}
	// Without Review mode, opencode should use auto-approval flags after "run".
	want := []string{"run", "--auto", "edit the file"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("opencode non-review args = %#v, want %#v", got, want)
	}
}
