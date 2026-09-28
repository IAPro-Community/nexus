package spec

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func longCodexProfileHome(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), strings.Repeat("profile-segment-", 6), "home")
}

func TestCodexInteractiveArgsAddsNoDaemonForUnsafeUnixSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SUN_LEN fallback is Unix-only")
	}
	home := longCodexProfileHome(t)
	got := CodexInteractiveArgs(home, []string{"--yolo"})
	if len(got) != 2 || got[0] != "--no-daemon" || got[1] != "--yolo" {
		t.Fatalf("CodexInteractiveArgs() = %#v, want --no-daemon before original args", got)
	}
}

func TestCodexInteractiveArgsPreservesResumeAndExistingNoDaemon(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SUN_LEN fallback is Unix-only")
	}
	home := longCodexProfileHome(t)

	resume := CodexInteractiveArgs(home, []string{"resume", "session-id"})
	if len(resume) != 3 || resume[0] != "--no-daemon" || resume[1] != "resume" {
		t.Fatalf("resume args = %#v", resume)
	}

	existing := CodexInteractiveArgs(home, []string{"--no-daemon", "--yolo"})
	if len(existing) != 2 || existing[0] != "--no-daemon" {
		t.Fatalf("existing --no-daemon must not be duplicated: %#v", existing)
	}
}

func TestCodexInteractiveArgsDoesNotAlterHeadlessSubcommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SUN_LEN fallback is Unix-only")
	}
	home := longCodexProfileHome(t)
	for _, args := range [][]string{
		{"exec", "fix the API"},
		{"login"},
		{"app-server", "--stdio"},
		{"agents"},
	} {
		got := CodexInteractiveArgs(home, args)
		if len(got) != len(args) || (len(got) > 0 && got[0] == "--no-daemon") {
			t.Fatalf("headless/service args %#v were changed to %#v", args, got)
		}
	}
}

func TestCodexInteractiveArgsLeavesSafeSocketPathAlone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SUN_LEN fallback is Unix-only")
	}
	got := CodexInteractiveArgs("/tmp/nx-codex", []string{"--yolo"})
	if len(got) != 1 || got[0] != "--yolo" {
		t.Fatalf("safe path args changed: %#v", got)
	}
}
