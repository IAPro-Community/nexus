package desktop

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type fakeDetachedPopupPresenter struct {
	operations []string
}

func (f *fakeDetachedPopupPresenter) Show() application.Window {
	f.operations = append(f.operations, "show")
	return nil
}

func (f *fakeDetachedPopupPresenter) Hide() application.Window {
	f.operations = append(f.operations, "hide")
	return nil
}

func (f *fakeDetachedPopupPresenter) Focus() {
	f.operations = append(f.operations, "focus")
}

func TestDetachedPopupPresentationShowsBeforeApplyingNativeGeometry(t *testing.T) {
	popup := &fakeDetachedPopupPresenter{}

	err := presentDetachedHUDPopup(popup, func() error {
		if len(popup.operations) != 1 || popup.operations[0] != "show" {
			t.Fatalf("popup must be realized before geometry is applied, got %v", popup.operations)
		}
		popup.operations = append(popup.operations, "geometry")
		return nil
	})
	if err != nil {
		t.Fatalf("presentation should succeed: %v", err)
	}

	want := []string{"show", "geometry", "focus"}
	if len(popup.operations) != len(want) {
		t.Fatalf("unexpected popup presentation operations: %v", popup.operations)
	}
	for index := range want {
		if popup.operations[index] != want[index] {
			t.Fatalf("unexpected popup presentation operations: %v", popup.operations)
		}
	}
}

func TestDetachedPopupPresentationHidesWhenGeometryFails(t *testing.T) {
	popup := &fakeDetachedPopupPresenter{}
	wantErr := errors.New("native positioning failed")

	err := presentDetachedHUDPopup(popup, func() error { return wantErr })

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected geometry error %v, got %v", wantErr, err)
	}
	want := []string{"show", "hide"}
	if len(popup.operations) != len(want) {
		t.Fatalf("popup should be hidden after geometry failure, got %v", popup.operations)
	}
	for index := range want {
		if popup.operations[index] != want[index] {
			t.Fatalf("popup should be hidden after geometry failure, got %v", popup.operations)
		}
	}
}

func TestDetachedPopupGeometryPreservesCapsuleAnchor(t *testing.T) {
	state := defaultHUDWindowState()
	capsule := application.Rect{X: 1600, Y: 930, Width: 220, Height: 60}
	work := application.Rect{X: 0, Y: 0, Width: 1920, Height: 1080}
	popup := detachedHUDPopupState(state, capsule, work)
	if popup.Width != 334 || popup.Height != 496 {
		t.Fatalf("wrong detached popup size: %+v", popup)
	}
	if popup.X != 1486 || popup.Y != 426 {
		t.Fatalf("detached popup should align above capsule's right edge, got %+v", popup)
	}
	if capsule.Width != 220 || capsule.Height != 60 {
		t.Fatal("popup geometry must not mutate capsule")
	}
}

func TestDetachedPopupGeometryPrefersBelowWhenTopIsConstrained(t *testing.T) {
	state := defaultHUDWindowState()
	capsule := application.Rect{X: 100, Y: 50, Width: 260, Height: 70}
	work := application.Rect{X: 0, Y: 0, Width: 1200, Height: 900}
	popup := detachedHUDPopupState(state, capsule, work)
	if popup.Y != 128 || popup.X != 26 {
		t.Fatalf("should anchor popup below when insufficient room above: %+v", popup)
	}
}

func TestDetachedPopupGeometryClampsToOffsetSmallWorkArea(t *testing.T) {
	state := defaultHUDWindowState()
	state.ExpandedWidth, state.ExpandedHeight = 440, 640
	capsule := application.Rect{X: -1470, Y: 190, Width: 320, Height: 84}
	work := application.Rect{X: -1600, Y: 0, Width: 600, Height: 500}
	popup := detachedHUDPopupState(state, capsule, work)
	if popup.X < work.X || popup.X+popup.Width > work.X+work.Width ||
		popup.Y < work.Y || popup.Y+popup.Height > work.Y+work.Height {
		t.Fatalf("popup must not leave non-origin work area: %+v", popup)
	}
	if popup.Width > work.Width || popup.Height > work.Height {
		t.Fatalf("oversize popup not clamped: %+v", popup)
	}
}

func TestDetachedModeKeepsLegacyDefaultAndFailsClosedWithoutWails(t *testing.T) {
	manager := NewHUDWindowStateManager(filepath.Join(t.TempDir(), "hud.json"))
	legacy := NewHUDController(manager)
	if legacy.GetHUDDetachedPopoverEnabled() || legacy.GetHUDExpanded() {
		t.Fatal("legacy single-window HUD must remain default")
	}
	detached := NewHUDControllerWithDetachedPopover(manager, true)
	if runtime.GOOS == "linux" && hudSession() == "wayland" {
		if detached.GetHUDDetachedPopoverEnabled() {
			t.Fatal("unpositionable Wayland popup must fall back to legacy window")
		}
		return
	}
	if !detached.GetHUDDetachedPopoverEnabled() {
		t.Fatal("explicitly enabled desktop mode should configure detached popup")
	}
	if err := detached.SetHUDExpanded(true); err == nil {
		t.Fatal("detached popup must fail closed when no native windows are running")
	}
	if detached.GetHUDExpanded() {
		t.Fatal("failed expansion must not report popup visible")
	}
	detached.NotifyHUDPopupHidden()
	if detached.GetHUDExpanded() {
		t.Fatal("native hide must keep the controller collapsed")
	}
}

func TestWaylandRejectsRuntimePopupActivation(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Wayland display sessions exist on Linux")
	}
	t.Setenv("XDG_SESSION_TYPE", "wayland")
	manager := NewHUDWindowStateManager(filepath.Join(t.TempDir(), "hud.json"))
	controller := NewHUDControllerWithDetachedPopover(manager, false)
	persistedPreference := manager.Load().DetachedPopover
	factoryCalls := 0
	controller.SetDetachedPopupFactory(func() (application.Window, error) {
		factoryCalls++
		return nil, nil
	})
	if err := controller.SetHUDDetachedPopoverPreference(true); err == nil {
		t.Fatal("Wayland must reject on-demand popup activation")
	}
	if factoryCalls != 0 {
		t.Fatal("Wayland must not attempt to create a native detached popup")
	}
	if controller.GetHUDDetachedPopoverEnabled() {
		t.Fatal("blocked Wayland popup must not become active")
	}
	if manager.Load().DetachedPopover != persistedPreference {
		t.Fatal("rejected runtime popup activation must preserve the existing preference")
	}
}

func TestDetachedPreferenceSaveFailurePreservesRuntimeState(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Setenv("XDG_SESSION_TYPE", "x11")
	}
	// Using a directory as the settings file fails reliably on all platforms,
	// including elevated Windows runners.
	manager := NewHUDWindowStateManager(t.TempDir())
	controller := NewHUDControllerWithDetachedPopover(manager, true)
	if !controller.GetHUDDetachedPopoverEnabled() {
		t.Fatal("test setup requires an active detached popup strategy")
	}
	controller.setHUDExpandedState(true)
	if err := controller.SetHUDDetachedPopoverPreference(false); err == nil {
		t.Fatal("persistence failure must be returned to the caller")
	}
	if !controller.GetHUDDetachedPopoverEnabled() || !controller.GetHUDExpanded() {
		t.Fatal("a failed write must not silently disable an active popup")
	}
}
