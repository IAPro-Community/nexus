package runtime

import (
	"io"
	"sync"
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
	flushIncompleteAfter time.Duration
	flushTimer           *time.Timer
	flushGeneration      uint64
	asyncErr             error
}

const terminalInputEscapeCoalesceWindow = 25 * time.Millisecond

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
		flushIncompleteAfter: terminalInputEscapeCoalesceWindow,
	}
}

func (s *TerminalOutputScrubber) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	s.stopFlushTimerLocked()
	asyncErr := s.asyncErr
	s.asyncErr = nil
	flushErr := s.flushLocked(true)
	if asyncErr != nil {
		return asyncErr
	}
	return flushErr
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
		if s.pending[pos+1] != '[' {
			clean = append(clean, s.pending[pos])
			pos++
			continue
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
				clean = append(clean, s.pending[pos:]...)
				pos = len(s.pending)
			}
			break
		}

		sequence := s.pending[pos : end+1]
		if s.drop == nil || !s.drop(sequence) {
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
		// Focus reporting and bracketed paste are input protocols. Enabling
		// either on the outer terminal makes its reports leak into the provider.
		return string(body) == "?1004" || string(body) == "?2004"
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
	default:
		return false
	}
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
