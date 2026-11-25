package main

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestNexV2WindowOptionsFixedBoundsAndTransparentChrome(t *testing.T) {
	state := desktop.DefaultNexV2PlacementState()
	rect := application.Rect{X: 1200, Y: 700, Width: 160, Height: 160}

	got := nexV2WindowOptions(state, rect, false)
	if got.URL != "/nex-v2.html" {
		t.Fatalf("unexpected Nex 2.0 URL %q", got.URL)
	}
	if !got.Frameless || got.BackgroundType != application.BackgroundTypeTransparent || got.BackgroundColour.Alpha != 0 {
		t.Fatalf("Nex 2.0 window must be frameless and transparent: %+v", got)
	}
	if !got.Windows.DisableFramelessWindowDecorations || !got.Mac.DisableShadow || !got.Linux.WindowIsTranslucent {
		t.Fatalf("Nex 2.0 must enforce cross-platform transparent chrome: %+v", got)
	}
	if got.Width != 160 || got.Height != 160 || got.MinWidth != 160 || got.MaxWidth != 160 {
		t.Fatalf("Nex 2.0 production window must lock fixed avatar bounds: %+v", got)
	}

	lab := nexV2WindowOptions(state, rect, true)
	if lab.URL != "/nex-v2.html?lab=1" || lab.Width < 380 || lab.Height < 340 {
		t.Fatalf("unexpected Nex 2.0 lab options: %+v", lab)
	}
}
