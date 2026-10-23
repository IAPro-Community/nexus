package desktop

import "fmt"

// Guardian presentation is orthogonal to the existing capacity/agent/mission
// information priority (HUDWindowState.Mode). Never overload that field.
func validHUDAppearance(value string) bool {
	switch value {
	case HUDAppearanceBar, HUDAppearanceMascot, HUDAppearanceTray:
		return true
	default:
		return false
	}
}

func hudCompactDimensions(state HUDWindowState) (int, int) {
	if state.Appearance == HUDAppearanceMascot {
		return HUDMascotWidth, HUDMascotHeight
	}
	return state.CompactWidth, state.CompactHeight
}

// GetHUDAppearance is served from the single persisted Wails controller.
// The two independent WebViews should not rely on localStorage events to sync.
func (h *HUDController) GetHUDAppearance() string {
	return h.manager.Load().Appearance
}

// SetHUDAppearance immediately applies a persisted Guardian presentation. The
// tray-only variant hides the window but is recoverable from the system tray.
func (h *HUDController) SetHUDAppearance(appearance string) error {
	if !validHUDAppearance(appearance) {
		return fmt.Errorf("invalid Nexus HUD appearance %q", appearance)
	}
	// Presentation and expansion strategy are independent. The Mascot works
	// with the detached detail window or the single-window fallback (Wayland).
	window, ok := h.nativeWindow()
	if !ok {
		return fmt.Errorf("native Nexus HUD window is unavailable")
	}
	h.closeDetachedHUD()
	state := h.manager.Load()
	oldWidth, oldHeight := window.Size()
	state.X, state.Y = window.Position()
	state.Appearance = appearance
	targetWidth, targetHeight := hudCompactDimensions(state)
	state.X, state.Y = hudResizeAnchor(state.X, state.Y, oldWidth, oldHeight, targetWidth, targetHeight, h.GetHUDCapabilities().GlobalPositioning)
	state.Width, state.Height = targetWidth, targetHeight
	if screen := h.screenForState(state, window); screen != nil {
		state = clampHUDWindowStateToWorkArea(state, screen.WorkArea)
		state.DisplayID, state.DisplayScale = screen.ID, screen.ScaleFactor
	}
	state.Placement = HUDPlacementCustom
	if !h.GetHUDCapabilities().GlobalPositioning {
		state.X, state.Y = window.Position()
	}
	if err := h.manager.Save(state); err != nil {
		return err
	}
	// Switching presentations always closes the previous details panel,
	// including the single-window fallback where no detached popup exists.
	h.setHUDExpandedState(false)
	window.SetSize(state.Width, state.Height)
	if h.GetHUDCapabilities().GlobalPositioning {
		window.SetPosition(state.X, state.Y)
	}
	if appearance == HUDAppearanceTray {
		window.Hide()
		return nil
	}
	window.Show()
	return nil
}
