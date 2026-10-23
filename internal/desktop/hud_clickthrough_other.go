//go:build !windows && !darwin

package desktop

import "github.com/wailsapp/wails/v3/pkg/application"

// Linux does not implement Wails SetIgnoreMouseEvents. Keep its
// native input handling intact until an X11 input-shape adapter is certified.
func hudHitTestingSupported() bool { return false }

func hudPointerWithinRegions(_ application.Window, _ []HUDHitRegion) bool { return true }
