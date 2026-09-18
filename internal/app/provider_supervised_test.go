package app

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
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

func TestWantInteractiveLeadDefaultFalse(t *testing.T) {
	if wantInteractiveLead(nil) {
		t.Fatal("default supervised shortcut must not opt into Lead")
	}
	if wantInteractiveLead([]string{"--yolo", "--model", "gpt-5"}) {
		t.Fatal("provider flags alone must not enable Lead")
	}
}

func TestWantInteractiveLeadOptIn(t *testing.T) {
	if !wantInteractiveLead([]string{"--lead"}) {
		t.Fatal("expected --lead to opt into Interactive Lead")
	}
	if !wantInteractiveLead([]string{"--yolo", "--Lead", "--model", "x"}) {
		t.Fatal("expected case-insensitive --lead")
	}
}

func TestStripControlFlagsRemovesLead(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"--lead", "--yolo", "--model", "test"},
		StdinIsTTY: true,
	})
	if !r.WantInteractiveLead() {
		t.Fatal("WantInteractiveLead must see --lead before strip")
	}
	got := r.StripControlFlags()
	for _, arg := range got {
		if arg == "--lead" || arg == "--direct" || arg == "--supervised" {
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
