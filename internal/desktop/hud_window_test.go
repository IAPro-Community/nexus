package desktop

import (
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func TestHUDWindowStateManagerDefaultsAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hud.json")
	manager := NewHUDWindowStateManager(path)
	initial := manager.Load()
	if initial.Placement != HUDPlacementBottomRight || initial.Width != HUDMascotWidth || initial.Height != HUDMascotHeight || initial.Appearance != HUDAppearanceMascot || !initial.AlwaysOnTop {
		t.Fatalf("unexpected defaults: %+v", initial)
	}

	want := HUDWindowState{
		DisplayID: "current", Placement: HUDPlacementCustom,
		X: 77, Y: 88, Width: 520, Height: 300,
		Mode: "RESOURCE", Opacity: 0.8, AlwaysOnTop: false,
	}
	if err := manager.Save(want); err != nil {
		t.Fatal(err)
	}
	got := manager.Load()
	if got.X != 77 || got.Y != 88 || got.Width != 520 || got.Mode != "RESOURCE" || got.Opacity != 0.8 {
		t.Fatalf("persisted state=%+v", got)
	}
}

func TestNormalizeHUDWindowStateDefersPositionClampUntilScreenIsKnown(t *testing.T) {
	got := normalizeHUDWindowState(HUDWindowState{
		DisplayID: "", Placement: "NOT_REAL", X: -100, Y: -50,
		Width: 10, Height: 10, Opacity: 8,
	})
	if got.Placement != HUDPlacementBottomRight || got.X != -100 || got.Y != -50 {
		t.Fatalf("normalization must preserve coordinates until work-area clamp: %+v", got)
	}
	if got.Width != HUDMascotWidth || got.Height != HUDMascotHeight || got.Opacity != 0.96 {
		t.Fatalf("unsafe size/opacity normalization: %+v", got)
	}
}

func TestClampHUDWindowStateToScreenRecoversOffscreenPosition(t *testing.T) {
	got := clampHUDWindowStateToScreen(HUDWindowState{
		DisplayID: "disconnected",
		Placement: HUDPlacementCustom,
		X:         3800,
		Y:         1900,
		Width:     430,
		Height:    520,
		Mode:      "AUTO",
		Opacity:   0.96,
	}, 1920, 1080)
	if got.X != 1490 || got.Y != 560 {
		t.Fatalf("recovered position=(%d,%d), want (1490,560)", got.X, got.Y)
	}
}

func TestClampHUDWindowStateToScreenShrinksOversizedWindow(t *testing.T) {
	got := clampHUDWindowStateToScreen(HUDWindowState{
		DisplayID: "current",
		Placement: HUDPlacementCustom,
		X:         200,
		Y:         100,
		Width:     1200,
		Height:    900,
		Mode:      "AUTO",
		Opacity:   0.96,
	}, 800, 600)
	if got.Width != 800 || got.Height != 600 || got.X != 0 || got.Y != 0 {
		t.Fatalf("recovered state=%+v", got)
	}
}

func TestHUDPlatformCapabilitiesKeepRuntimeBackedClaimsFailClosed(t *testing.T) {
	caps := DefaultHUDPlatformCapabilities()
	if caps.Platform != runtime.GOOS {
		t.Fatalf("platform=%q want %q", caps.Platform, runtime.GOOS)
	}
	if caps.Tray || caps.MultiMonitor || caps.AutoStart || caps.AutoHide {
		t.Fatalf("runtime-backed HUD capabilities must remain false before ServiceStartup: %+v", caps)
	}
}

func TestHUDSessionHonorsExplicitGDKBackend(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("GDK backend selection is specific to Linux desktop sessions")
	}
	t.Setenv("XDG_SESSION_TYPE", "wayland")
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	t.Setenv("DISPLAY", ":0")
	t.Setenv("GDK_BACKEND", "x11")

	caps := DefaultHUDPlatformCapabilities()
	if caps.Session != "x11" || !caps.GlobalPositioning {
		t.Fatalf("explicit X11 backend must enable global HUD positioning, got %+v", caps)
	}

	t.Setenv("GDK_BACKEND", "wayland")
	caps = DefaultHUDPlatformCapabilities()
	if caps.Session != "wayland" || caps.GlobalPositioning {
		t.Fatalf("explicit Wayland backend must keep global HUD positioning disabled, got %+v", caps)
	}
}

