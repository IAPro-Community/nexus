package app

import (
	"strings"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/core/model"
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

func TestProfileInCandidates(t *testing.T) {
	candidates := []model.Profile{{Name: "agy1"}, {Name: "agy2"}}
	if !profileInCandidates(candidates, "agy2") {
		t.Fatal("expected configured profile")
	}
	if profileInCandidates(candidates, "codex1") {
		t.Fatal("unexpected profile match")
	}
}

func TestLaunchableProfilesSkipsBusyCodexWhenAutoSelecting(t *testing.T) {
	orig := profileLaunchBusy
	t.Cleanup(func() { profileLaunchBusy = orig })
	profileLaunchBusy = func(provider, name string) bool {
		return provider == "codex" && name == "busy"
	}

	got, err := launchableProfiles("codex", "", []model.Profile{
		{Name: "busy"},
		{Name: "free"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "free" {
		t.Fatalf("got %#v want only free", got)
	}
}

func TestLaunchableProfilesExplicitBusyFailsFast(t *testing.T) {
	orig := profileLaunchBusy
	t.Cleanup(func() { profileLaunchBusy = orig })
	profileLaunchBusy = func(provider, name string) bool {
		return provider == "codex" && name == "busy"
	}

	_, err := launchableProfiles("codex", "busy", []model.Profile{
		{Name: "busy"},
		{Name: "free"},
	})
	if err == nil {
		t.Fatal("expected busy explicit profile to fail fast")
	}
	if !strings.Contains(err.Error(), "busy") {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(err.Error(), "attach") || !strings.Contains(err.Error(), "stop") {
		t.Fatalf("busy error must tell the user how to attach/stop: %v", err)
	}
}

func TestBusyLaunchErrorMentionsAttachAndStop(t *testing.T) {
	err := busyLaunchError("codex", "work")
	msg := err.Error()
	for _, want := range []string{"codex", "work", "attach", "stop", "ps"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("busyLaunchError missing %q: %s", want, msg)
		}
	}
}
