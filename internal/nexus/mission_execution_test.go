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
	want := []string{"--ask-for-approval", "never", "--sandbox", "workspace-write", "-c", "sandbox_workspace_write.network_access=false", "exec", "write the validation note"}
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

func TestMissionProviderArgsOpenCodeFailsClosedWithoutVerifiedIsolation(t *testing.T) {
	_, err := missionProviderArgs("opencode", []string{"run", "edit the file"}, agentPromptPolicy{
		Contract: runner.DefaultAutonomyContract(),
	})
	if err == nil {
		t.Fatal("expected restrictive default contract to reject unverified OpenCode isolation")
	}
}

func TestMissionProviderArgsOpenCodeAllowsExplicitUnrestrictedContract(t *testing.T) {
	contract := runner.DefaultAutonomyContract()
	contract.DisallowDestructiveGit = false
	contract.AllowGitPush = true
	contract.AllowDeploy = true
	contract.AllowExternalNetwork = true
	contract.AllowSecretAccess = true
	contract.AllowPaidServices = true
	got, err := missionProviderArgs("opencode", []string{"run", "edit the file"}, agentPromptPolicy{Contract: contract})
	if err != nil {
		t.Fatalf("explicit unrestricted contract should allow provider-native auto mode: %v", err)
	}
	want := []string{"run", "--auto", "edit the file"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("opencode unrestricted args = %#v, want %#v", got, want)
	}
}

func TestMissionProviderArgsOpenCodeReviewFailsClosedWithoutVerifiedIsolation(t *testing.T) {
	_, err := missionProviderArgs("opencode", []string{"run", "review the code"}, agentPromptPolicy{
		Contract: runner.DefaultAutonomyContract(),
		Review:   true,
	})
	if err == nil {
		t.Fatal("expected OpenCode review to fail without verified read-only isolation")
	}
}

func TestMissionProviderArgsGeminiFailsClosedWithoutCrossPlatformIsolationProof(t *testing.T) {
	_, err := missionProviderArgs("gemini", []string{"-p", "edit safely"}, agentPromptPolicy{
		Contract: runner.DefaultAutonomyContract(),
	})
	if err == nil {
		t.Fatal("expected restrictive default contract to reject unverified Gemini isolation")
	}
}
