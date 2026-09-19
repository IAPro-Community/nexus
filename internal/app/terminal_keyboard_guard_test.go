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

func TestForwardAttachedInputReassertsKeyboardResetAfterCtrlC(t *testing.T) {
	var destination bytes.Buffer
	resets := 0
	input := []byte("hello\x03world")

	if err := forwardAttachedInput(&destination, input, func() { resets++ }); err != nil {
		t.Fatalf("forwardAttachedInput: %v", err)
	}
	if got := destination.String(); got != string(input) {
		t.Fatalf("forwarded input = %q, want %q", got, input)
	}
	if resets != 1 {
		t.Fatalf("keyboard reset calls = %d, want 1", resets)
	}
}

func TestForwardAttachedInputDoesNotResetForOrdinaryInput(t *testing.T) {
	var destination bytes.Buffer
	resets := 0

	if err := forwardAttachedInput(&destination, []byte("normal input\r"), func() { resets++ }); err != nil {
		t.Fatalf("forwardAttachedInput: %v", err)
	}
	if resets != 0 {
		t.Fatalf("keyboard reset calls = %d, want 0", resets)
	}
}

func TestContainsTerminalInterrupt(t *testing.T) {
	if !containsTerminalInterrupt([]byte("before\x03after")) {
		t.Fatal("Ctrl+C was not detected")
	}
	if containsTerminalInterrupt([]byte("ordinary input\r")) {
		t.Fatal("ordinary input was treated as Ctrl+C")
	}
}
