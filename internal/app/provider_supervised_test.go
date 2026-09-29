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

func TestWantInteractiveLeadDefaultsToAGYOnly(t *testing.T) {
	if !wantInteractiveLead("agy", nil) {
		t.Fatal("AGY supervised sessions must bind the orchestration Lead by default")
	}
	if !wantInteractiveLead("agy", []string{"--yolo", "--model", "gemini"}) {
		t.Fatal("AGY provider flags must preserve the default Interactive Lead")
	}
	if wantInteractiveLead("codex", []string{"--yolo"}) {
		t.Fatal("Codex must remain byte-transparent by default")
	}
	if wantInteractiveLead("opencode", nil) {
		t.Fatal("unverified provider TUIs must not become interception defaults")
	}
}

func TestWantInteractiveLeadExplicitOptInAndNoLeadWins(t *testing.T) {
	if !wantInteractiveLead("codex", []string{"--lead"}) {
		t.Fatal("explicit --lead must opt a provider into Interactive Lead")
	}
	if !wantInteractiveLead("claude", []string{"--yolo", "--LEAD", "--model", "x"}) {
		t.Fatal("expected case-insensitive --lead compatibility")
	}
	if wantInteractiveLead("agy", []string{"--lead", "--no-lead"}) {
		t.Fatal("--no-lead must win when both compatibility flags are present")
	}
	if wantInteractiveLead("agy", []string{"--no-lead"}) {
		t.Fatal("expected --no-lead to keep provider-native behavior")
	}
}

func TestSupervisedRuntimeLabelsAuthorizeOnlyBoundLead(t *testing.T) {
	withoutLead := supervisedRuntimeLabels(false)
	if got := withoutLead["nexus.delegation_mode"]; got != string(nexus.DelegationOff) {
		t.Fatalf("delegation mode without Lead = %q, want %q", got, nexus.DelegationOff)
	}
	if got := withoutLead["nexus.interactive_lead"]; got != "false" {
		t.Fatalf("interactive Lead without opt-in = %q, want false", got)
	}

	withLead := supervisedRuntimeLabels(true)
	if got := withLead["nexus.interactive_lead"]; got != "true" {
		t.Fatalf("interactive Lead = %q, want true", got)
	}
	if got := withLead["nexus.delegation_mode"]; got != string(nexus.DelegationAuto) {
		t.Fatalf("bound Lead must enter AUTO delegation: got %q, want %q", got, nexus.DelegationAuto)
	}
}

func TestStripControlFlagsRemovesLeadControls(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"--lead", "--yolo", "--model", "test"},
		StdinIsTTY: true,
	})
	if !r.WantInteractiveLead() {
		t.Fatal("WantInteractiveLead must see --lead before strip")
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
