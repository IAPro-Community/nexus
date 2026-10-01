package host

import (
	"bytes"
	"testing"
)

func TestShellPipeInputNormalizerConvertsTerminalEnterToCRLF(t *testing.T) {
	var normalizer shellPipeInputNormalizer
	var got []byte
	got = append(got, normalizer.Normalize([]byte("echo marker\r"))...)
	got = append(got, normalizer.Normalize([]byte("\nnext\n"))...)

	if want := []byte("echo marker\r\nnext\r\n"); !bytes.Equal(got, want) {
		t.Fatalf("normalized shell input = %q, want %q", got, want)
	}
}

func TestShellPipeInputNormalizerPreservesOrdinaryAndControlBytes(t *testing.T) {
	var normalizer shellPipeInputNormalizer
	input := []byte("hello\x03\x1b[A\r")
	want := []byte("hello\x03\x1b[A\r\n")
	if got := normalizer.Normalize(input); !bytes.Equal(got, want) {
		t.Fatalf("normalized shell input = %q, want %q", got, want)
	}
}
