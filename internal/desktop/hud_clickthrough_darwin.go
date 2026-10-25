//go:build darwin && cgo

package desktop

/*
#cgo LDFLAGS: -framework CoreGraphics
#include <CoreGraphics/CoreGraphics.h>

// Quartz provides global display-space coordinates even when the HUD has
// temporarily opted out of native mouse events. The values are logical
// display points, matching Wails' macOS window coordinates.
static int nexusHUDCursorPoint(double *x, double *y) {
    CGEventRef event = CGEventCreate(NULL);
    if (event == NULL) {
        return 0;
    }
    CGPoint p = CGEventGetLocation(event);
    CFRelease(event);
    *x = p.x;
    *y = p.y;
    return 1;
}
*/
import "C"

import (
	"math"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// This is opt-in until Retina/multi-monitor and click-through have been
// tested using the macOS native acceptance harness on the exact source SHA.
// Wails v3 supports ignoring native mouse events on macOS, but correct cursor
// position mapping still needs real hardware evidence.
func hudHitTestingSupported() bool {
	return os.Getenv("NEXUS_HUD_MAC_CLICKTHROUGH") == "1"
}

func applyHUDHitRegions(_ application.Window, _ []HUDHitRegion) {}

func resetHUDHitRegions(_ application.Window) {}

func hudPointerWithinRegions(window application.Window, regions []HUDHitRegion) bool {
	var x, y C.double
	if C.nexusHUDCursorPoint(&x, &y) == 0 {
		return true // Fail open: the user must be able to recover the HUD.
	}
	cursorX, cursorY := float64(x), float64(y)
	if math.IsNaN(cursorX) || math.IsNaN(cursorY) || math.IsInf(cursorX, 0) || math.IsInf(cursorY, 0) {
		return true
	}
	windowX, windowY := window.Position()
	return hudPointerWithinRegionsDip(int(math.Round(cursorX)), int(math.Round(cursorY)), windowX, windowY, regions)
}
