package web

import (
	"bytes"
	"io"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
)

func TestWebInteractiveLeadRequiresExplicitLabel(t *testing.T) {
	if webInteractiveLeadEnabled(registry.RuntimeSession{}) {
		t.Fatal("missing label must not enable Nexus Lead interception")
	}
	if webInteractiveLeadEnabled(registry.RuntimeSession{Labels: map[string]string{"nexus.interactive_lead": "false"}}) {
		t.Fatal("false label must keep provider-native terminal behavior")
	}
	if !webInteractiveLeadEnabled(registry.RuntimeSession{Labels: map[string]string{"nexus.interactive_lead": "TRUE"}}) {
		t.Fatal("explicit true label must enable the Lead promotion boundary")
	}
}

func TestWebInteractiveDelegationModeFailsClosed(t *testing.T) {
	t.Setenv("NEXUS_DELEGATION_MODE", "")
	if got := webInteractiveDelegationMode(registry.RuntimeSession{}); got != nexus.DelegationOff {
		t.Fatalf("missing web delegation mode=%q want OFF", got)
	}
	if got := webInteractiveDelegationMode(registry.RuntimeSession{Labels: map[string]string{"nexus.delegation_mode": "garbage"}}); got != nexus.DelegationOff {
		t.Fatalf("invalid web delegation mode=%q want OFF", got)
	}
	if got := webInteractiveDelegationMode(registry.RuntimeSession{Labels: map[string]string{"nexus.delegation_mode": "AUTO"}}); got != nexus.DelegationAuto {
		t.Fatalf("explicit web AUTO=%q", got)
	}
}

func TestWebTerminalInputWriterCoalescesSplitSelectorKeys(t *testing.T) {
	var out bytes.Buffer
	stream := nexusruntime.NewTerminalInputScrubber(terminalInputCommandWriter(func(data []byte) (int, error) {
		return out.Write(data)
	}))

	for _, chunk := range [][]byte{
		{0x1b},
		[]byte("[A"),
		{0x1b},
		{'O'},
		{'B'},
	} {
		if _, err := stream.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}
	if err := stream.Flush(); err != nil {
		t.Fatal(err)
	}

	if got, want := out.String(), "\x1b[A\x1bOB"; got != want {
		t.Fatalf("selector input changed: got %q want %q", got, want)
	}
}

func TestWebTerminalDirectInputNeverEntersPromotionFilter(t *testing.T) {
	var direct bytes.Buffer
	var intercepted bytes.Buffer
	// Fragments include an arrow, Ctrl+C and Enter. Direct Web traffic must
	// reach the SessionHost unchanged; it alone handles PTY coalescing.
	chunks := []string{"\x1b", "[A", "codex", "\x03", "\r"}
	for _, chunk := range chunks {
		dispatchWebTerminalInput(chunk, false, func(raw string) {
			_, _ = direct.WriteString(raw)
		}, func(raw string) {
			_, _ = intercepted.WriteString(raw)
		})
	}
	if got, want := direct.String(), "\x1b[Acodex\x03\r"; got != want {
		t.Fatalf("raw Web input was changed: got %q want %q", got, want)
	}
	if intercepted.Len() != 0 {
		t.Fatal("direct Codex traffic must not reach the promotion path")
	}
}

func TestWebTerminalPromotionInputRequiresExplicitMode(t *testing.T) {
	var direct, promotion bytes.Buffer
	dispatchWebTerminalInput("test\r", true, func(raw string) {
		_, _ = direct.WriteString(raw)
	}, func(raw string) {
		_, _ = promotion.WriteString(raw)
	})
	if direct.Len() != 0 || promotion.String() != "test\r" {
		t.Fatalf("interactive promotion dispatch mismatch: direct=%q promoted=%q", direct.String(), promotion.String())
	}
}

func TestWriteAttachedTerminalInputPreservesBytesWithoutRPCFraming(t *testing.T) {
	var out bytes.Buffer
	for _, chunk := range []string{"echo __NEXUS_DOCS_TERMINAL_OK__", "\r", "\x1b[A", "\x03"} {
		if err := writeAttachedTerminalInput(&out, chunk); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := out.String(), "echo __NEXUS_DOCS_TERMINAL_OK__\r\x1b[A\x03"; got != want {
		t.Fatalf("terminal input changed: got %q want %q", got, want)
	}
	if bytes.Contains(out.Bytes(), []byte(`"command":"input"`)) {
		t.Fatal("browser input must never be wrapped as CmdInput JSON")
	}
}

type shortTerminalWriter struct{}

func (shortTerminalWriter) Write(data []byte) (int, error) {
	return len(data) - 1, nil
}

func TestWriteAttachedTerminalInputRejectsShortWrite(t *testing.T) {
	if got := writeAttachedTerminalInput(shortTerminalWriter{}, "hello"); got != io.ErrShortWrite {
		t.Fatalf("short write must be reported, got %v", got)
	}
}
