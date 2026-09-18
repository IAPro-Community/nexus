package launcher

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
)

func TestClassifyDetachedHostFailureMapsCodexTUIBusy(t *testing.T) {
	t.Setenv("AI_MANAGER_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_DATA_DIR", os.Getenv("AI_MANAGER_DATA_DIR"))

	runtimeID := "codex-busy-host"
	path := HostLogPath(runtimeID)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := "nexus: failed to start SessionHost: failed to acquire Codex TUI lease: codex profile lock is busy\n"
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("write log: %v", err)
	}

	err := classifyDetachedHostFailure(runtimeID, 4242)
	if !errors.Is(err, codex.ErrTUIBusy) {
		t.Fatalf("err=%v want ErrTUIBusy", err)
	}
	if !strings.Contains(err.Error(), "4242") || !strings.Contains(err.Error(), "codex profile lock is busy") {
		t.Fatalf("unexpected error text: %q", err.Error())
	}
}

func TestClassifyDetachedHostFailureWithoutLog(t *testing.T) {
	t.Setenv("AI_MANAGER_DATA_DIR", t.TempDir())
	err := classifyDetachedHostFailure("missing-runtime", 7)
	if errors.Is(err, codex.ErrTUIBusy) {
		t.Fatalf("unexpected ErrTUIBusy: %v", err)
	}
	if got := err.Error(); got != "detached SessionHost exited before handshake (pid 7)" {
		t.Fatalf("got %q", got)
	}
}
