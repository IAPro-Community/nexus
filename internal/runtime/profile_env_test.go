package runtime

import (
	"path/filepath"
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
