package runtime

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Golden PTY replay for issue #196.
//
// The fixture is a sanitized capture of a real 1000-event interactive PTY
// session: the byte stream contains keyboard-protocol negotiation, an OSC 11
// color query, truecolor SGR, cursor movement and the final multi-line
// prompt. Redaction replaced the operator email and workspace path only; the
// ANSI structure is byte-identical to the captured stream.
//
// The scrubbed fixture is the byte-exact expectation for the output scrubber.
// Any change that alters, reorders or drops unrelated bytes breaks the
// replay, which is exactly the regression class that produced the black
// screen and lost prompt reports.
func TestTerminalOutputScrubberGoldenReplay(t *testing.T) {
	raw := readGoldenPTY(t, "pty196_golden.raw.bin")
	want := readGoldenPTY(t, "pty196_golden.scrubbed.bin")

	var out bytes.Buffer
	scrubber := NewTerminalOutputScrubber(&out)
	if _, err := scrubber.Write(raw); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := scrubber.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	if !bytes.Equal(out.Bytes(), want) {
		t.Fatalf("golden replay mismatch: got %d bytes, want %d bytes", out.Len(), len(want))
	}

	for _, sequence := range [][]byte{
		[]byte("\x1b[>4m"),
		[]byte("\x1b[>4;2m"),
		[]byte("\x1b[>5u"),
	} {
		if bytes.Contains(want, sequence) {
			t.Fatalf("scrubbed golden still contains keyboard negotiation %q", sequence)
		}
	}

	for _, sequence := range [][]byte{
		[]byte("\x1b[?u"),
		[]byte("\x1b]11;?\x07"),
	} {
		if !bytes.Contains(want, sequence) {
			t.Fatalf("scrubbed golden lost terminal sequence %q", sequence)
		}
	}
}

// TestTerminalOutputScrubberGoldenReplayChunked proves the golden result is
// independent of how the PTY reader splits the stream.
func TestTerminalOutputScrubberGoldenReplayChunked(t *testing.T) {
	raw := readGoldenPTY(t, "pty196_golden.raw.bin")
	want := readGoldenPTY(t, "pty196_golden.scrubbed.bin")

	for _, chunkSize := range []int{1, 2, 3, 7, 64, 1024} {
		var out bytes.Buffer
		scrubber := NewTerminalOutputScrubber(&out)
		for offset := 0; offset < len(raw); offset += chunkSize {
			end := offset + chunkSize
			if end > len(raw) {
				end = len(raw)
			}
			if _, err := scrubber.Write(raw[offset:end]); err != nil {
				t.Fatalf("chunk %d: Write: %v", chunkSize, err)
			}
		}
		if err := scrubber.Flush(); err != nil {
			t.Fatalf("chunk %d: Flush: %v", chunkSize, err)
		}
		if !bytes.Equal(out.Bytes(), want) {
			t.Fatalf("chunk %d: golden replay mismatch: got %d bytes, want %d bytes", chunkSize, out.Len(), len(want))
		}
	}
}

func readGoldenPTY(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read golden fixture %s: %v", name, err)
	}
	if len(data) == 0 {
		t.Fatalf("golden fixture %s is empty", name)
	}
	return data
}
