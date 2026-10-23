//go:build darwin && !cgo

package desktop

import "github.com/wailsapp/wails/v3/pkg/application"

// Without cgo/Quartz there is no reliable global cursor location.
// Never silently enable an irreversible click-through native surface.
func hudHitTestingSupported() bool { return false }

func hudPointerWithinRegions(_ application.Window, _ []HUDHitRegion) bool { return true }
