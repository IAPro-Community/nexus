package runtime

import (
	"bytes"
	"testing"
)

func TestTerminalOutputScrubberDropsKeyboardNegotiation(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalOutputScrubber(&out)

	_, _ = scrubber.Write([]byte("before\x1b[>1u\x1b[>4;1m\x1b[?1004h\x1b[?2004hafter"))
	if err := scrubber.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	if got := out.String(); got != "beforeafter" {
		t.Fatalf("scrubbed output = %q, want %q", got, "beforeafter")
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

func TestTerminalInputScrubberForwardsStandaloneEscape(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalInputScrubber(&out)

	_, _ = scrubber.Write([]byte{0x1b})

	if got := out.String(); got != "\x1b" {
		t.Fatalf("standalone Escape was delayed: %q", got)
	}
}
