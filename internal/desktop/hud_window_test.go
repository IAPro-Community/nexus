package desktop

import (
	"path/filepath"
	"runtime"
	"testing"
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
