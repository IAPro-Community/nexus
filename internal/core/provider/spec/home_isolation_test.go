package spec

import (
	"testing"
)

// Every provider spec must isolate both Unix (HOME) and Windows (USERPROFILE)
// home variables to the profile directory. A missing USERPROFILE lets a
// profiled launch on Windows read/write the real user profile, breaking
// hermetic auth fixtures and leaking credentials across profiles.
func TestAllSpecsIsolateHomeOnUnixAndWindows(t *testing.T) {
	for _, s := range All() {
		t.Run(s.ID, func(t *testing.T) {
			env := s.EnvKeys("/tmp/profile-home", "/tmp/bin", "test")
			if got := env["HOME"]; got != "/tmp/profile-home" {
				t.Fatalf("HOME = %q, want isolated profile home", got)
			}
			if got := env["USERPROFILE"]; got != "/tmp/profile-home" {
				t.Fatalf("USERPROFILE = %q, want isolated profile home", got)
			}
		})
	}
}
