package runtime

import (
	"os"
	"strings"
	"testing"
)

func TestTerminalTraceShapeMasksPromptTextAndKeepsControlStructure(t *testing.T) {
	got := TerminalTraceShape([]byte("secret\x1b[A123\r\x03"))
	if strings.Contains(got, "secret") || strings.Contains(got, "123") {
		t.Fatalf("trace leaked prompt content: %q", got)
	}
	if !strings.Contains(got, "<1b>[A") {
		t.Fatalf("trace lost CSI structure: %q", got)
	}
	if !strings.Contains(got, "<0d>") || !strings.Contains(got, "<03>") {
		t.Fatalf("trace lost control bytes: %q", got)
	}
}

func TestTraceTerminalBytesWritesPrivateJSONLOnlyWhenEnabled(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("NEXUS_TERMINAL_TRACE", "1")

	TraceTerminalBytes("rt/test", "host.raw input", []byte("hello\x1b[C"))
	path := TerminalTracePath("rt/test")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("trace file missing: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("trace permissions too broad: %o", info.Mode().Perm())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Contains(text, "hello") {
		t.Fatalf("trace file leaked raw prompt: %s", text)
	}
	if !strings.Contains(text, "host.raw_input") || !strings.Contains(text, "<1b>[C") {
		t.Fatalf("trace record missing diagnostic structure: %s", text)
	}
}

func TestTraceTerminalBytesIsDisabledByDefault(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("NEXUS_TERMINAL_TRACE", "")

	TraceTerminalBytes("rt-disabled", "stage", []byte("\x1b[A"))
	if _, err := os.Stat(TerminalTracePath("rt-disabled")); !os.IsNotExist(err) {
		t.Fatalf("trace file should not exist when disabled, err=%v", err)
	}
}
