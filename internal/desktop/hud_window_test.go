package desktop

import (
	"path/filepath"
	"testing"
)

func TestHUDWindowStateManagerDefaultsAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hud.json")
	manager := NewHUDWindowStateManager(path)
	initial := manager.Load()
	if initial.Placement != HUDPlacementTopRight || initial.Width < 280 || !initial.AlwaysOnTop {
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

func TestNormalizeHUDWindowStateFailsSafe(t *testing.T) {
	got := normalizeHUDWindowState(HUDWindowState{
		DisplayID: "", Placement: "NOT_REAL", X: -100, Y: -50,
		Width: 10, Height: 10, Opacity: 8,
	})
	if got.Placement != HUDPlacementTopRight || got.X != 0 || got.Y != 0 {
		t.Fatalf("unsafe placement normalization: %+v", got)
	}
	if got.Width != 430 || got.Height != 64 || got.Opacity != 0.96 {
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
