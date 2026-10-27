package runtime

import (
	"strings"
	"testing"
)

func envValue(env []string, key string) (string, bool) {
	for _, entry := range env {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 && parts[0] == key {
			return parts[1], true
		}
	}
	return "", false
}

func TestApplyEmbeddedTerminalCompatAddsCodexKeyboardDefault(t *testing.T) {
	env := []string{"PATH=/usr/bin", "TERM=xterm-256color"}

	got := ApplyEmbeddedTerminalCompat(env)

	value, ok := envValue(got, EnvCodexDisableKeyboardEnhancement)
	if !ok || value != "1" {
		t.Fatalf("%s=%q (present=%t), want 1", EnvCodexDisableKeyboardEnhancement, value, ok)
	}
	if len(env) != 2 {
		t.Fatalf("input environment was mutated: %#v", env)
	}
}

func TestApplyEmbeddedTerminalCompatPreservesExplicitValue(t *testing.T) {
	for _, want := range []string{"0", "1", ""} {
		env := []string{EnvCodexDisableKeyboardEnhancement + "=" + want}

		got := ApplyEmbeddedTerminalCompat(env)

		value, ok := envValue(got, EnvCodexDisableKeyboardEnhancement)
		if !ok || value != want {
			t.Fatalf("explicit %s=%q became %q (present=%t)", EnvCodexDisableKeyboardEnhancement, want, value, ok)
		}
		if len(got) != 1 {
			t.Fatalf("explicit value was duplicated: %#v", got)
		}
	}
}

func TestApplyEmbeddedTerminalCompatDoesNotDuplicateExistingKeys(t *testing.T) {
	env := []string{
		EnvCodexDisableKeyboardEnhancement + "=1",
		EnvCodexDisableKeyboardEnhancement + "=0",
	}

	got := ApplyEmbeddedTerminalCompat(env)

	count := 0
	for _, entry := range got {
		if strings.HasPrefix(entry, EnvCodexDisableKeyboardEnhancement+"=") {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("compat helper changed existing entries: %#v", got)
	}
}
