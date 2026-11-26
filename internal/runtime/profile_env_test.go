package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyProfileIsolationEnvConfinesXDGAndPreservesRealHome(t *testing.T) {
	realHome := filepath.Join(t.TempDir(), "host-home")
	t.Setenv("NEXUS_REAL_HOME", realHome)
	t.Setenv("GOTMPDIR", filepath.Join(realHome, ".cache", "go-tmp"))

	profileHome := filepath.Join(realHome, ".local", "share", "nexus", "profiles", "claude", "work", "home")
	env := ApplyProfileIsolationEnv(profileHome, map[string]string{
		"HOME":              profileHome,
		"CLAUDE_CONFIG_DIR": filepath.Join(profileHome, ".claude"),
	})

	checks := map[string]string{
		"HOME":              profileHome,
		"USERPROFILE":       profileHome,
		"XDG_CONFIG_HOME":   filepath.Join(profileHome, ".config"),
		"XDG_CACHE_HOME":    filepath.Join(profileHome, ".cache"),
		"XDG_DATA_HOME":     filepath.Join(profileHome, ".local", "share"),
		"XDG_STATE_HOME":    filepath.Join(profileHome, ".local", "state"),
		"GOCACHE":           filepath.Join(profileHome, ".cache", "go-build"),
		"GOTMPDIR":          filepath.Join(profileHome, ".cache", "go-tmp"),
		"NPM_CONFIG_CACHE":  filepath.Join(profileHome, ".cache", "npm"),
		"NEXUS_REAL_HOME":   realHome,
		"CLAUDE_CONFIG_DIR": filepath.Join(profileHome, ".claude"),
	}
	for k, want := range checks {
		if got := env[k]; got != want {
			t.Fatalf("%s = %q, want %q", k, got, want)
		}
	}
	if _, hasGOPATH := env["GOPATH"]; hasGOPATH {
		t.Fatalf("GOPATH should remain inherited from host, got override %q", env["GOPATH"])
	}
}

func TestEnhancedPATHPrioritizesRealHomeLocalBinOverProfileHome(t *testing.T) {
	realHome := filepath.Join(t.TempDir(), "host-home")
	profileHome := filepath.Join(realHome, ".local", "share", "ai-manager", "profiles", "codex", "work", "home")
	wantReal := filepath.Join(realHome, ".local", "bin")
	wantProfile := filepath.Join(profileHome, ".local", "bin")
	if err := os.MkdirAll(wantReal, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", wantReal, err)
	}
	if err := os.MkdirAll(wantProfile, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", wantProfile, err)
	}

	t.Setenv("NEXUS_REAL_HOME", realHome)
	t.Setenv("HOME", profileHome)
	t.Setenv("PATH", "/usr/bin:/bin")

	pathVal := EnhancedPATH()
	parts := strings.Split(pathVal, string(os.PathListSeparator))

	realIdx, profileIdx := -1, -1
	for i, part := range parts {
		if part == wantReal && realIdx == -1 {
			realIdx = i
		}
		if part == wantProfile && profileIdx == -1 {
			profileIdx = i
		}
	}
	if realIdx == -1 || profileIdx == -1 {
		t.Fatalf("expected both %q (idx=%d) and %q (idx=%d) in PATH %q", wantReal, realIdx, wantProfile, profileIdx, pathVal)
	}
	if realIdx >= profileIdx {
		t.Fatalf("expected real host %q (idx=%d) strictly before profile %q (idx=%d) in PATH %q", wantReal, realIdx, wantProfile, profileIdx, pathVal)
	}
}
