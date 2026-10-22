package desktop

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// detachedHUDPopupState anchors the detail window above the actual compact
// capsule without ever resizing that capsule. It prefers below the capsule
// when there is insufficient vertical room above, and clamps to the work area.
// This geometry is pure so multi-monitor and small displays are testable.
func detachedHUDPopupState(state HUDWindowState, capsule, work application.Rect) HUDWindowState {
	state = normalizeHUDWindowState(state)
	state.Width, state.Height = state.ExpandedWidth, state.ExpandedHeight
	state.X = capsule.X + capsule.Width - state.Width
	state.Y = capsule.Y - state.Height - 8
	if work.Width > 0 && work.Height > 0 && state.Y < work.Y {
		state.Y = capsule.Y + capsule.Height + 8
	}
	if work.Width > 0 && work.Height > 0 {
		state = clampHUDWindowStateToWorkArea(state, work)
	}
	return state
}

// positionDetachedHUDPopup is called only after the Wails application has
// created both windows. A missing popup fails closed, preserving the capsule.
func (h *HUDController) positionDetachedHUDPopup(capsule application.Window) error {
	popup, ok := h.detachedPopupWindow()
	if !ok {
		return fmt.Errorf("detached Nexus HUD popup is unavailable")
	}
	state := h.manager.Load()
	x, y := capsule.Position()
	width, height := capsule.Size()
	work := application.Rect{}
	capsuleRect := application.Rect{X: x, Y: y, Width: width, Height: height}
	// Anchor to the monitor currently containing the capsule, not a stale
	// persisted DisplayID from another monitor or an earlier session.
	if app := application.Get(); app != nil {
		if screen := app.Screen.ScreenNearestDipRect(capsuleRect); screen != nil {
			work = screen.WorkArea
		}
	}
	geometry := detachedHUDPopupState(state, capsuleRect, work)
	popup.SetSize(geometry.Width, geometry.Height)
	if h.GetHUDCapabilities().GlobalPositioning {
		popup.SetPosition(geometry.X, geometry.Y)
	}
	popup.SetAlwaysOnTop(state.AlwaysOnTop)
	return nil
}

// setHUDDetachedExpanded uses an independently rendered Wails window.
// The compact capsule remains at its own size and position throughout.
func (h *HUDController) setHUDDetachedExpanded(expanded bool) error {
	capsule, ok := h.nativeWindow()
	if !ok {
		return fmt.Errorf("native Nexus HUD capsule is unavailable")
	}
	popup, ok := h.detachedPopupWindow()
	if !ok {
		return fmt.Errorf("native Nexus HUD popup is unavailable")
	}
	if !expanded {
		h.setHUDExpandedState(false)
		popup.Hide()
		return nil
	}
	if err := h.positionDetachedHUDPopup(capsule); err != nil {
		return err
	}
	// Set this before focusing the popup: the capsule's WindowLostFocus
	// handler must not auto-hide the capsule during the transition.
	h.setHUDExpandedState(true)
	popup.Show()
	popup.Restore()
	popup.Focus()
	return nil
}

// In detached mode presets resize the compact capsule and then reposition
// the independent popup if it happens to be visible. The capsule never
// changes to the popup's larger dimensions.
func (h *HUDController) setDetachedHUDSizePreset(preset string) error {
	capsule, ok := h.nativeWindow()
	if !ok {
		return fmt.Errorf("native Nexus HUD capsule is unavailable")
	}
	if _, ok := h.detachedPopupWindow(); !ok {
		return fmt.Errorf("native Nexus HUD popup is unavailable")
	}
	selected, ok := resolveHUDSizePreset(preset)
	if !ok {
		return fmt.Errorf("unknown HUD size preset %q", preset)
	}
	state := h.manager.Load()
	state.CompactWidth, state.CompactHeight = selected.CompactWidth, selected.CompactHeight
	state.ExpandedWidth, state.ExpandedHeight = selected.ExpandedWidth, selected.ExpandedHeight
	state.X, state.Y = capsule.Position()
	oldWidth, oldHeight := capsule.Size()
	compactWidth, compactHeight := hudCompactDimensions(state)
	state.X += oldWidth - compactWidth
	state.Y += oldHeight - compactHeight
	state.Width, state.Height = compactWidth, compactHeight
	if screen := h.screenForState(state, capsule); screen != nil {
		state = clampHUDWindowStateToWorkArea(state, screen.WorkArea)
		state.DisplayID = screen.ID
		state.DisplayScale = screen.ScaleFactor
	}
	capsule.SetSize(state.Width, state.Height)
	if h.GetHUDCapabilities().GlobalPositioning {
		capsule.SetPosition(state.X, state.Y)
	}
	state.Placement = HUDPlacementCustom
	if err := h.manager.Save(state); err != nil {
		return err
	}
	if h.GetHUDExpanded() {
		return h.positionDetachedHUDPopup(capsule)
	}
	return nil
}
