package runtime

import (
	"bytes"
	"errors"
	"io"
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

func TestTerminalInputScrubberDropsX10WheelPacketsWithoutCancelingSelection(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalInputScrubber(&out)
	defer scrubber.Close()

	// 96/97 are wheel-up/down button codes encoded as 32+64/65.
	// X10 uses six bytes; CSI M is not a complete keyboard sequence.
	input := []byte("draft\x1b[M\x60\x48\x28\x1b[M\x61\x48\x28\x1b[A\x1b[B\r")
	if _, err := scrubber.Write(input); err != nil {
		t.Fatal(err)
	}
	if err := scrubber.Flush(); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "draft\x1b[A\x1b[B\r"; got != want {
		t.Fatalf("X10 wheel contaminated Codex selection: got %q, want %q", got, want)
	}
}

func TestTerminalInputScrubberCoalescesFragmentedX10AcrossWrites(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalInputScrubber(&out)
	defer scrubber.Close()

	parts := [][]byte{
		[]byte("choice"),
		{0x1b}, []byte("[M"), {0x60}, {0x48}, {0x28},
		[]byte("\x1bOB"), // legitimate application-cursor Down
		[]byte("\x1b[<65;21;3M"), // SGR wheel Down
		[]byte("\x1b[96;21;3M"),  // URXVT wheel Up
		[]byte("\x1b[A"),         // legitimate cursor Up
	}
	for i, part := range parts {
		if _, err := scrubber.Write(part); err != nil {
			t.Fatal(err)
		}
		if i >= 2 && i <= 5 && out.String() != "choice" {
			t.Fatalf("X10 fragment was forwarded before complete: %q", out.String())
		}
	}
	if err := scrubber.Flush(); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "choice\x1bOB\x1b[A"; got != want {
		t.Fatalf("mixed mouse/selector input = %q, want %q", got, want)
	}
}

func TestTerminalInputScrubberDropsIncompleteX10InsteadOfReleasingEscape(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalInputScrubber(&out)
	defer scrubber.Close()

	_, _ = scrubber.Write([]byte("\x1b[M\x60"))
	if got := out.String(); got != "" {
		t.Fatalf("incomplete X10 was forwarded: %q", got)
	}
	if err := scrubber.Flush(); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "" {
		t.Fatalf("incomplete X10 must be discarded on final flush: %q", got)
	}
}

func TestTerminalInputScrubberPreservesNonMouseCSIAndOutputDeleteLine(t *testing.T) {
	var in, out bytes.Buffer
	input := NewTerminalInputScrubber(&in)
	output := NewTerminalOutputScrubber(&out)
	defer input.Close()
	defer output.Close()

	// ESC [ M is a delete-line command in output, not a mouse packet.
	_, _ = output.Write([]byte("prompt\x1b[Mnext"))
	_ = output.Flush()
	if got := out.String(); got != "prompt\x1b[Mnext" {
		t.Fatalf("output CSI M changed: %q", got)
	}
	// Invalid legacy packet must not cause subsequent valid arrow keys to
	// disappear. Enter and explicit Ctrl+C remain byte-transparent.
	_, _ = input.Write([]byte("\x1b[M\x1b[A\x03\r"))
	_ = input.Flush()
	if got := in.String(); got != "\x1b[M\x1b[A\x03\r" {
		t.Fatalf("ordinary control bytes changed: %q", got)
	}
}

func TestTerminalInputScrubberDropsIncompleteSGRMouseOnFlush(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalInputScrubber(&out)
	defer scrubber.Close()

	_, _ = scrubber.Write([]byte("choice\x1b[<65;120;"))
	if got := out.String(); got != "choice" {
		t.Fatalf("partial SGR mouse report escaped before final flush: %q", got)
	}
	if err := scrubber.Flush(); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "choice" {
		t.Fatalf("partial SGR mouse report escaped on final flush: %q", got)
	}
}

func TestTerminalInputScrubberMouseReportTimeoutCannotEmitPartialX10(t *testing.T) {
	out := newSynchronizedBuffer()
	scrubber := NewTerminalInputScrubber(out)
	defer scrubber.Close()

	_, _ = scrubber.Write([]byte("\x1b[M\x60"))
	select {
	case <-out.wrote:
		t.Fatalf("partial mouse report leaked on timer: %q", out.String())
	case <-time.After(terminalInputSequenceCoalesceWindow + 150*time.Millisecond):
	}
	if got := out.String(); got != "" {
		t.Fatalf("timeout leaked X10 prefix into provider: %q", got)
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

	_, _ = scrubber.Write([]byte("before\x1b[?1000h\x1b[?1003h\x1b[?1006h\x1b[?1007h\x1b[?1015h\x1b[?1016hafter"))
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
	if terminalInputEscapeCoalesceWindow < 200*time.Millisecond {
		t.Fatalf("coalesce window %s is too short for delayed terminal selector fragments", terminalInputEscapeCoalesceWindow)
	}
	if terminalInputEscapeCoalesceWindow > 350*time.Millisecond {
		t.Fatalf("coalesce window %s adds excessive standalone Escape latency", terminalInputEscapeCoalesceWindow)
	}
	if terminalInputSequenceCoalesceWindow <= terminalInputEscapeCoalesceWindow {
		t.Fatalf("known CSI/SS3 prefixes need a longer hold window: escape=%s sequence=%s", terminalInputEscapeCoalesceWindow, terminalInputSequenceCoalesceWindow)
	}
}

func TestTerminalInputScrubberSurvivesDelayLongerThanOldArrowWindow(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalInputScrubber(&out)

	_, _ = scrubber.Write([]byte{0x1b})
	// The old 100ms timeout released ESC here and the later "[A" became
	// visible prompt text. Keep this above that historical boundary.
	time.Sleep(150 * time.Millisecond)
	_, _ = scrubber.Write([]byte("[A"))

	if got := out.String(); got != "\x1b[A" {
		t.Fatalf("delayed arrow sequence = %q, want %q", got, "\\x1b[A")
	}
}

func TestTerminalInputScrubberHoldsKnownCSIPrefixBeyondEscapeTimeout(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalInputScrubber(&out)

	_, _ = scrubber.Write([]byte{0x1b, '['})
	time.Sleep(terminalInputEscapeCoalesceWindow + 50*time.Millisecond)
	if got := out.String(); got != "" {
		t.Fatalf("known CSI prefix was flushed as raw input: %q", got)
	}
	_, _ = scrubber.Write([]byte{'D'})

	if got := out.String(); got != "\x1b[D" {
		t.Fatalf("delayed CSI completion = %q, want %q", got, "\\x1b[D")
	}
}

func TestTerminalOutputScrubberBlocksApplicationCursorNegotiation(t *testing.T) {
	var out bytes.Buffer
	scrubber := NewTerminalOutputScrubber(&out)

	_, _ = scrubber.Write([]byte("before\x1b[?1hafter\x1b[?1l"))
	if err := scrubber.Flush(); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "beforeafter" {
		t.Fatalf("DECCKM negotiation leaked to outer terminal: %q", got)
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

func TestTerminalInputScrubberCloseCancelsPendingEscapeTimer(t *testing.T) {
	out := newSynchronizedBuffer()
	scrubber := NewTerminalInputScrubber(out)
	if _, err := scrubber.Write([]byte{0x1b}); err != nil {
		t.Fatal(err)
	}
	if err := scrubber.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-out.wrote:
		t.Fatal("pending Escape was written after scrubber Close")
	case <-time.After(terminalInputEscapeCoalesceWindow + 50*time.Millisecond):
	}
	if got := out.String(); got != "" {
		t.Fatalf("closed scrubber wrote stale input: %q", got)
	}
	if _, err := scrubber.Write([]byte("text")); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("write after Close should reject input, got %v", err)
	}
	if err := scrubber.Flush(); err != nil {
		t.Fatalf("Flush after Close should remain idempotent: %v", err)
	}
}
