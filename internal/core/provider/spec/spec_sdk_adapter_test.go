package spec

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNonAGYSpecEnvironmentDoesNotCreateBrowserHelper(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)

	codex := MustGet("codex")
	env := codex.EnvKeys(t.TempDir(), "", "test-profile")
	if env["BROWSER"] != "" {
		t.Fatalf("Codex environment unexpectedly configures a browser helper: %q", env["BROWSER"])
	}
	if _, err := os.Stat(filepath.Join(dataDir, "runtime", "bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("resolving Codex env created the shared browser helper directory: %v", err)
	}
}
