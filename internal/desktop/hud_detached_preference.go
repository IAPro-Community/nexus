package desktop

import (
	"fmt"
	"runtime"
)

// GetHUDDetachedPopoverPreference reports the persisted opt-in for the
// independent native popup. The popup is what makes the mascot presentation
// available, so the setting is separate from the runtime capability.
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
	if enabled {
		if err := h.EnsureDetachedPopup(); err != nil {
			return err
		}
	} else {
		h.closeDetachedHUD()
		h.setDetachedEnabled(false)
	}

	state := h.manager.Load()
	state.DetachedPopover = enabled
	// The legacy path grows the single window into the panel. Switching to the
	// independent popup must shrink the capsule back, keeping its bottom-right
	// anchor, otherwise the transparent expanded window would stay behind.
	state.Width, state.Height = hudCompactDimensions(state)
	if err := h.manager.Save(state); err != nil {
		if enabled {
			h.closeDetachedHUD()
			h.setDetachedEnabled(false)
		}
		return err
	}
	// Both directions return to a compact capsule before exposing the new
	// expansion strategy. Prevent a stale legacy expanded state from being
	// mistaken for an already open detached popup.
	h.setHUDExpandedState(false)
	if window, ok := h.nativeWindow(); ok {
		currentWidth, currentHeight := window.Size()
		if currentWidth != state.Width || currentHeight != state.Height {
			positionX, positionY := window.Position()
			window.SetSize(state.Width, state.Height)
			if h.GetHUDCapabilities().GlobalPositioning {
				window.SetPosition(
					positionX+currentWidth-state.Width,
					positionY+currentHeight-state.Height,
				)
			}
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
