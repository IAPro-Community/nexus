package runtime

import (
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// TerminalOutputScrubber removes keyboard-protocol negotiation emitted by an
// interactive provider before its output reaches the user's terminal. The
// provider's PTY output is otherwise byte-transparent: screen drawing,
// cursor movement, colors and application data are preserved.
type TerminalOutputScrubber struct {
	mu                   sync.Mutex
	out                  io.Writer
	pending              []byte
	drop                 func([]byte) bool
	inputMetadata        bool
	mouseState           *TerminalMouseState
	forwardMouse         bool
	flushIncompleteAfter time.Duration
	flushTimer           *time.Timer
	flushGeneration      uint64
	asyncErr             error
	closed               bool
}

// ConPTY and embedded terminal bridges may deliver ESC separately from the
// remainder of CSI/SS3 selector keys under load. A short fixed timeout can
// turn an arrow into a standalone ESC followed by literal "[A"/"OA" bytes.
// Keep standalone Escape responsive, but once CSI/SS3 is recognized hold the
// incomplete sequence much longer: a partial control sequence is never useful
// provider input by itself.
const (
	terminalInputEscapeCoalesceWindow   = 250 * time.Millisecond
	terminalInputSequenceCoalesceWindow = 2 * time.Second
)

func NewTerminalOutputScrubber(out io.Writer) *TerminalOutputScrubber {
	return &TerminalOutputScrubber{out: out, drop: isKeyboardNegotiation}
}

// NewTerminalInputScrubber removes only terminal metadata that can arrive
// from the outer emulator. Navigation, control keys and paste bytes remain
// untouched.
func NewTerminalInputScrubber(out io.Writer) *TerminalOutputScrubber {
	return &TerminalOutputScrubber{
		out:                  out,
		drop:                 isTerminalInputMetadata,
		inputMetadata:        true,
		flushIncompleteAfter: terminalInputEscapeCoalesceWindow,
	}
}

// TerminalMouseState records tracking modes requested by the child TUI, not
// unsolicited mouse reports from the outer emulator. The final host input
// boundary only accepts mouse events while tracking is enabled.
type TerminalMouseState struct {
	modes atomic.Uint32
}

func (m *TerminalMouseState) Enabled() bool {
	return m != nil && m.modes.Load() != 0
}

func (m *TerminalMouseState) ObserveOutputSequence(sequence []byte) {
	if m == nil || len(sequence) < 5 || sequence[0] != 0x1b || sequence[1] != '[' {
		return
	}
	final := sequence[len(sequence)-1]
	if final != 'h' && final != 'l' {
		return
	}
	body := string(sequence[2 : len(sequence)-1])
	if !strings.HasPrefix(body, "?") {
		return
	}
	var bits uint32
	for _, mode := range strings.Split(body[1:], ";") {
		switch mode {
		case "9":
			bits |= 1 // X10 mouse tracking
		case "1000":
			bits |= 2 // normal mouse tracking
		case "1002":
			bits |= 4 // button-event tracking
		case "1003":
			bits |= 8 // any-event tracking
		}
	}
	if bits == 0 {
		return
	}
	for {
		before := m.modes.Load()
		after := before | bits
		if final == 'l' {
			after = before &^ bits
		}
		if m.modes.CompareAndSwap(before, after) {
			return
		}
	}
}

// Output negotiation must reach xterm/host TTY so cursor and mouse modes
// work. The same parser updates the host's mouse admission state even when
// escape sequences arrive fragmented across PTY reads.
func NewTerminalOutputScrubberTrackingMouse(out io.Writer, state *TerminalMouseState) *TerminalOutputScrubber {
	s := NewTerminalOutputScrubber(out)
	s.mouseState = state
	return s
}

// Upstream WebSocket/CLI filters may pass complete mouse packets through.
// The SessionHost still validates the child-negotiated tracking state.
func NewTerminalInputScrubberAllowMouse(out io.Writer) *TerminalOutputScrubber {
	s := NewTerminalInputScrubber(out)
	s.forwardMouse = true
	return s
}

func NewTerminalInputScrubberWithMouseState(out io.Writer, state *TerminalMouseState) *TerminalOutputScrubber {
	s := NewTerminalInputScrubber(out)
	s.mouseState = state
	return s
}

func (s *TerminalOutputScrubber) mouseForwardAllowed() bool {
	return s.forwardMouse || s.mouseState.Enabled()
}

func (s *TerminalOutputScrubber) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, io.ErrClosedPipe
	}
	if s.asyncErr != nil {
		err := s.asyncErr
		s.asyncErr = nil
		return 0, err
	}
	s.stopFlushTimerLocked()
	s.pending = append(s.pending, p...)
	if err := s.flushLocked(false); err != nil {
		return len(p), err
	}
	s.scheduleIncompleteFlushLocked()
	return len(p), nil
}

