package desktop

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestNexV2LogicalPhysicalConversionAcrossDPI(t *testing.T) {
	cases := []struct {
		dip   int
		scale float32
		want  int
	}{
		{160, 1.0, 160},
		{160, 1.25, 200},
		{160, 1.5, 240},
		{160, 2.0, 320},
		{160, 0, 160},
		{160, -1.5, 160},
	}
	for _, tc := range cases {
		px := LogicalToPhysical(tc.dip, tc.scale)
		if px != tc.want {
			t.Fatalf("LogicalToPhysical(%d, %v)=%d, want %d", tc.dip, tc.scale, px, tc.want)
		}
		roundTrip := PhysicalToLogical(px, tc.scale)
		if roundTrip != tc.dip {
			t.Fatalf("PhysicalToLogical(%d, %v)=%d, want %d", px, tc.scale, roundTrip, tc.dip)
		}
	}
}

func TestResolveNexV2WindowRectNegativeCoordsAndHotplug(t *testing.T) {
	displays := []DisplayDescriptor{
		{
			ID:          "DP-left",
			WorkArea:    application.Rect{X: -1920, Y: -200, Width: 1920, Height: 1040},
			ScaleFactor: 1.5,
		},
		{
			ID:          "eDP-main",
			WorkArea:    application.Rect{X: 0, Y: 0, Width: 1920, Height: 1032},
			ScaleFactor: 1.25,
			Primary:     true,
		},
	}

	// Valid negative-origin secondary monitor preserves custom offset within its work area.
	rect, disp, fellBack := ResolveNexV2WindowRect(NexV2PlacementState{
		DisplayID: "DP-left",
		Anchor:    NexV2AnchorCustom,
		OffsetX:   120,
		OffsetY:   80,
		PresetID:  "large",
	}, displays, true)
	if fellBack || disp.ID != "DP-left" {
		t.Fatalf("expected DP-left without fallback, got disp=%+v fellBack=%v", disp, fellBack)
	}
	if rect.X != -1800 || rect.Y != -120 || rect.Width != 192 || rect.Height != 192 {
		t.Fatalf("unexpected negative monitor rect: %+v", rect)
	}

	// Off-work-area offset clamps inside the left monitor rather than bleeding across.
	clamped, _, _ := ResolveNexV2WindowRect(NexV2PlacementState{
		DisplayID: "DP-left",
		Anchor:    NexV2AnchorCustom,
		OffsetX:   5000,
		OffsetY:   5000,
		PresetID:  "standard",
	}, displays, true)
	if clamped.X != -160 || clamped.Y != 680 {
		t.Fatalf("expected clamp to (-160,680), got %+v", clamped)
	}

	// Disconnected monitor falls back cleanly to primary display work area.
	recovered, disp, fellBack := ResolveNexV2WindowRect(NexV2PlacementState{
		DisplayID: "HDMI-removed",
		Anchor:    NexV2AnchorBottomRight,
		PresetID:  "compact",
	}, displays[1:], true)
	if !fellBack || disp.ID != "eDP-main" {
		t.Fatalf("expected fallback to eDP-main, got disp=%+v fellBack=%v", disp, fellBack)
	}
	if recovered.X != 1920-128-16 || recovered.Y != 1032-128-16 || recovered.Width != 128 {
		t.Fatalf("unexpected fallback bottom-right rect: %+v", recovered)
	}
}

func TestResolveNexV2WindowRectWaylandAndPresetSwitch(t *testing.T) {
	primary := DisplayDescriptor{
		ID:          "wayland-0",
		WorkArea:    application.Rect{X: 0, Y: 0, Width: 1440, Height: 900},
		ScaleFactor: 2.0,
		Primary:     true,
	}

	rect, _, _ := ResolveNexV2WindowRect(NexV2PlacementState{
		DisplayID: "wayland-0",
		Anchor:    NexV2AnchorBottomRight,
		PresetID:  "standard",
	}, []DisplayDescriptor{primary}, false)
	if rect.X != 0 || rect.Y != 0 || rect.Width != 160 || rect.Height != 160 {
		t.Fatalf("Wayland fallback must keep fixed preset dimensions without global coordinates: %+v", rect)
	}

	start := application.Rect{X: 1264, Y: 724, Width: 160, Height: 160}
	next, nextState, ok := SwitchNexV2Preset(start, "large", primary, true)
	if !ok {
		t.Fatal("expected valid preset switch to large")
	}
	// Bottom-right edge (1264+160=1424, 724+160=884) stays anchored.
	if next.Width != 192 || next.Height != 192 || next.X != 1424-192 || next.Y != 884-192 {
		t.Fatalf("preset switch lost bottom-right anchor: %+v", next)
	}
	if nextState.PresetID != "large" || nextState.ScaleFactor != 2.0 {
		t.Fatalf("unexpected placement state after preset switch: %+v", nextState)
	}
}

func TestNexV2StartupRestorerRunsOnceConcurrently(t *testing.T) {
	var restorer NexV2StartupRestorer
	var count atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			restorer.Run(func() {
				count.Add(1)
			})
		}()
	}
	wg.Wait()
	if got := count.Load(); got != 1 {
		t.Fatalf("startup restorer ran %d times, want 1", got)
	}
}
