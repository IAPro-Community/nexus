package desktop

import (
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
	if !strings.Contains(text, `Exec="/tmp/IAPro Nexus/next\\\"release/nexus" --minimized`) {
		t.Fatalf("autostart executable was not safely quoted: %q", text)
	}
}

func TestCapabilities(t *testing.T) {
	app := NewApp(nil, nil)
	caps := app.GetCapabilities()
	if !caps.Native || !caps.WindowManagement {
		t.Fatalf("desktop core capabilities must be available, got %+v", caps)
	}
	if caps.Tray || caps.NativeMenus || caps.DeepLinks || caps.AutoStart {
		t.Fatalf("unimplemented capabilities must not be advertised, got %+v", caps)
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
