package terminal

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestKeyboardModeResetDisablesEveryProviderEnhancement(t *testing.T) {
	var out bytes.Buffer
	if err := ResetKeyboardModes(&out); err != nil {
		t.Fatal(err)
	}
	reset := out.String()
	for _, sequence := range []string{
		"\x1b[?1l",    // application cursor
		"\x1b[>0u",    // Kitty/CSI-u enhancements
		"\x1b[>4;0m",  // modifyOtherKeys
		"\x1b[?1004l", // focus reporting
		"\x1b[?2004l", // bracketed paste
	} {
		if !strings.Contains(reset, sequence) {
			t.Fatalf("reset %q does not contain %q", reset, sequence)
		}
	}
}

func TestResetKeyboardModesReportsWriterFailure(t *testing.T) {
	err := ResetKeyboardModes(errorWriter{})
	if err == nil || !strings.Contains(err.Error(), "reset terminal keyboard modes") {
		t.Fatalf("expected contextual reset error, got %v", err)
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errors.New("closed") }

func TestTTYControllerNonTTYIsSafeNoOp(t *testing.T) {
	controller := NewTTYController(nil, &bytes.Buffer{})
	if err := controller.Enter(); err != nil {
		t.Fatal(err)
	}
	if err := controller.Reassert(); err != nil {
		t.Fatal(err)
	}
	if err := controller.Restore(); err != nil {
		t.Fatal(err)
	}
}
