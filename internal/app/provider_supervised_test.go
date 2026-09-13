package app

import (
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
