package runner

import (
	"reflect"
	"testing"
)

func boolPtr(v bool) *bool { return &v }
func intPtr(v int) *int    { return &v }

func TestApplyAutonomyContractPatchPreservesSafeDefaultsAndExplicitValues(t *testing.T) {
	patch := &AutonomyContractPatch{
		MaxRetries:          intPtr(5),
		RequireVerification: boolPtr(false),
		AllowGitPush:        boolPtr(true),
	}

	got := ApplyAutonomyContractPatch(patch)
	defaults := DefaultAutonomyContract()

	if got.MaxRetries != 5 {
		t.Fatalf("expected explicit max retries 5, got %d", got.MaxRetries)
	}
	if got.MaxTotalIterations != defaults.MaxTotalIterations {
		t.Fatalf("expected omitted max total iterations to inherit %d, got %d", defaults.MaxTotalIterations, got.MaxTotalIterations)
	}
	if got.RequireVerification {
		t.Fatal("explicit false require_verification must be preserved")
	}
	if !got.DisallowDestructiveGit {
		t.Fatal("omitted disallow_destructive_git must inherit safe true default")
	}
	if !got.AllowGitPush {
		t.Fatal("explicit allow_git_push=true must be preserved")
	}
	if got.AllowDeploy || got.AllowExternalNetwork || got.AllowSecretAccess || got.AllowPaidServices {
		t.Fatal("dangerous permissions omitted from patch must remain denied")
	}
	if got.AllowComputerUse || got.AllowScreenCapture || got.AllowKeyboardInput || got.AllowPointerInput || got.AllowWindowManagement {
		t.Fatal("computer-control permissions omitted from patch must remain denied")
	}
}

func TestApplyAutonomyContractPatchNilReturnsDefaults(t *testing.T) {
	got := ApplyAutonomyContractPatch(nil)
	want := DefaultAutonomyContract()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("nil patch should return defaults: got %#v want %#v", got, want)
	}
}

func TestApplyAutonomyContractPatchComputerUseIsExplicit(t *testing.T) {
	patch := &AutonomyContractPatch{
		AllowComputerUse:      boolPtr(true),
		AllowScreenCapture:    boolPtr(true),
		AllowKeyboardInput:    boolPtr(false),
		AllowPointerInput:     boolPtr(false),
		AllowWindowManagement: boolPtr(false),
	}
	got := ApplyAutonomyContractPatch(patch)
	if !got.AllowComputerUse || !got.AllowScreenCapture {
		t.Fatal("explicit observation permissions must be preserved")
	}
	if got.AllowKeyboardInput || got.AllowPointerInput || got.AllowWindowManagement {
		t.Fatal("input/window control must not be implied by computer-use observation")
	}
}
