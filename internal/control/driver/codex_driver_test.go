package driver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/testutil"
)

func TestCodexDriverBuildCommandIsolatesHome(t *testing.T) {
	data := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	hostHome := t.TempDir()
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)

	// Fake codex on PATH so LookPath succeeds.
	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.WriteFakeBinary(t, binDir, "codex", "")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	d := &CodexDriver{}
	_, _, env, err := d.BuildCommand(context.Background(), model.Profile{Name: "work", Provider: "codex"}, nil)
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}

	wantHome, err := config.ProfileHome("codex", "work")
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]string{}
	for _, entry := range env {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			got[parts[0]] = parts[1]
		}
	}
	if got["HOME"] != wantHome {
		t.Fatalf("HOME=%q, want profile home %q", got["HOME"], wantHome)
	}
	if got["CODEX_HOME"] != wantHome {
		t.Fatalf("CODEX_HOME=%q, want profile home %q", got["CODEX_HOME"], wantHome)
	}
	if got["CODEX_TUI_DISABLE_KEYBOARD_ENHANCEMENT"] != "1" {
		t.Fatalf("CODEX_TUI_DISABLE_KEYBOARD_ENHANCEMENT=%q, want 1", got["CODEX_TUI_DISABLE_KEYBOARD_ENHANCEMENT"])
	}
	if got["HOME"] == hostHome || strings.HasPrefix(got["CODEX_HOME"], filepath.Join(hostHome, ".codex")) {
		t.Fatalf("Codex env leaked host home: HOME=%q CODEX_HOME=%q host=%q", got["HOME"], got["CODEX_HOME"], hostHome)
	}
}