func (s *TerminalOutputScrubber) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.stopFlushTimerLocked()
	asyncErr := s.asyncErr
	s.asyncErr = nil
	flushErr := s.flushLocked(true)
	if asyncErr != nil {
		return asyncErr
	}
	return flushErr
}

// Close cancels delayed input flushing and discards incomplete terminal
// sequences. A stopped provider must never receive a timer-driven PTY write.
func (s *TerminalOutputScrubber) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopFlushTimerLocked()
	s.pending = nil
	s.asyncErr = nil
	s.closed = true
	return nil
}

func (s *TerminalOutputScrubber) stopFlushTimerLocked() {
	s.flushGeneration++
	if s.flushTimer == nil {
		return
	}
	s.flushTimer.Stop()
	s.flushTimer = nil
}

func (s *TerminalOutputScrubber) scheduleIncompleteFlushLocked() {
	if s.flushIncompleteAfter <= 0 || len(s.pending) == 0 {
		return
	}
	delay := s.flushIncompleteAfter
	if isIncompleteTerminalSequence(s.pending) && terminalInputSequenceCoalesceWindow > delay {
		delay = terminalInputSequenceCoalesceWindow
	}
	s.flushGeneration++
	generation := s.flushGeneration
	s.flushTimer = time.AfterFunc(delay, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if generation != s.flushGeneration {
			return
		}
		s.flushTimer = nil
		s.flushGeneration++
		if err := s.flushLocked(true); err != nil && s.asyncErr == nil {
			s.asyncErr = err
		}
	})
}

func isIncompleteTerminalSequence(pending []byte) bool {
	if len(pending) < 2 || pending[0] != 0x1b {
		return false
	}
	switch pending[1] {
	case '[':
		// flushLocked leaves CSI here only when no final byte has arrived yet.
		return true
	case 'O':
		// SS3 selector keys are exactly ESC O <final>; hold ESC O atomically.
		return len(pending) < 3
	default:
		return false
	}
}

func (s *TerminalOutputScrubber) flushLocked(final bool) error {
	if len(s.pending) == 0 {
		return nil
	}

	clean := make([]byte, 0, len(s.pending))
	pos := 0
	for pos < len(s.pending) {
		if s.pending[pos] != 0x1b {
			clean = append(clean, s.pending[pos])
			pos++
			continue
		}
		if pos+1 >= len(s.pending) {
			if final {
				clean = append(clean, s.pending[pos:]...)
				pos = len(s.pending)
			}
			break
		}
		if s.pending[pos+1] == 'O' {
			// SS3 application-cursor sequences (ESC O A/B/C/D/H/F) are used
			// by interactive selectors when DECCKM is active. Coalesce the
			// complete sequence so a split WebSocket/IPC write can never turn
			// the final key byte into literal provider input.
			if pos+2 >= len(s.pending) {
				if final {
					clean = append(clean, s.pending[pos:]...)
					pos = len(s.pending)
				}
				break
			}
			clean = append(clean, s.pending[pos:pos+3]...)
			pos += 3
			continue
		}
		if s.pending[pos+1] != '[' {
			clean = append(clean, s.pending[pos])
			pos++
			continue
		}

		// X10 mouse reports are ESC [ M Cb Cx Cy. Unlike SGR mouse reports,
		// the final 'M' is a prefix, not the end of the six-byte sequence.
		// Parsing CSI first used to forward ESC[M and then the three mouse
		// bytes as separate keystrokes, disrupting Codex's choice dialogs.
		// This branch belongs to the input scrubber only: ESC[M is a valid
		// delete-line output command and must remain untouched on output.
		if s.inputMetadata && pos+2 < len(s.pending) && s.pending[pos+2] == 'M' {
			if len(s.pending)-pos < 6 {
				if final {
					// An interrupted mouse packet is never valid keyboard input.
					// Do not release a stale ESC[M prefix into a live selector.
					pos = len(s.pending)
				}
				break
			}
			mouse := s.pending[pos : pos+6]
			if mouse[3] >= 0x20 && mouse[4] >= 0x20 && mouse[5] >= 0x20 {
				if s.mouseForwardAllowed() {
					clean = append(clean, mouse...)
				}
				pos += 6
				continue
			}
			// An invalid packet is not a mouse report. Let the ordinary
			// CSI decoder preserve it rather than swallowing user input.
		}

		end := -1
		for i := pos + 2; i < len(s.pending); i++ {
			// A few embedded bridges leak malformed focus reports such as
			// ESC[0[I. Treat the nested bracket plus its final I/O as one
			// metadata sequence so it cannot leave "0[I" in the prompt.
			if s.pending[i] == '[' && i+1 < len(s.pending) {
				next := s.pending[i+1]
				if next == 'I' || next == 'O' || next == 'i' || next == 'o' {
					end = i + 1
					break
				}
			}
			if s.pending[i] >= 0x40 && s.pending[i] <= 0x7e {
				end = i
				break
			}
		}
		if end < 0 {
			if final {
				// A CSI '<' prefix is unambiguously an SGR mouse packet,
				// not a keyboard arrow. Never flush its truncated payload
				// into a live Codex choice prompt when a source disconnects
				// or the incomplete-input coalescing timer expires.
				if s.inputMetadata && pos+2 < len(s.pending) && s.pending[pos+2] == '<' {
					pos = len(s.pending)
					break
				}
				clean = append(clean, s.pending[pos:]...)
				pos = len(s.pending)
			}
			break
		}

		sequence := s.pending[pos : end+1]
		if !s.inputMetadata {
			s.mouseState.ObserveOutputSequence(sequence)
		}
		drop := s.drop != nil && s.drop(sequence)
		if drop && s.inputMetadata && s.mouseForwardAllowed() && isMouseInputSequence(sequence) {
			drop = false
		}
		if !drop {
			clean = append(clean, sequence...)
		}
		pos = end + 1
	}

	s.pending = append(s.pending[:0], s.pending[pos:]...)
	if len(clean) == 0 || s.out == nil {
		return nil
	}
	n, err := s.out.Write(clean)
	if err == nil && n != len(clean) {
		return io.ErrShortWrite
	}
	return err
}

