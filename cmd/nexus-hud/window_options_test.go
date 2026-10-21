package main

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestHUDWindowOptionsTransparentChromeAcrossPlatforms(t *testing.T) {
	state := desktop.HUDWindowState{
		X: 171,
		Y: 319,
		Width: 260,
		Height: 70,
		AlwaysOnTop: true,
	}
	got := hudWindowOptions(state)

	if !got.Frameless || got.BackgroundType != application.BackgroundTypeTransparent {
		t.Fatalf("HUD must be frameless and transparent: %+v", got)
	}
	if got.BackgroundColour.Alpha != 0 {
		t.Fatalf("HUD transparent background must have zero alpha: %+v", got.BackgroundColour)
	}
	if !got.Windows.DisableFramelessWindowDecorations {
		t.Fatal("Windows must disable native frameless DWM shadow and rounded outline")
	}
	// Linux has no equivalent native DisableShadow in Wails beta.25. Guard
	// the GTK transparent/frameless contract, then require compositor smoke.
	if !got.Linux.WindowIsTranslucent {
		t.Fatal("Linux HUD must explicitly request GTK translucent presentation")
	}
	if !got.Mac.DisableShadow || got.Mac.CornerType != application.MacWindowCornerTypeSquare {
		t.Fatalf("macOS must disable native rectangular shadow and native corner mask: %+v", got.Mac)
	}
	if got.Width != state.Width || got.Height != state.Height || got.X != state.X || got.Y != state.Y {
		t.Fatalf("HUD must preserve persisted native size and position: %+v", got)
	}
	if !got.AlwaysOnTop || got.URL != "/?nativeHud=1" || got.Name != desktop.HUDWindowName {
		t.Fatalf("HUD presentation or identity changed: %+v", got)
	}
}

func TestHUDWindowOptionsPreserveUserChoices(t *testing.T) {
	state := desktop.HUDWindowState{
		X: 500,
		Y: 200,
		Width: 320,
		Height: 84,
		AlwaysOnTop: false,
		AutoHide: true,
	}
	got := hudWindowOptions(state)
	if got.AlwaysOnTop || !got.HideOnEscape || got.Width != 320 || got.Height != 84 {
		t.Fatalf("HUD presentation flags not propagated: %+v", got)
	}
	if got.MinWidth > got.Width || got.MinHeight > got.Height {
		t.Fatalf("native minimum exceeds user-selected compact size: %+v", got)
	}
}
