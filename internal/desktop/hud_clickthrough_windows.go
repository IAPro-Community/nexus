//go:build windows

package desktop

import (
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	hudUser32           = syscall.NewLazyDLL("user32.dll")
	hudProcGetCursorPos = hudUser32.NewProc("GetCursorPos")
)

type hudNativePoint struct {
	X int32
	Y int32
}

// hudCursorPosition returns the global cursor position in physical pixels.
// x/sys/windows does not expose GetCursorPos, so the call is made directly.
func hudCursorPosition() (int, int, bool) {
	var point hudNativePoint
	ok, _, _ := hudProcGetCursorPos.Call(uintptr(unsafe.Pointer(&point)))
	if ok == 0 {
		return 0, 0, false
	}
	return int(point.X), int(point.Y), true
}

func hudHitTestingSupported() bool { return true }

// hudPointerWithinRegions reads the global cursor position (physical pixels)
// and compares it against the window geometry (DIPs) plus the renderer-reported
// CSS rectangles.
func hudPointerWithinRegions(window application.Window, regions []HUDHitRegion) bool {
	physicalX, physicalY, ok := hudCursorPosition()
	if !ok {
		return true
	}
	dip := application.PhysicalToDipPoint(application.Point{X: physicalX, Y: physicalY})
	x, y := window.Position()
	inside := hudPointerWithinRegionsDip(dip.X, dip.Y, x, y, regions)
	hudDebugf("hud-hit: cursorDIP=(%d,%d) window=(%d,%d) regions=%d inside=%v", dip.X, dip.Y, x, y, len(regions), inside)
	return inside
}