func TestCompactHUDWindowStatePreservesBottomRightAnchor(t *testing.T) {
	cases := []struct {
		name string
		in   HUDWindowState
	}{
		{"expanded", HUDWindowState{Appearance: HUDAppearanceBar, Placement: HUDPlacementCustom, X: 100, Y: 200, Width: 334, Height: 496}},
		{"legacy", HUDWindowState{Appearance: HUDAppearanceBar, Placement: HUDPlacementCustom, X: 80, Y: 90, Width: 430, Height: 64}},
		{"compact", HUDWindowState{Appearance: HUDAppearanceBar, Placement: HUDPlacementCustom, X: 600, Y: 800, Width: 220, Height: 60}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := compactHUDWindowState(tt.in)
			if got.Width != 220 || got.Height != 60 {
				t.Fatalf("restored HUD must be compact, got %+v", got)
			}
			if got.X+got.Width != tt.in.X+tt.in.Width || got.Y+got.Height != tt.in.Y+tt.in.Height {
				t.Fatalf("taskbar anchor changed: in=%+v got=%+v", tt.in, got)
			}
			again := compactHUDWindowState(got)
			if got != again {
				t.Fatalf("compact normalization must be idempotent: %+v != %+v", got, again)
			}
		})
	}
}

func TestCompactHUDWindowStateUsesConfiguredCompactDimensions(t *testing.T) {
	in := HUDWindowState{
		DisplayID: "current", Placement: HUDPlacementCustom,
		X: 100, Y: 200, Width: 380, Height: 560,
		CompactWidth: 260, CompactHeight: 70,
		ExpandedWidth: 380, ExpandedHeight: 560,
		Mode: "AUTO", Appearance: HUDAppearanceBar, Opacity: 0.96,
	}
	got := compactHUDWindowState(in)
	if got.Width != 260 || got.Height != 70 {
		t.Fatalf("custom compact size lost: %+v", got)
	}
	if got.X+got.Width != in.X+in.Width || got.Y+got.Height != in.Y+in.Height {
		t.Fatalf("custom compact size must preserve bottom-right anchor: in=%+v got=%+v", in, got)
	}
}

func TestNormalizeHUDWindowStateKeepsSafeCustomSizes(t *testing.T) {
	got := normalizeHUDWindowState(HUDWindowState{
		DisplayID: "current", Placement: HUDPlacementCustom,
		CompactWidth: 260, CompactHeight: 70,
		ExpandedWidth: 380, ExpandedHeight: 560,
	})
	if got.CompactWidth != 260 || got.CompactHeight != 70 ||
		got.ExpandedWidth != 380 || got.ExpandedHeight != 560 {
		t.Fatalf("safe custom sizes were changed: %+v", got)
	}

	unsafe := normalizeHUDWindowState(HUDWindowState{
		DisplayID: "current", Placement: HUDPlacementCustom,
		CompactWidth: 10, CompactHeight: 5000,
		ExpandedWidth: 100, ExpandedHeight: 100,
	})
	if unsafe.CompactWidth != 220 || unsafe.CompactHeight != 60 ||
		unsafe.ExpandedWidth != 334 || unsafe.ExpandedHeight != 496 {
		t.Fatalf("unsafe custom sizes must fall back to defaults: %+v", unsafe)
	}
}

