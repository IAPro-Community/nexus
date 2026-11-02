//go:build !windows

package nexus

import (
	"os"
	"path/filepath"
	"testing"
)

func installCodexTestFixture(t *testing.T, binDir string) {
	t.Helper()
	path := filepath.Join(binDir, "codex")
	content := "#!/bin/sh\nif [ \"${1:-}\" = \"--version\" ]; then printf '%s\\n' 'codex fixture 1.0'; exit 0; fi\nexit 0\n"
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatalf("write isolated Codex fixture: %v", err)
	}
}

func installCodexSkillSessionFixture(t *testing.T, binDir string) {
	t.Helper()
	path := filepath.Join(binDir, "codex")
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then printf '%s\\n' 'codex fixture 1.0'; exit 0; fi\n" +
		"printf '%s\\0' \"$@\" > \"$NEXUS_SKILL_FIXTURE_CAPTURE\"\n" +
		"printf 'HOME=%s\\nCODEX_HOME=%s\\n' \"$HOME\" \"$CODEX_HOME\" > \"$NEXUS_SKILL_FIXTURE_ENV\"\n" +
		"printf '%s\\n' 'FIXTURE_SKILL_SESSION_COMPLETED'\n" +
		"sleep 1\n"
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatalf("write isolated Codex skill fixture: %v", err)
	}
}
