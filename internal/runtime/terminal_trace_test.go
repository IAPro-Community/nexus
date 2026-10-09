package runtime

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTerminalTraceShapeDoesNotLeakAnsiLikePlainText(t *testing.T) {
	got := TerminalTraceShape([]byte("ARM COMMAND MARCH"))
	if strings.Contains(got, "ARM") || strings.Contains(got, "COMMAND") || strings.Contains(got, "MARCH") {
		t.Fatalf("trace leaked ANSI-like plain text: %q", got)
	}
	if strings.ContainsAny(got, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		t.Fatalf("trace preserved ordinary uppercase text outside a terminal sequence: %q", got)
	}
}

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
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
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
	if !strings.Contains(text, "host.raw_input") || (!strings.Contains(text, "<1b>[C") && !strings.Contains(text, `\u003c1b\u003e[C`)) {
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

func TestClientTerminalTraceShapeRejectsUnsanitizedFrames(t *testing.T) {
	for _, shape := range []string{
		"password", "ARM COMMAND MARCH", "<1b>[A password", "A", "<1b>[A\nprivate", "<41><50><49>",
	} {
		if validClientTerminalTraceShape(shape) {
			t.Fatalf("accepted unsanitized client shape %q", shape)
		}
	}
	for _, shape := range []string{
		"<1b>[A", "[<##;##;##M", "<1b>", "<03>", "....", "<1b>O", "<1b>].....<07>",
	} {
		if !validClientTerminalTraceShape(shape) {
			t.Fatalf("rejected legitimate client trace shape %q", shape)
		}
	}
}

func TestClientTerminalTraceShapeIsValidatedBeforePersistence(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("NEXUS_TERMINAL_TRACE", "1")
	TraceTerminalShape("rt-untrusted", "web.xterm.ondata", "API_KEY_IS_SECRET", 17)
	if _, err := os.Stat(TerminalTracePath("rt-untrusted")); !os.IsNotExist(err) {
		t.Fatalf("untrusted raw text must never be stored, err=%v", err)
	}
	TraceTerminalShape("rt-untrusted", "web.xterm.ondata", "<1b>[A", 3)
	data, err := os.ReadFile(TerminalTracePath("rt-untrusted"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "API_KEY_IS_SECRET") || (!strings.Contains(string(data), "<1b>[A") && !strings.Contains(string(data), `\u003c1b\u003e[A`)) {
		t.Fatalf("sanitization or valid shape persistence failed: %s", data)
	}
}

func TestTerminalTraceStopsAtFileSizeLimit(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("NEXUS_TERMINAL_TRACE", "1")
	path := TerminalTracePath("rt-limit")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(path, terminalTraceMaxFileBytes); err != nil {
		t.Fatal(err)
	}
	TraceTerminalBytes("rt-limit", "host.pty.write", []byte("\x1b[A"))
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != terminalTraceMaxFileBytes {
		t.Fatalf("trace grew beyond limit: %d", info.Size())
	}
}

func TestTerminalTraceShapeEncoderAlwaysProducesValidatedShape(t *testing.T) {
	for _, payload := range []string{
		"secret", "\x1b[A", "\x1bOC", "\x1b[?2004h", "\x1b[<35;12;9M",
		"\x1b]0;PRIVATE TITLE\x07", "\x03\r", "[A", "ação", "ARM COMMAND MARCH",
		strings.Repeat("x", 300),
	} {
		shape := TerminalTraceShape([]byte(payload))
		if !validClientTerminalTraceShape(shape) {
			t.Fatalf("masked encoder output was incorrectly rejected, input=%q shape=%q", payload, shape)
		}
	}
}