func TestHUDSizePresetsAreDistinctAndBounded(t *testing.T) {
	presets := (&HUDController{}).GetHUDSizePresets()
	if len(presets) < 3 {
		t.Fatalf("expected at least three size presets, got %d", len(presets))
	}
	seen := map[string]bool{}
	for _, preset := range presets {
		if preset.ID == "" || preset.Label == "" || seen[preset.ID] {
			t.Fatalf("invalid or duplicate preset: %+v", preset)
		}
		seen[preset.ID] = true
		if preset.CompactWidth < 190 || preset.CompactHeight < 48 ||
			preset.ExpandedWidth < preset.CompactWidth || preset.ExpandedHeight < preset.CompactHeight {
			t.Fatalf("preset violates geometry contract: %+v", preset)
		}
	}
	if _, ok := resolveHUDSizePreset("comfortable"); !ok {
		t.Fatal("comfortable preset must resolve")
	}
	if _, ok := resolveHUDSizePreset("invented"); ok {
		t.Fatal("unknown presets must not resolve")
	}
}

func TestHUDControllerSizePresetRequiresNativeWindow(t *testing.T) {
	controller := NewHUDController(NewHUDWindowStateManager(filepath.Join(t.TempDir(), "hud.json")))
	if err := controller.SetHUDSizePreset("comfortable", false); err == nil {
		t.Fatal("size changes without a native window must fail closed")
	}
	if err := controller.MoveHUD(10, 10); err == nil {
		t.Fatal("moves without a native window must fail closed")
	}
}

func TestHUDResizeAnchorRespectsUnsupportedGlobalPositioning(t *testing.T) {
	const originalX, originalY = 500, 350
	movedX, movedY := hudResizeAnchor(originalX, originalY, 108, 112, 334, 496, true)
	if movedX != 274 || movedY != -34 {
		t.Fatalf("movable HUD must preserve bottom-right anchor, got (%d,%d)", movedX, movedY)
	}
	waylandX, waylandY := hudResizeAnchor(originalX, originalY, 108, 112, 334, 496, false)
	if waylandX != originalX || waylandY != originalY {
		t.Fatalf("non-positionable compositor must preserve native origin, got (%d,%d)", waylandX, waylandY)
	}
}

// fakeHUDWindow embeds the Wails interface so the regression test only has to
// implement the handful of methods RestoreHUDWindow actually calls.
type fakeHUDWindow struct {
	application.Window
	mu           sync.Mutex
	handlers     map[events.WindowEventType]func(*application.WindowEvent)
	sizeCalls    int
	positionCall int
	lastWidth    int
	lastHeight   int
	lastX        int
	lastY        int
}

func newFakeHUDWindow() *fakeHUDWindow {
	return &fakeHUDWindow{handlers: map[events.WindowEventType]func(*application.WindowEvent){}}
}

func (w *fakeHUDWindow) OnWindowEvent(eventType events.WindowEventType, callback func(*application.WindowEvent)) func() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers[eventType] = callback
	return func() {}
}

func (w *fakeHUDWindow) emit(eventType events.WindowEventType) {
	w.mu.Lock()
	handler := w.handlers[eventType]
	w.mu.Unlock()
	if handler != nil {
		handler(nil)
	}
}

func (w *fakeHUDWindow) Size() (int, int) { return w.lastWidth, w.lastHeight }

func (w *fakeHUDWindow) Position() (int, int) { return w.lastX, w.lastY }

func (w *fakeHUDWindow) SetSize(width, height int) application.Window {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.sizeCalls++
	w.lastWidth, w.lastHeight = width, height
	return w
}

func (w *fakeHUDWindow) SetPosition(x, y int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.positionCall++
	w.lastX, w.lastY = x, y
}

func (w *fakeHUDWindow) SetAlwaysOnTop(bool) application.Window { return w }

