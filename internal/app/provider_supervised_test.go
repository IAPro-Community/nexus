package app

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/nexus"
)

func TestProviderLaunchFlagsAreNexusOwned(t *testing.T) {
	args := []string{"--yolo", "--supervised", "--model", "test"}
	if !hasProviderLaunchFlag(args, "--supervised") {
		t.Fatal("expected supervised flag to be recognized")
	}
	got := removeProviderLaunchFlag(args, "--supervised")
	if len(got) != 3 || got[0] != "--yolo" || got[1] != "--model" || got[2] != "test" {
		t.Fatalf("provider args = %#v", got)
	}
	if hasProviderLaunchFlag(got, "--supervised") {
		t.Fatal("Nexus-owned flag leaked to provider args")
	}
}

func TestWantInteractiveLeadDefaultsToStickyForTTY(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{StdinIsTTY: true, Provider: "codex"})
	if !r.WantInteractiveLead() {
		t.Fatal("interactive supervised TTY must bind STICKY_LEAD by default")
	}
	if r.RequireInteractiveLead() {
		t.Fatal("implicit STICKY_LEAD must be best-effort, not required")
	}
}

func TestWantInteractiveLeadControls(t *testing.T) {
	explicit := NewLaunchModeResolver(LaunchModeInput{Args: []string{"--lead"}, StdinIsTTY: true})
	if !explicit.WantInteractiveLead() || !explicit.RequireInteractiveLead() {
		t.Fatal("explicit --lead must enable and require persistent Lead identity")
	}

	disabled := NewLaunchModeResolver(LaunchModeInput{Args: []string{"--no-lead"}, StdinIsTTY: true})
	if disabled.WantInteractiveLead() || disabled.RequireInteractiveLead() {
		t.Fatal("--no-lead must disable STICKY_LEAD")
	}

	both := NewLaunchModeResolver(LaunchModeInput{Args: []string{"--lead", "--no-lead"}, StdinIsTTY: true})
	if both.WantInteractiveLead() || both.RequireInteractiveLead() {
		t.Fatal("--no-lead must win over --lead")
	}

	headless := NewLaunchModeResolver(LaunchModeInput{StdinIsTTY: false})
	if headless.WantInteractiveLead() {
		t.Fatal("non-TTY supervised launches must not gain an implicit Lead")
	}
}

func TestSupervisedRuntimeLabelsDoNotAuthorizeMissionPromotion(t *testing.T) {
	withoutLead := supervisedRuntimeLabels()
	if got := withoutLead["nexus.delegation_mode"]; got != string(nexus.DelegationOff) {
		t.Fatalf("delegation mode without Lead = %q, want %q", got, nexus.DelegationOff)
	}
	if got := withoutLead["nexus.interactive_lead"]; got != "false" {
		t.Fatalf("interactive Lead without opt-in = %q, want false", got)
	}

	// Lead=true is persisted only by UpdateInteractiveLeadBinding after the
	// binding succeeds; launch labels intentionally start false.
	if got := withoutLead["nexus.delegation_mode"]; got != string(nexus.DelegationOff) {
		t.Fatalf("sticky Lead launch must keep delegation OFF: got %q", got)
	}
}

func TestStripControlFlagsRemovesLeadControls(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"--lead", "--yolo", "--model", "test"},
		StdinIsTTY: true,
	})
	if !r.WantInteractiveLead() || !r.RequireInteractiveLead() {
		t.Fatal("explicit --lead must be required before strip")
	}
	got := r.StripControlFlags()
	for _, arg := range got {
		if arg == "--lead" || arg == "--no-lead" || arg == "--direct" || arg == "--supervised" {
			t.Fatalf("Nexus-owned flag leaked: %#v", got)
		}
	}
	if len(got) != 3 || got[0] != "--yolo" || got[1] != "--model" || got[2] != "test" {
		t.Fatalf("StripControlFlags = %#v", got)
	}
}

func TestProfileInCandidates(t *testing.T) {
	candidates := []model.Profile{{Name: "agy1"}, {Name: "agy2"}}
	if !profileInCandidates(candidates, "agy2") {
		t.Fatal("expected configured profile")
	}
	if profileInCandidates(candidates, "codex1") {
		t.Fatal("unexpected profile match")
	}
}

func TestLaunchableProfilesAllowsBusyCodexWhenAutoSelecting(t *testing.T) {
	got, err := launchableProfiles("codex", "", []model.Profile{
		{Name: "busy"},
		{Name: "free"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "busy" || got[1].Name != "free" {
		t.Fatalf("got %#v want all configured profiles", got)
	}
}

func TestLaunchableProfilesExplicitBusyRemainsLaunchable(t *testing.T) {
	got, err := launchableProfiles("codex", "busy", []model.Profile{
		{Name: "busy"},
		{Name: "free"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "busy" {
		t.Fatalf("got %#v", got)
	}
}

func TestStripControlFlagsRemovesNoLead(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"--no-lead", "--yolo"},
		StdinIsTTY: true,
	})
	if r.WantInteractiveLead() {
		t.Fatal("--no-lead must disable Interactive Lead before stripping")
	}
	got := r.StripControlFlags()
	if len(got) != 1 || got[0] != "--yolo" {
		t.Fatalf("StripControlFlags = %#v", got)
	}
}
