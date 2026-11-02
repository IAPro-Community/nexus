//go:build windows

package nexus

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if strings.EqualFold(filepath.Base(os.Args[0]), "codex.exe") {
		os.Exit(runWindowsCodexFixture())
	}
	os.Exit(m.Run())
}

func installCodexTestFixture(t *testing.T, binDir string) {
	installWindowsCodexFixture(t, filepath.Join(binDir, "codex.exe"))
}

func installCodexSkillSessionFixture(t *testing.T, binDir string) {
	installWindowsCodexFixture(t, filepath.Join(binDir, "codex.exe"))
}

func installWindowsCodexFixture(t *testing.T, destination string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("locate test executable for isolated Codex fixture: %v", err)
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		t.Fatalf("read test executable for isolated Codex fixture: %v", err)
	}
	if err := os.WriteFile(destination, binary, 0o700); err != nil {
		t.Fatalf("write isolated Codex fixture executable: %v", err)
	}
}

func runWindowsCodexFixture() int {
	if os.Getenv("NEXUS_SKILL_FIXTURE_CAPTURE") == "" {
		if len(os.Args) > 1 && os.Args[1] == "--version" {
			fmt.Println("codex fixture 1.0")
		}
		return 0
	}
	var capture bytes.Buffer
	for _, arg := range os.Args[1:] {
		capture.WriteString(arg)
		capture.WriteByte(0)
	}
	if err := os.WriteFile(os.Getenv("NEXUS_SKILL_FIXTURE_CAPTURE"), capture.Bytes(), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	environment := fmt.Sprintf("HOME=%s\nCODEX_HOME=%s\n", os.Getenv("HOME"), os.Getenv("CODEX_HOME"))
	if err := os.WriteFile(os.Getenv("NEXUS_SKILL_FIXTURE_ENV"), []byte(environment), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("FIXTURE_SKILL_SESSION_COMPLETED")
	time.Sleep(time.Second)
	return 0
}
