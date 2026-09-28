package runtime

import (
	"bytes"
	"sync"
	"testing"
	"time"
)

type synchronizedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
	wrote  chan struct{}
}

func newSynchronizedBuffer() *synchronizedBuffer {
	return &synchronizedBuffer{wrote: make(chan struct{}, 1)}
}

func (b *synchronizedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	n, err := b.buffer.Write(p)
	b.mu.Unlock()
	select {
	case b.wrote <- struct{}{}:
	default:
	}
	return n, err
}

func (b *synchronizedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

func TestTerminalOutputScrubberDropsKeyboardNegotiation(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalOutputScrubber(&out)

	_, _ = scrubber.Write([]byte("before\x1b[>1u\x1b[>4;1m\x1b[?1004h\x1b[?1003hafter"))
	if err := scrubber.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	if got := out.String(); got != "beforeafter" {
		t.Fatalf("scrubbed output = %q, want %q", got, "beforeafter")
	}
}

func TestTerminalOutputScrubberPreservesBracketedPaste(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalOutputScrubber(&out)

	_, _ = scrubber.Write([]byte("start\x1b[?2004hpaste-content\x1b[?2004lend"))
	if err := scrubber.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	if got := out.String(); got != "start\x1b[?2004hpaste-content\x1b[?2004lend" {
		t.Fatalf("bracketed paste was altered: %q", got)
	}
}

func TestTerminalOutputScrubberPreservesVisualANSI(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalOutputScrubber(&out)

	_, _ = scrubber.Write([]byte("\x1b[?25lvisible\x1b[2J"))
	_ = scrubber.Flush()

	if got := out.String(); got != "\x1b[?25lvisible\x1b[2J" {
		t.Fatalf("visual ANSI was changed: %q", got)
	}
}

func TestTerminalOutputScrubberHandlesSplitNegotiation(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalOutputScrubber(&out)

	_, _ = scrubber.Write([]byte("before\x1b[?100"))
	_, _ = scrubber.Write([]byte("4hafter"))
	_ = scrubber.Flush()

	if got := out.String(); got != "beforeafter" {
		t.Fatalf("split negotiation output = %q, want %q", got, "beforeafter")
	}
}

func TestTerminalInputScrubberDropsFocusAndCPR(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalInputScrubber(&out)

	_, _ = scrubber.Write([]byte("text\x1b[I\x1b[0[I\x1b[12;34R\x1b[C\x03\x16"))
	_ = scrubber.Flush()

	if got := out.String(); got != "text\x1b[C\x03\x16" {
		t.Fatalf("scrubbed input = %q, want %q", got, "text\\x1b[C\\x03\\x16")
	}
}

func TestTerminalInputScrubberDropsSGRMouseReports(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalInputScrubber(&out)

	_, _ = scrubber.Write([]byte("draft\x1b[<35;173;3M\x1b[<35;171;3M\x1b[<0;44;12mtext"))
	_ = scrubber.Flush()

	if got := out.String(); got != "drafttext" {
		t.Fatalf("SGR mouse reports leaked into provider input: %q", got)
	}
}

func TestTerminalInputScrubberCoalescesSplitSGRMouseReport(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalInputScrubber(&out)

	_, _ = scrubber.Write([]byte("\x1b[<35;173"))
	if got := out.String(); got != "" {
		t.Fatalf("partial SGR mouse report leaked before completion: %q", got)
	}
	_, _ = scrubber.Write([]byte(";3M"))

	if got := out.String(); got != "" {
		t.Fatalf("split SGR mouse report leaked into provider input: %q", got)
	}
}

func TestTerminalOutputScrubberDropsMouseTrackingNegotiation(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalOutputScrubber(&out)

	_, _ = scrubber.Write([]byte("before\x1b[?1000h\x1b[?1003h\x1b[?1006hafter"))
	_ = scrubber.Flush()

	if got := out.String(); got != "beforeafter" {
		t.Fatalf("mouse tracking negotiation leaked to outer terminal: %q", got)
	}
}

func TestTerminalInputScrubberCoalescesSplitArrowSequence(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalInputScrubber(&out)

	_, _ = scrubber.Write([]byte{0x1b})
	if got := out.String(); got != "" {
		t.Fatalf("escape prefix leaked before sequence completed: %q", got)
	}
	_, _ = scrubber.Write([]byte("[D"))

	if got := out.String(); got != "\x1b[D" {
		t.Fatalf("split arrow sequence = %q, want %q", got, "\\x1b[D")
	}
}

func TestTerminalInputScrubberCoalescesSplitSS3ArrowSequence(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalInputScrubber(&out)

	_, _ = scrubber.Write([]byte{0x1b})
	if got := out.String(); got != "" {
		t.Fatalf("escape prefix leaked before SS3 sequence completed: %q", got)
	}
	_, _ = scrubber.Write([]byte{'O'})
	if got := out.String(); got != "" {
		t.Fatalf("SS3 prefix leaked before final selector key: %q", got)
	}
	_, _ = scrubber.Write([]byte{'A'})

	if got := out.String(); got != "\x1bOA" {
		t.Fatalf("split SS3 arrow sequence = %q, want %q", got, "\\x1bOA")
	}
}

func TestTerminalInputEscapeCoalesceWindowCoversConPTYFragmentation(t *testing.T) {
	if terminalInputEscapeCoalesceWindow < 75*time.Millisecond {
		t.Fatalf("coalesce window %s is too short for delayed ConPTY selector fragments", terminalInputEscapeCoalesceWindow)
	}
	if terminalInputEscapeCoalesceWindow > 150*time.Millisecond {
		t.Fatalf("coalesce window %s adds excessive standalone Escape latency", terminalInputEscapeCoalesceWindow)
	}
}

func TestTerminalInputScrubberForwardsStandaloneEscapeAfterCoalesceWindow(t *testing.T) {
	out := newSynchronizedBuffer()
	scrubber := NewTerminalInputScrubber(out)

	_, _ = scrubber.Write([]byte{0x1b})
	if got := out.String(); got != "" {
		t.Fatalf("standalone Escape must be held briefly, got %q", got)
	}

	select {
	case <-out.wrote:
	case <-time.After(10 * terminalInputEscapeCoalesceWindow):
		t.Fatal("standalone Escape was not forwarded after coalesce window")
	}
	if got := out.String(); got != "\x1b" {
		t.Fatalf("standalone Escape was not forwarded after coalesce window: %q", got)
	}
}
