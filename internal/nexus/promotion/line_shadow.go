// Package promotion contains the terminal-side policy boundary that decides
// when an interactive Lead prompt may become a Nexus Mission.
package promotion

import (
	"bytes"
	"unicode/utf8"
)

// LineEvent is the result of consuming terminal bytes. Forward is always the
// exact byte stream that may be written to the provider PTY. Enter is held so
// the caller can decide whether to submit the line to the Lead or promote it.
type LineEvent struct {
	Forward     []byte
	Line        string
	Submit      bool
	Submissions []string
}

// LineShadow keeps a byte-transparent terminal shadow. Escape/control
// sequences and bracketed paste are never copied into the logical line.
type LineShadow struct {
	enabled   bool
	line      []byte
	escape    []byte
	inPaste   bool
	pasteScan []byte
}

func NewLineShadow() *LineShadow { return &LineShadow{enabled: true} }

func (s *LineShadow) SetEnabled(enabled bool) {
	if s == nil {
		return
	}
	s.enabled = enabled
}

func (s *LineShadow) Enabled() bool { return s != nil && s.enabled }

func (s *LineShadow) Line() string {
	if s == nil {
		return ""
	}
	return string(s.line)
}

func (s *LineShadow) Reset() {
	if s == nil {
		return
	}
	s.line = s.line[:0]
	s.escape = s.escape[:0]
	s.inPaste = false
	s.pasteScan = s.pasteScan[:0]
}

// Process and Feed are aliases kept convenient for stream adapters.
func (s *LineShadow) Process(input []byte) LineEvent { return s.Consume(input) }
func (s *LineShadow) Feed(input []byte) LineEvent    { return s.Consume(input) }

func (s *LineShadow) Consume(input []byte) LineEvent {
	event := LineEvent{}
	if len(input) == 0 {
		return event
	}
	if s == nil || !s.enabled {
		event.Forward = append([]byte(nil), input...)
		return event
	}
	event.Forward = make([]byte, 0, len(input))
	for _, b := range input {
		if s.inPaste {
			event.Forward = append(event.Forward, b)
			s.consumePasteByte(b)
			continue
		}
		if len(s.escape) > 0 {
			if b < 0x20 && b != 0x07 {
				event.Forward = append(event.Forward, b)
				s.line = s.line[:0]
				s.resetControlState()
				continue
			}
			event.Forward = append(event.Forward, b)
			s.consumeEscapeByte(b)
			continue
		}
		if b != 0x1b && b != '\r' && b != '\n' && b < 0x20 {
			event.Forward = append(event.Forward, b)
			if b == 0x08 {
				s.removeLastRune()
			} else {
				s.line = s.line[:0]
			}
			s.resetControlState()
			continue
		}
		// Enter is deliberately held for the caller's KEEP/PROMOTE/ASK
		// decision. Control keys remain byte-transparent but reset the shadow.
		if len(s.escape) == 0 {
			switch b {
			case 0x1b:
				event.Forward = append(event.Forward, b)
				s.escape = append(s.escape[:0], b)
				continue
			case '\r', '\n':
				event.Submit = true
				event.Submissions = append(event.Submissions, string(s.line))
				event.Line = string(s.line)
				s.line = s.line[:0]
				continue
			case 0x08, 0x7f:
				event.Forward = append(event.Forward, b)
				s.removeLastRune()
				continue
			}
		}
		event.Forward = append(event.Forward, b)
		switch b {
		case 0x1b:
			s.escape = append(s.escape[:0], b)
		default:
			s.line = append(s.line, b)
		}
	}
	return event
}

func (s *LineShadow) consumeEscapeByte(b byte) {
	s.escape = append(s.escape, b)
	if len(s.escape) == 2 {
		switch b {
		case '[':
			return
		case ']':
			return
		default:
			s.escape = s.escape[:0]
			return
		}
	}
	if len(s.escape) == 3 && s.escape[1] == '[' && b == '2' {
		return
	}
	if len(s.escape) >= 3 && s.escape[1] == '[' {
		// CSI final bytes are 0x40–0x7e. Recognize bracketed paste before
		// clearing the sequence so its body remains opaque to the shadow.
		if b >= 0x40 && b <= 0x7e {
			if bytes.Equal(s.escape[1:], []byte("[200~")) {
				s.inPaste = true
				s.pasteScan = s.pasteScan[:0]
			}
			s.escape = s.escape[:0]
		}
		return
	}
	// OSC terminates at BEL or ST (ESC backslash).
	if s.escape[1] == ']' {
		if b == 0x07 || (len(s.escape) >= 2 && b == '\\' && s.escape[len(s.escape)-2] == 0x1b) {
			s.escape = s.escape[:0]
		}
	}
}

func (s *LineShadow) consumePasteByte(b byte) {
	s.pasteScan = append(s.pasteScan, b)
	const end = "\x1b[201~"
	if len(s.pasteScan) > len(end) {
		s.pasteScan = s.pasteScan[len(s.pasteScan)-len(end):]
	}
	if bytes.Equal(s.pasteScan, []byte(end)) {
		s.inPaste = false
		s.pasteScan = s.pasteScan[:0]
	}
}

func (s *LineShadow) removeLastRune() {
	if len(s.line) == 0 {
		return
	}
	_, size := utf8.DecodeLastRune(s.line)
	if size <= 0 || size > len(s.line) {
		s.line = s.line[:len(s.line)-1]
		return
	}
	s.line = s.line[:len(s.line)-size]
}

func (s *LineShadow) resetControlState() {
	s.escape = s.escape[:0]
	s.inPaste = false
	s.pasteScan = s.pasteScan[:0]
}
