package app

import (
	"bytes"
	"testing"
)

func TestForwardAttachedInputPreservesArrowAndCtrlCBytes(t *testing.T) {
	var destination bytes.Buffer
	reassertions := 0
	input := []byte("\x1b[A\x1b[B\x03\x1bOA\x1bOB")
	if err := forwardAttachedInput(&destination, input, func() { reassertions++ }); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(destination.Bytes(), input) {
		t.Fatalf("input was changed: got %q want %q", destination.Bytes(), input)
	}
	if reassertions != 1 {
		t.Fatalf("reassertions=%d, want 1", reassertions)
	}
}

func TestForwardAttachedInputDoesNotReassertWithoutCtrlC(t *testing.T) {
	var destination bytes.Buffer
	reassertions := 0
	if err := forwardAttachedInput(&destination, []byte("\x1b[Atexto\r"), func() { reassertions++ }); err != nil {
		t.Fatal(err)
	}
	if reassertions != 0 {
		t.Fatalf("reassertions=%d, want 0", reassertions)
	}
}
