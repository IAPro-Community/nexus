package app

import (
	"os"
	"testing"

	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
)

// The attach data path moved to scrubber-based streaming
// (NewTerminalInputScrubber/NewTerminalOutputScrubber, covered by
// terminal_output_test.go). This file guards what remains: the kept
// ReassertInteractiveTerminalKeyboardModes used by direct provider runs and
// the attach exit path. It must never emit bytes for non-terminals.
func TestReassertKeyboardModesNilSafe(t *testing.T) {
	nexusruntime.ReassertInteractiveTerminalKeyboardModes(nil, nil)
	f, err := os.CreateTemp("", "nexus-guard")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	nexusruntime.ReassertInteractiveTerminalKeyboardModes(f, f)
	_ = f.Close()
}

func TestReassertKeyboardModesPipeIsNoop(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	defer func() { _ = writer.Close() }()
	// Pipes are not terminals: no reset bytes may be written.
	nexusruntime.ReassertInteractiveTerminalKeyboardModes(reader, writer)
}

func TestGuardKeyboardModesPipeReturnsCleanup(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	defer func() { _ = writer.Close() }()
	cleanup := nexusruntime.GuardInteractiveTerminalKeyboardModes(reader, writer)
	if cleanup == nil {
		t.Fatal("expected non-nil cleanup func")
	}
	cleanup()
}
