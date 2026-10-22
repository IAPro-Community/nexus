package desktop

import (
	"log"
	"os"
	"time"
)

// hudHitDebug keeps the 30ms hit-test trace opt-in; normal runs stay quiet.
var hudHitDebug = os.Getenv("NEXUS_HUD_DEBUG_HIT") == "1"

func hudDebugf(format string, args ...any) {
	if hudHitDebug {
		log.Printf(format, args...)
	}
}

// HUDHitRegion is a rectangle in CSS pixels relative to the top-left corner of
// the HUD webview that must keep receiving native mouse input.
//
// The HUD window is larger than the visible capsule (and, when expanded, the
// panel). Everything outside the union of the reported rectangles is a
// transparent surface: the native window must let those clicks reach whatever
// is behind it. CSS `pointer-events` alone cannot do that because the operating
// system still routes the click to the top-most native window.
type HUDHitRegion struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

const (
	hudHitPollInterval = 30 * time.Millisecond
	hudHitRegionLimit  = 512
)

// SetHUDHitRegions receives the interactive rectangles of the HUD surface from
// the renderer. Calling it starts the native watcher on platforms that support
// click-through.
func (h *HUDController) SetHUDHitRegions(regions []HUDHitRegion) error {
	if len(regions) > hudHitRegionLimit {
		regions = regions[:hudHitRegionLimit]
	}
	clean := make([]HUDHitRegion, 0, len(regions))
	for _, region := range regions {
		if region.Width <= 0 || region.Height <= 0 {
			continue
		}
		clean = append(clean, region)
	}
	h.hitMu.Lock()
	h.hitRegions = clean
	h.hitMu.Unlock()
	hudDebugf("hud-hit: received %d interactive regions", len(clean))
	h.startHUDHitWatcher()
	return nil
}

func (h *HUDController) startHUDHitWatcher() {
	if !hudHitTestingSupported() {
		return
	}
	h.hitOnce.Do(func() {
		stop := make(chan struct{})
		h.hitMu.Lock()
		h.hitStop = stop
		h.hitMu.Unlock()
		go h.watchHUDHitTesting(stop)
	})
}

func (h *HUDController) stopHUDHitWatcher() {
	h.hitMu.Lock()
	stop := h.hitStop
	h.hitStop = nil
	h.hitMu.Unlock()
	if stop != nil {
		close(stop)
	}
}

func (h *HUDController) watchHUDHitTesting(stop <-chan struct{}) {
	ticker := time.NewTicker(hudHitPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			h.resetHUDHitTesting()
			return
		case <-ticker.C:
			h.applyHUDHitTesting()
		}
	}
}

func (h *HUDController) applyHUDHitTesting() {
	if h.detached {
		// The detached popover is a second native window with its own focus
		// lifecycle. Click-through is only proven for the capsule window, so
		// the split presentation keeps its previous behavior.
		return
	}
	window, ok := h.nativeWindow()
	if !ok || !window.IsVisible() {
		return
	}
	h.hitMu.RLock()
	regions := h.hitRegions
	h.hitMu.RUnlock()

	ignore := !hudPointerWithinRegions(window, regions)
	h.hitMu.Lock()
	changed := ignore != h.hitIgnoring
	h.hitIgnoring = ignore
	h.hitMu.Unlock()
	if !changed {
		return
	}
	window.SetIgnoreMouseEvents(ignore)
}

// resetHUDHitTesting restores normal mouse handling before the window closes so
// a later process never inherits a click-through capsule.
func (h *HUDController) resetHUDHitTesting() {
	h.hitMu.Lock()
	ignoring := h.hitIgnoring
	h.hitIgnoring = false
	h.hitMu.Unlock()
	if !ignoring {
		return
	}
	if window, ok := h.nativeWindow(); ok {
		window.SetIgnoreMouseEvents(false)
	}
}

// hudPointerWithinRegionsDip is the testable core of the hit test. Cursor and
// window geometry are in DIPs; regions are CSS pixels, which match DIPs at the
// HUD's fixed 1.0 zoom.
//
// With no regions the window fails open: the HUD stays fully interactive until
// the renderer reports its layout, so a slow or broken bridge can never trap
// the cursor inside an invisible click-through window.
func hudPointerWithinRegionsDip(cursorX, cursorY, windowX, windowY int, regions []HUDHitRegion) bool {
	if len(regions) == 0 {
		return true
	}
	relX := float64(cursorX - windowX)
	relY := float64(cursorY - windowY)
	for _, region := range regions {
		if relX >= region.X && relX <= region.X+region.Width && relY >= region.Y && relY <= region.Y+region.Height {
			return true
		}
	}
	return false
}
