package spec

import (
	"path/filepath"
	"strings"
	"testing"
)

func envToMap(env []string) map[string]string {
	out := make(map[string]string, len(env))
	for _, item := range env {
		if k, v, ok := strings.Cut(item, "="); ok {
			out[k] = v
		}
	}
	return out
}

func TestOverlayEnvSharesToolchainCachesWhileIsolatingCredentials(t *testing.T) {
	hostHome := t.TempDir()
	t.Setenv("NEXUS_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)

	profileHome := filepath.Join(hostHome, ".local", "share", "ai-manager", "profiles", "codex", "work", "home")
	s := MustGet("codex")
	envMap := envToMap(s.OverlayEnv([]string{"PATH=/usr/bin"}, profileHome, "/tmp/bin", "work"))

	if got := envMap["HOME"]; got != profileHome {
		t.Fatalf("HOME = %q, want isolated %q", got, profileHome)
	}
	if got := envMap["XDG_CONFIG_HOME"]; got != filepath.Join(profileHome, ".config") {
		t.Fatalf("XDG_CONFIG_HOME = %q, want isolated config dir", got)
	}
	wantShared := filepath.Join(hostHome, ".cache", "nexus-toolchains", "go-build")
	if got := envMap["GOCACHE"]; got != wantShared {
		t.Fatalf("GOCACHE = %q, want shared %q", got, wantShared)
	}

	t.Setenv("NEXUS_ISOLATE_BUILD_CACHES", "1")
	isolatedMap := envToMap(s.OverlayEnv([]string{"PATH=/usr/bin"}, profileHome, "/tmp/bin", "work"))
	wantIsolated := filepath.Join(profileHome, ".cache", "go-build")
	if got := isolatedMap["GOCACHE"]; got != wantIsolated {
		t.Fatalf("with NEXUS_ISOLATE_BUILD_CACHES=1 GOCACHE = %q, want %q", got, wantIsolated)
	}
}

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
