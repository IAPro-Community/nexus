package desktop

import "testing"

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
