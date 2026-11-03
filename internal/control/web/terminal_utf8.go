package web

import "unicode/utf8"

// terminalUTF8Stream keeps a split multi-byte code point intact between PTY
// reads. JSON encoding individual raw chunks would otherwise replace a
// fragmented UTF-8 rune with U+FFFD before xterm could reconstruct it.
// ANSI control bytes are unaffected.
type terminalUTF8Stream struct {
	pending []byte
}

func (s *terminalUTF8Stream) Push(chunk []byte) string {
	data := append(s.pending, chunk...)
	split := len(data)
	for i := len(data) - 1; i >= 0 && i >= len(data)-utf8.UTFMax; i-- {
		if utf8.RuneStart(data[i]) {
			if !utf8.FullRune(data[i:]) {
				split = i
			}
			break
		}
	}
	s.pending = append(s.pending[:0], data[split:]...)
	return string(data[:split])
}

func (s *terminalUTF8Stream) Flush() string {
	rest := string(s.pending)
	s.pending = nil
	return rest
}
