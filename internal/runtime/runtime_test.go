package runtime

import (
	"bytes"
	"testing"
)

func TestWriteTerminalKeyboardModeReset(t *testing.T) {
	var out bytes.Buffer
	writeTerminalKeyboardModeReset(&out)
	if got := out.String(); got != terminalKeyboardModeReset {
		t.Fatalf("keyboard reset = %q, want %q", got, terminalKeyboardModeReset)
	}
}

func TestNormalizeInteractiveEnvReplacesDumbTerminal(t *testing.T) {
	env := []string{"PATH=/bin", "TERM=dumb", "LANG=pt_BR.UTF-8"}
	normalized := NormalizeInteractiveEnv(env)

	if normalized[1] != "TERM=xterm-256color" {
		t.Fatalf("expected interactive terminal type, got %q", normalized[1])
	}
	if env[1] != "TERM=dumb" {
		t.Fatalf("normalization must not mutate caller environment, got %q", env[1])
	}
}

func TestNormalizeInteractiveEnvPreservesNormalTerminal(t *testing.T) {
	env := []string{"TERM=screen-256color"}
	normalized := NormalizeInteractiveEnv(env)

	if len(normalized) != 1 || normalized[0] != env[0] {
		t.Fatalf("normal terminal environment changed: %+v", normalized)
	}
}
