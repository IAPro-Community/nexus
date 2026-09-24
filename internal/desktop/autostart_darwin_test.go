//go:build darwin

package desktop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDarwinHUDAutoStartWritesLaunchAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	manager := NewAutoStartManagerWithOptions(AutoStartOptions{
		AppName:  "Nexus HUD",
		AppID:    "com.iapro.nexus.hud",
		ExecPath: "/Applications/Nexus HUD.app/Contents/MacOS/nexus-hud",
	})
	if err := manager.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, "Library", "LaunchAgents", "com.iapro.nexus.hud.plist")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "<key>RunAtLoad</key>") || !strings.Contains(text, manager.ExecPath) {
		t.Fatalf("unexpected launch agent: %s", text)
	}
	if err := manager.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("launch agent was not removed: %v", err)
	}
}
