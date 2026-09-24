//go:build windows

package desktop

import (
	"strings"
	"testing"
)

func TestWindowsAutostartCommandQuotesExecutable(t *testing.T) {
	got := windowsAutostartCommand(`C:\Program Files\Nexus\nexus-hud.exe`, []string{"--example", "two words"})
	if !strings.HasPrefix(got, `"C:\Program Files\Nexus\nexus-hud.exe"`) {
		t.Fatalf("command=%q", got)
	}
	if !strings.Contains(got, `"two words"`) {
		t.Fatalf("command=%q", got)
	}
}
