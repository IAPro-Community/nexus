package desktop

import (
	"fmt"
	"runtime"
)

// GetHUDDetachedPopoverPreference reports the persisted opt-in for the
// independent native popup. The Mascot works with either the detached popup
// or the single-window fallback; preference and capability stay separate.
func (h *HUDController) GetHUDDetachedPopoverPreference() bool {
	return h.manager.Load().DetachedPopover
}

// SetHUDDetachedPopoverPreference persists and applies the user's choice.
//
// Enabling builds the second native window on demand, without a restart.
// Disabling hides it and switches to single-window expansion without changing
// whether the user selected Mascot or Bar. Both paths fail closed: if
// the window cannot be created, the preference is not persisted and the HUD
// stays exactly as it was.
func (h *HUDController) SetHUDDetachedPopoverPreference(enabled bool) error {
	wasEnabled := h.GetHUDDetachedPopoverEnabled()
	if enabled && !wasEnabled {
		if err := h.EnsureDetachedPopup(); err != nil {
			return err
		}
	}

	state := h.manager.Load()
	state.DetachedPopover = enabled
	// Preserve the actual bottom-right anchor before changing the native size.
	// Saving only compact dimensions with the old expanded coordinates would
	// move the Guardian after a restart.
	window, hasWindow := h.nativeWindow()
	if hasWindow {
		state.X, state.Y = window.Position()
		state.Width, state.Height = window.Size()
		state = compactHUDWindowState(state)
		if screen := h.screenForState(state, window); screen != nil {
			state = clampHUDWindowStateToWorkArea(state, screen.WorkArea)
			state.DisplayID, state.DisplayScale = screen.ID, screen.ScaleFactor
		}
		state.Placement = HUDPlacementCustom
		if !h.GetHUDCapabilities().GlobalPositioning {
			state.X, state.Y = window.Position()
		}
	} else {
		state.Width, state.Height = hudCompactDimensions(state)
	}
	if err := h.manager.Save(state); err != nil {
		if enabled && !wasEnabled {
			h.closeDetachedHUD()
			h.setDetachedEnabled(false)
		}
		return err
	}
	// Do not discard a working popup or its in-memory state until persistence
	// succeeds. A disk error must leave the current presentation recoverable.
	if !enabled {
		h.closeDetachedHUD()
		h.setDetachedEnabled(false)
	}
	// Both directions return to a compact capsule before exposing the new
	// expansion strategy. Prevent stale legacy expansion state from being
	// mistaken for an already open detached popup.
	h.setHUDExpandedState(false)
	if hasWindow {
		window.SetSize(state.Width, state.Height)
		if h.GetHUDCapabilities().GlobalPositioning {
			window.SetPosition(state.X, state.Y)
		}
	}
	return nil
}

// EnsureDetachedPopup creates the popup exactly once through the factory
// installed by the HUD binary. Startup and the settings toggle share it.
func (h *HUDController) EnsureDetachedPopup() error {
	// Runtime toggles must obey the same constraint as startup; Wayland does
	// not allow portable absolute popup positioning with this Wails backend.
	if runtime.GOOS == "linux" && hudSession() == "wayland" {
		return fmt.Errorf("independent Nexus HUD popup is unsupported on Wayland")
	}
	if h.GetHUDDetachedPopoverEnabled() {
		return nil
	}
	h.detachedMu.Lock()
	factory := h.popupFactory
	h.detachedMu.Unlock()
	if factory == nil {
		return fmt.Errorf("independent Nexus HUD popup is unavailable on this platform")
	}
	popup, err := factory()
	if err != nil {
		return err
	}
	if popup == nil {
		return fmt.Errorf("independent Nexus HUD popup could not be created")
	}
	h.setDetachedEnabled(true)
	return nil
}
