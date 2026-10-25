package desktop

import (
	"math"
	"testing"
)

func TestHUDPointerWithinRegionsFailsOpenWithoutRegions(t *testing.T) {
	if !hudPointerWithinRegionsDip(10, 10, 100, 100, nil) {
		t.Fatal("an uninitialized HUD must stay interactive")
	}
}

func TestHUDPointerWithinRegionsMatchesTransparentMargin(t *testing.T) {
	// Capsule geometry: 220x60 window, pill starting at x=40.
	regions := []HUDHitRegion{{X: 40, Y: 0, Width: 180, Height: 60}}

	cases := []struct {
		name            string
		cursorX         int
		cursorY         int
		wantInteractive bool
	}{
		{name: "transparent top-left corner", cursorX: 104, cursorY: 104, wantInteractive: false},
		{name: "transparent left margin", cursorX: 120, cursorY: 130, wantInteractive: false},
		{name: "inside the pill", cursorX: 160, cursorY: 130, wantInteractive: true},
		{name: "pill left edge", cursorX: 140, cursorY: 130, wantInteractive: true},
		{name: "pill bottom edge", cursorX: 200, cursorY: 160, wantInteractive: true},
		{name: "below the window", cursorX: 200, cursorY: 161, wantInteractive: false},
		{name: "left of the window", cursorX: 99, cursorY: 130, wantInteractive: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := hudPointerWithinRegionsDip(tc.cursorX, tc.cursorY, 100, 100, regions)
			if got != tc.wantInteractive {
				t.Fatalf("hudPointerWithinRegionsDip(%d, %d) = %v, want %v", tc.cursorX, tc.cursorY, got, tc.wantInteractive)
			}
		})
	}
}

func TestHUDPointerWithinRegionsSupportsMultipleRegions(t *testing.T) {
	regions := []HUDHitRegion{
		{X: 0, Y: 0, Width: 20, Height: 20},
		{X: 100, Y: 100, Width: 40, Height: 40},
	}
	if !hudPointerWithinRegionsDip(110, 110, 0, 0, regions) {
		t.Fatal("second region must be interactive")
	}
	if hudPointerWithinRegionsDip(60, 60, 0, 0, regions) {
		t.Fatal("gap between regions is a transparent surface")
	}
}

func TestHUDHitRegionInputRejectsNonFiniteCoordinates(t *testing.T) {
	controller := NewHUDController(nil)
	defer controller.stopHUDHitWatcher()
	if err := controller.SetHUDHitRegions([]HUDHitRegion{
		{X: 1, Y: 2, Width: 32, Height: 44},
		{X: math.NaN(), Y: 0, Width: 20, Height: 20},
		{X: 0, Y: math.Inf(1), Width: 20, Height: 20},
		{X: 0, Y: 0, Width: math.Inf(1), Height: 20},
		{X: 0, Y: 0, Width: 200000, Height: 20},
		{X: 0, Y: 0, Width: -10, Height: 20},
		{X: 20000, Y: 0, Width: 20, Height: 20},
	}); err != nil {
		t.Fatal(err)
	}
	controller.hitMu.RLock()
	defer controller.hitMu.RUnlock()
	if len(controller.hitRegions) != 1 {
		t.Fatalf("malformed geometry must not enter native hit testing: %+v", controller.hitRegions)
	}
	if controller.hitRegions[0].Width != 32 {
		t.Fatal("valid native input rectangle was discarded")
	}
}

func TestHUDHitRegionsConvertToEnclosingNativeRectangles(t *testing.T) {
	regions := []HUDHitRegion{
		{X: 10.25, Y: 20.5, Width: 30.5, Height: 40.25},
		{X: -0.25, Y: 4.1, Width: 1.1, Height: 2.1},
	}
	want := []hudInputRect{
		{X: 10, Y: 20, Width: 31, Height: 41},
		{X: -1, Y: 4, Width: 2, Height: 3},
	}
	got := hudRegionsToInputRectangles(regions)
	if len(got) != len(want) {
		t.Fatalf("hudRegionsToInputRectangles() returned %d rectangles, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("rectangle %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
