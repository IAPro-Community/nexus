package desktop

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDeepLinkParser(t *testing.T) {
	cases := []struct {
		url          string
		wantResource string
		wantID       string
		wantErr      bool
	}{
		{"nexus://project/proj-123", "project", "proj-123", false},
		{"nexus://mission/msn-456", "mission", "msn-456", false},
		{"nexus://agent/agt-789?tab=terminal", "agent", "agt-789", false},
		{"http://project/123", "", "", true},
		{"nexus://unknown/123", "", "", true},
		{"nexus://project/", "", "", true},
	}

	for _, c := range cases {
		action, err := ParseDeepLink(c.url)
		if c.wantErr {
			if err == nil {
				t.Fatalf("url %s: expected error, got nil", c.url)
			}
			continue
		}
		if err != nil {
			t.Fatalf("url %s unexpected error: %v", c.url, err)
		}
		if action.Resource != c.wantResource || action.ID != c.wantID {
			t.Fatalf("url %s: got %+v, want resource=%s id=%s", c.url, action, c.wantResource, c.wantID)
		}
	}
}

func TestDeepLinkParserRejectsAmbiguousOrUnsafeTargets(t *testing.T) {
	tooLong := "nexus://project/" + string(make([]byte, maxDeepLinkIDSize+1))
	for _, raw := range []string{
		"nexus://project/proj-123/extra",
		"nexus://project/../secrets",
		"nexus://project/id%2Fwith-slash",
		"nexus://project/id%20with-space",
		"nexus://project/id#fragment",
		tooLong,
	} {
		if _, err := ParseDeepLink(raw); err == nil {
			t.Fatalf("expected unsafe or ambiguous deep link %q to be rejected", raw)
		}
	}
}

func TestWindowStateManager(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "window-state.json")
	mgr := NewWindowStateManager(stateFile)

	initial := mgr.Load()
	if initial.Width < 640 || initial.Height < 480 {
		t.Fatalf("expected valid default window dimensions, got %+v", initial)
	}

	custom := WindowState{
		Width:     1440,
		Height:    900,
		X:         100,
		Y:         100,
		Maximized: true,
	}
	if err := mgr.Save(custom); err != nil {
		t.Fatalf("save window state failed: %v", err)
	}

	loaded := mgr.Load()
	if loaded.Width != 1440 || loaded.Height != 900 || !loaded.Maximized {
		t.Fatalf("loaded state does not match saved state: %+v", loaded)
	}
}

func TestAutoStartQuotesExecutablePath(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("desktop entry autostart is Linux-specific")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := `/tmp/IAPro Nexus/next\"release/nexus`
	manager := NewAutoStartManager("IAPro Nexus", path)
	if err := manager.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".config", "autostart", "iapro-nexus.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `Exec="/tmp/IAPro Nexus/next\\\"release/nexus" "--minimized"`) {
		t.Fatalf("autostart executable was not safely quoted: %q", text)
	}
}

func TestCapabilitiesFailClosedBeforeWailsStartup(t *testing.T) {
	app := NewApp(nil, nil)
	caps := app.GetCapabilities()
	if !caps.Native || !caps.WindowManagement {
		t.Fatalf("desktop core capabilities must be available, got %+v", caps)
	}
	if caps.Tray || caps.NativeMenus || caps.DeepLinks || caps.AutoStart || caps.FilePicker || caps.FolderPicker {
		t.Fatalf("runtime-backed capabilities must remain false before Wails startup, got %+v", caps)
	}
}

func TestOpenExternalRejectsUnsafeSchemes(t *testing.T) {
	app := NewApp(nil, nil)
	for _, raw := range []string{"javascript:alert(1)", "file:///tmp/a", "data:text/plain,hello", "not a url"} {
		if err := app.OpenExternal(raw); err == nil {
			t.Fatalf("expected unsafe URL %q to be rejected", raw)
		}
	}
}

func TestAutoStartOptionsHaveIndependentIdentity(t *testing.T) {
	manager := NewAutoStartManagerWithOptions(AutoStartOptions{
		AppName:  "Nexus HUD",
		AppID:    "com.iapro.nexus.hud",
		ExecPath: "/tmp/nexus-hud",
	})
	if manager.AppID != "com.iapro.nexus.hud" {
		t.Fatalf("AppID=%q", manager.AppID)
	}
	if len(manager.Args) != 0 {
		t.Fatalf("HUD autostart must not inherit desktop args: %v", manager.Args)
	}
	if got := sanitizeAutoStartID(" unsafe/id "); got != "unsafe-id" {
		t.Fatalf("sanitizeAutoStartID=%q", got)
	}
}

func TestEmbeddedBrandAndHUDIcons(t *testing.T) {
	nexusApp := NexusAppIcon()
	hudApp := NexHUDAppIcon()
	hudTray := NexHUDTrayIcon()

	nexusCfg, err := png.DecodeConfig(bytes.NewReader(nexusApp))
	if err != nil {
		t.Fatalf("decode NexusAppIcon: %v", err)
	}
	if nexusCfg.Width != 256 || nexusCfg.Height != 256 {
		t.Fatalf("NexusAppIcon dimensions = %dx%d, want 256x256", nexusCfg.Width, nexusCfg.Height)
	}

	hudAppCfg, err := png.DecodeConfig(bytes.NewReader(hudApp))
	if err != nil {
		t.Fatalf("decode NexHUDAppIcon: %v", err)
	}
	if hudAppCfg.Width != 256 || hudAppCfg.Height != 256 {
		t.Fatalf("NexHUDAppIcon dimensions = %dx%d, want 256x256", hudAppCfg.Width, hudAppCfg.Height)
	}
	if bytes.Equal(nexusApp, hudApp) {
		t.Fatal("NexHUDAppIcon must use the NEX agent icon, not NexusAppIcon")
	}

	hudTrayCfg, err := png.DecodeConfig(bytes.NewReader(hudTray))
	if err != nil {
		t.Fatalf("decode NexHUDTrayIcon: %v", err)
	}
	if hudTrayCfg.Width != 32 || hudTrayCfg.Height != 32 {
		t.Fatalf("NexHUDTrayIcon dimensions = %dx%d, want 32x32", hudTrayCfg.Width, hudTrayCfg.Height)
	}
}