func TestCaptureHUDWindowStatePreservesWaylandGeometry(t *testing.T) {
	manager := NewHUDWindowStateManager(filepath.Join(t.TempDir(), "hud.json"))
	want := HUDWindowState{
		DisplayID: "current", Placement: HUDPlacementCustom,
		X: 100, Y: 200, Width: HUDMascotWidth, Height: HUDMascotHeight,
		CompactWidth: 220, CompactHeight: 60, ExpandedWidth: 334, ExpandedHeight: 496,
		Mode: "AUTO", Appearance: HUDAppearanceMascot, Opacity: 0.96, AlwaysOnTop: true,
	}
	if err := manager.Save(want); err != nil {
		t.Fatal(err)
	}

	controller := NewHUDController(manager)
	controller.caps.GlobalPositioning = false
	window := newFakeHUDWindow()
	window.lastWidth, window.lastHeight = 80, 48
	window.lastX, window.lastY = 0, 0

	if err := controller.captureHUDWindowState(window); err != nil {
		t.Fatal(err)
	}
	got := manager.Load()
	if got.Width != want.Width || got.Height != want.Height || got.X != want.X || got.Y != want.Y {
		t.Fatalf("non-positionable compositor overwrote persisted geometry: got (%d,%d %dx%d), want (%d,%d %dx%d)",
			got.X, got.Y, got.Width, got.Height, want.X, want.Y, want.Width, want.Height)
	}
}

func TestCaptureHUDWindowStatePersistsPositionableGeometry(t *testing.T) {
	manager := NewHUDWindowStateManager(filepath.Join(t.TempDir(), "hud.json"))
	want := HUDWindowState{
		DisplayID: "current", Placement: HUDPlacementCustom,
		X: 100, Y: 200, Width: HUDMascotWidth, Height: HUDMascotHeight,
		CompactWidth: 220, CompactHeight: 60, ExpandedWidth: 334, ExpandedHeight: 496,
		Mode: "AUTO", Appearance: HUDAppearanceMascot, Opacity: 0.96, AlwaysOnTop: true,
	}
	if err := manager.Save(want); err != nil {
		t.Fatal(err)
	}

	controller := NewHUDController(manager)
	controller.caps.GlobalPositioning = true
	window := newFakeHUDWindow()
	window.lastWidth, window.lastHeight = 320, 210
	window.lastX, window.lastY = 50, 60

	if err := controller.captureHUDWindowState(window); err != nil {
		t.Fatal(err)
	}
	got := manager.Load()
	if got.Width != 320 || got.Height != 210 || got.X != 50 || got.Y != 60 {
		t.Fatalf("positionable geometry was not captured: got (%d,%d %dx%d)", got.X, got.Y, got.Width, got.Height)
	}
}

// Regression: ServiceStartup runs before the native window exists on Windows,
// so restore must also be wired to WindowShow. It must run exactly once even
// when the window emits show events repeatedly.
func TestRestoreHUDWindowOnShowAppliesPersistedGeometryOnce(t *testing.T) {
	manager := NewHUDWindowStateManager(filepath.Join(t.TempDir(), "hud.json"))
	if err := manager.Save(HUDWindowState{
		DisplayID: "current", Placement: HUDPlacementCustom,
		X: 300, Y: 200, Width: 220, Height: 60,
		Mode: "AUTO", Appearance: HUDAppearanceBar, Opacity: 0.96, AlwaysOnTop: true,
	}); err != nil {
		t.Fatal(err)
	}
	controller := NewHUDController(manager)
	// This test covers the show-event restoration contract. Select the
	// position-capable path explicitly so it is independent of the host's
	// Wayland/X11 session; Wayland capability behavior has separate coverage.
	controller.caps.GlobalPositioning = true
	window := newFakeHUDWindow()

	controller.restoreHUDWindowWhenShown(window)
	if window.sizeCalls != 0 {
		t.Fatalf("restore must not run before the show event, calls=%d", window.sizeCalls)
	}

	window.emit(events.Common.WindowShow)
	if window.sizeCalls != 1 {
		t.Fatalf("show event must apply geometry once, calls=%d", window.sizeCalls)
	}
	if window.lastWidth != 220 || window.lastHeight != 60 {
		t.Fatalf("persisted expanded size must restore compact: %dx%d", window.lastWidth, window.lastHeight)
	}
	if window.lastX != 300 || window.lastY != 200 {
		t.Fatalf("custom position lost: %d,%d", window.lastX, window.lastY)
	}

	window.emit(events.Common.WindowShow)
	if window.sizeCalls != 1 {
		t.Fatalf("repeated show events must not re-apply stale geometry, calls=%d", window.sizeCalls)
	}
}
