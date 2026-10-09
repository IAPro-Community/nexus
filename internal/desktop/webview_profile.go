package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// WindowsWebviewUserDataPath returns a per-surface WebView2 profile under the
// user's local cache directory. An empty result preserves Wails' platform default.
func WindowsWebviewUserDataPath(surface string) string {
	if runtime.GOOS != "windows" {
		return ""
	}

	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return webviewUserDataPath(cacheRoot, surface)
}

func webviewUserDataPath(cacheRoot, surface string) string {
	cacheRoot = strings.TrimSpace(cacheRoot)
	surface = strings.ToLower(strings.TrimSpace(surface))
	if cacheRoot == "" || (surface != "desktop" && surface != "hud") {
		return ""
	}

	return filepath.Join(cacheRoot, "IAPro", "Nexus", surface, "WebView2")
}
