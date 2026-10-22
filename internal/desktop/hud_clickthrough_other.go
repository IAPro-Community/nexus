//go:build !windows

package desktop

import "github.com/wailsapp/wails/v3/pkg/application"

// Click-through is only certified on Windows. Other platforms keep their
// current native behavior until they have equivalent exact-SHA evidence.
func hudHitTestingSupported() bool { return false }

func hudPointerWithinRegions(_ application.Window, _ []HUDHitRegion) bool { return true }