func isKeyboardNegotiation(sequence []byte) bool {
	if len(sequence) < 4 || sequence[0] != 0x1b || sequence[1] != '[' {
		return false
	}

	final := sequence[len(sequence)-1]
	body := sequence[2 : len(sequence)-1]
	switch final {
	case 'u', 'm':
		// Kitty keyboard protocol and modifyOtherKeys use a private CSI
		// parameter beginning with '>'.
		return len(body) > 0 && body[0] == '>'
	case 'h', 'l':
		// Preserve DECCKM (?1), mouse tracking and mouse encodings: xterm
		// must see those negotiations to send the correct keys and clicks.
		// Unsolicited focus reports remain filtered from provider input.
		return string(body) == "?1004"
	default:
		return false
	}
}

func isTerminalInputMetadata(sequence []byte) bool {
	if len(sequence) < 3 || sequence[0] != 0x1b || sequence[1] != '[' {
		return false
	}

	final := sequence[len(sequence)-1]
	body := sequence[2 : len(sequence)-1]
	switch final {
	case 'I', 'O', 'i', 'o':
		// Focus reports and their parameterized variants, including the
		// malformed ESC[0[I shape seen in embedded terminal bridges.
		if len(body) == 0 || numericCSIParameters(body) {
			return true
		}
		return len(body) > 1 && body[len(body)-1] == '[' && numericCSIParameters(body[:len(body)-1])
	case 'R':
		// Cursor-position reports are terminal metadata, never user input.
		return numericCSIParameters(body)
	case 'M', 'm':
		return isMouseInputSequence(sequence)
	default:
		return false
	}
}

// isMouseInputSequence recognizes complete mouse reports only; cursor keys
// and text remain distinct even when input packets are coalesced.
func isMouseInputSequence(sequence []byte) bool {
	if len(sequence) < 4 || sequence[0] != 0x1b || sequence[1] != '[' {
		return false
	}
	final := sequence[len(sequence)-1]
	if final != 'M' && final != 'm' {
		return false
	}
	body := sequence[2 : len(sequence)-1]
	if len(body) > 1 && body[0] == '<' && numericCSIParameters(body[1:]) {
		return true
	}
	if final == 'M' {
		fields := bytesSplitMouseParameters(body)
		return len(fields) == 3 && fields[0] >= 32
	}
	return false
}

// bytesSplitMouseParameters recognizes only complete numeric URXVT packets.
// General-purpose CSI numeric validation deliberately allows empty fields;
// mouse tracking must be stricter to avoid dropping ambiguous user input.
func bytesSplitMouseParameters(body []byte) []int {
	fields := strings.Split(string(body), ";")
	if len(fields) != 3 {
		return nil
	}
	values := make([]int, 0, 3)
	for _, field := range fields {
		if field == "" {
			return nil
		}
		n, err := strconv.Atoi(field)
		if err != nil || n < 0 {
			return nil
		}
		values = append(values, n)
	}
	return values
}

func numericCSIParameters(value []byte) bool {
	if len(value) == 0 {
		return true
	}
	for _, b := range value {
		if (b < '0' || b > '9') && b != ';' {
			return false
		}
	}
	return true
}
