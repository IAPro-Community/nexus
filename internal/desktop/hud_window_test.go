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
	if initial.Placement != HUDPlacementBottomRight || initial.Width < 190 || !initial.AlwaysOnTop {
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
	if got.Width != 220 || got.Height != 60 || got.Opacity != 0.96 {
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
		{"expanded", HUDWindowState{Placement: HUDPlacementCustom, X: 100, Y: 200, Width: 334, Height: 496}},
		{"legacy", HUDWindowState{Placement: HUDPlacementCustom, X: 80, Y: 90, Width: 430, Height: 64}},
		{"compact", HUDWindowState{Placement: HUDPlacementCustom, X: 600, Y: 800, Width: 220, Height: 60}},
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
