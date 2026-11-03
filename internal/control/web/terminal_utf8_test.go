package web

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTerminalUTF8StreamPreservesSplitRunesAndANSI(t *testing.T) {
	input := []byte("\x1b[32m╭─olá\x1b[0m\r\n")
	var stream terminalUTF8Stream
	var got strings.Builder
	for _, b := range input {
		output := stream.Push([]byte{b})
		if output == "" {
			continue
		}
		// The WebSocket's JSON transport must not silently inject U+FFFD.
		marshaled, err := json.Marshal(TerminalMessage{Type: "output", Data: output})
		if err != nil {
			t.Fatal(err)
		}
		var frame TerminalMessage
		if err := json.Unmarshal(marshaled, &frame); err != nil {
			t.Fatal(err)
		}
		got.WriteString(frame.Data)
	}
	got.WriteString(stream.Flush())
	if got.String() != string(input) {
		t.Fatalf("stream changed PTY bytes: got %q; want %q", got.String(), string(input))
	}
}

func TestTerminalUTF8StreamFlushesIncompleteTailOnlyAtEnd(t *testing.T) {
	var stream terminalUTF8Stream
	if got := stream.Push([]byte{0xe2, 0x94}); got != "" {
		t.Fatalf("incomplete UTF-8 must be retained, got %q", got)
	}
	if got := stream.Push([]byte{0x80, 'A'}); got != "─A" {
		t.Fatalf("expected completed rune followed by ASCII, got %q", got)
	}
	if got := stream.Flush(); got != "" {
		t.Fatalf("completed stream must have no pending tail, got %q", got)
	}
}
