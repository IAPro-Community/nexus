package desktop

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestHUDAppearanceNormalizationAndPersistence(t *testing.T) {
	manager := NewHUDWindowStateManager(filepath.Join(t.TempDir(), "hud.json"))
	if got := manager.Load().Appearance; got != HUDAppearanceBar {
		t.Fatalf("new installs should use the original bar: %s", got)
	}
	state := manager.Load()
	state.Appearance = HUDAppearanceMascot
	state.Mode = "RESOURCE"
	state.Width, state.Height = HUDMascotWidth, HUDMascotHeight
	if err := manager.Save(state); err != nil {
		t.Fatal(err)
	}
	got := manager.Load()
	if got.Appearance != HUDAppearanceMascot || got.Mode != "RESOURCE" ||
		got.Width != HUDMascotWidth || got.Height != HUDMascotHeight {
		t.Fatalf("appearance must persist independently of information mode: %+v", got)
	}
	for _, invalid := range []string{"", "unknown", "MASCOT\n"} {
		state.Appearance = invalid
		if err := manager.Save(state); err != nil {
			t.Fatal(err)
		}
		if actual := manager.Load().Appearance; actual != HUDAppearanceBar {
			t.Fatalf("%q must recover to bar: %q", invalid, actual)
		}
	}
}

func TestGuardianCompactFootprintPreservesScreenAnchor(t *testing.T) {
	state := defaultHUDWindowState()
	state.Appearance = HUDAppearanceMascot
	state.X, state.Y = 700, 480
	state.Width, state.Height = 220, 60
	compact := compactHUDWindowState(state)
	if compact.Width != HUDMascotWidth || compact.Height != HUDMascotHeight {
		t.Fatalf("mascot should occupy its own small window: %+v", compact)
	}
	if compact.X+compact.Width != 920 || compact.Y+compact.Height != 540 {
		t.Fatalf("mascot lost bottom-right anchor: %+v", compact)
	}
	onceMore := compactHUDWindowState(compact)
	if onceMore != compact {
		t.Fatalf("mascot geometry must be idempotent: %+v != %+v", compact, onceMore)
	}
}

func TestGuardianTrayModeAndFallback(t *testing.T) {
	manager := NewHUDWindowStateManager(filepath.Join(t.TempDir(), "hud.json"))
	state := manager.Load()
	state.Appearance = HUDAppearanceTray
	if err := manager.Save(state); err != nil {
		t.Fatal(err)
	}
	controller := NewHUDControllerWithDetachedPopover(manager, true)
	if controller.GetHUDAppearance() != HUDAppearanceTray {
		t.Fatal("tray-only preference should persist when there is a native tray")
	}
	if err := controller.SetHUDAppearance("WRONG"); err == nil {
		t.Fatal("invalid modes must not change the native HUD")
	}
	if err := controller.SetHUDAppearance(HUDAppearanceBar); err == nil {
		t.Fatal("cannot report successful native presentation without a window")
	}
	if controller.GetHUDAppearance() != HUDAppearanceTray {
		t.Fatal("failed native change must not overwrite persisted preference")
	}

	state.Appearance = HUDAppearanceMascot
	state.Width, state.Height = HUDMascotWidth, HUDMascotHeight
	if err := manager.Save(state); err != nil {
		t.Fatal(err)
	}
	legacy := NewHUDController(manager)
	if legacy.GetHUDDetachedPopoverEnabled() {
		t.Fatal("legacy fallback must not enable a second native window")
	}
	if manager.Load().Appearance != HUDAppearanceBar {
		t.Fatal("legacy fallback must recover to a visible full-sized bar")
	}
	if runtime.GOOS == "linux" && hudSession() == "wayland" &&
		NewHUDControllerWithDetachedPopover(manager, true).GetHUDDetachedPopoverEnabled() {
		t.Fatal("Wayland cannot force unsupported global popup positioning")
	}
}
