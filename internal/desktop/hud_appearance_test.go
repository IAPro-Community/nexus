package desktop

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestHUDAppearanceNormalizationAndPersistence(t *testing.T) {
	manager := NewHUDWindowStateManager(filepath.Join(t.TempDir(), "hud.json"))
	if got := manager.Load().Appearance; got != HUDAppearanceMascot {
		t.Fatalf("new installs must display the Guardian Mascot: %s", got)
	}
	if !manager.Load().DetachedPopover {
		t.Fatal("new installations should prefer the detached popup automatically")
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
		if actual := manager.Load().Appearance; actual != HUDAppearanceMascot {
			t.Fatalf("%q must recover to the Guardian default: %q", invalid, actual)
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

func TestDetachedPopoverPreferencePersistsAndFailsClosed(t *testing.T) {
	manager := NewHUDWindowStateManager(filepath.Join(t.TempDir(), "hud.json"))
	controller := NewHUDController(manager)
	if !controller.GetHUDDetachedPopoverPreference() {
		t.Fatal("the independent popup should be preferred without an environment variable")
	}
	if err := controller.SetHUDDetachedPopoverPreference(true); err == nil {
		t.Fatal("enabling without a native window factory must fail closed")
	}
	if controller.GetHUDDetachedPopoverEnabled() || !manager.Load().DetachedPopover {
		t.Fatal("failed native enable must not change the persisted default")
	}

	state := manager.Load()
	state.DetachedPopover = true
	state.Appearance = HUDAppearanceMascot
	state.Width, state.Height = HUDMascotWidth, HUDMascotHeight
	if err := manager.Save(state); err != nil {
		t.Fatal(err)
	}
	detached := NewHUDControllerWithDetachedPopover(manager, true)
	if !detached.GetHUDDetachedPopoverPreference() {
		t.Fatal("the opt-in must survive a restart")
	}
	if err := detached.SetHUDDetachedPopoverPreference(false); err != nil {
		t.Fatal(err)
	}
	got := manager.Load()
	if got.DetachedPopover {
		t.Fatal("disabling must persist the opt-out")
	}
	if got.Appearance != HUDAppearanceMascot {
		t.Fatalf("disabling the popup must preserve the chosen Mascot, got %q", got.Appearance)
	}
	if got.Width != HUDMascotWidth || got.Height != HUDMascotHeight {
		t.Fatalf("single-window Mascot must remain compact: %+v", got)
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
	if manager.Load().Appearance != HUDAppearanceMascot {
		t.Fatal("single-window fallback must preserve the Mascot choice")
	}
	if runtime.GOOS == "linux" && hudSession() == "wayland" &&
		NewHUDControllerWithDetachedPopover(manager, true).GetHUDDetachedPopoverEnabled() {
		t.Fatal("Wayland cannot force unsupported global popup positioning")
	}
}

func TestExplicitBarPreferenceSurvivesMascotDefault(t *testing.T) {
	manager := NewHUDWindowStateManager(filepath.Join(t.TempDir(), "hud.json"))
	state := manager.Load()
	state.Appearance = HUDAppearanceBar
	state.Width, state.Height = state.CompactWidth, state.CompactHeight
	state.DetachedPopover = false
	if err := manager.Save(state); err != nil {
		t.Fatal(err)
	}
	controller := NewHUDControllerWithDetachedPopover(manager, false)
	if controller.GetHUDAppearance() != HUDAppearanceBar {
		t.Fatal("explicitly saved BAR must never be replaced with MASCOT")
	}
	if manager.Load().DetachedPopover {
		t.Fatal("explicit opt-out of the independent popup must persist")
	}
	restored := compactHUDWindowState(manager.Load())
	if restored.Width != state.CompactWidth || restored.Height != state.CompactHeight {
		t.Fatalf("BAR compact geometry must survive a restart: %+v", restored)
	}
}

