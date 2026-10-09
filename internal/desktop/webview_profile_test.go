package desktop

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestWebviewUserDataPathIsIsolatedBySurface(t *testing.T) {
	cacheRoot := filepath.Join(t.TempDir(), "local-cache")

	desktopPath := webviewUserDataPath(cacheRoot, "desktop")
	hudPath := webviewUserDataPath(cacheRoot, "hud")

	if want := filepath.Join(cacheRoot, "IAPro", "Nexus", "desktop", "WebView2"); desktopPath != want {
		t.Fatalf("desktop WebView2 path = %q, want %q", desktopPath, want)
	}
	if want := filepath.Join(cacheRoot, "IAPro", "Nexus", "hud", "WebView2"); hudPath != want {
		t.Fatalf("HUD WebView2 path = %q, want %q", hudPath, want)
	}
	if desktopPath == hudPath {
		t.Fatal("Desktop and HUD must not share a WebView2 user data folder")
	}
}

func TestWebviewUserDataPathRejectsMissingRootAndUnknownSurface(t *testing.T) {
	if got := webviewUserDataPath("", "desktop"); got != "" {
		t.Fatalf("missing cache root returned %q", got)
	}
	if got := webviewUserDataPath(t.TempDir(), "unknown"); got != "" {
		t.Fatalf("unknown surface returned %q", got)
	}
}

func TestWindowsWebviewUserDataPathUsesLocalAppData(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows local cache mapping")
	}

	cacheRoot := t.TempDir()
	t.Setenv("LOCALAPPDATA", cacheRoot)

	want := filepath.Join(cacheRoot, "IAPro", "Nexus", "desktop", "WebView2")
	if got := WindowsWebviewUserDataPath("Desktop"); got != want {
		t.Fatalf("Windows desktop WebView2 path = %q, want %q", got, want)
	}
}
