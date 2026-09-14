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
