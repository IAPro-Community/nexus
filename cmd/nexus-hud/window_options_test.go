package main

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestHUDWindowOptionsTransparentChromeAcrossPlatforms(t *testing.T) {
	state := desktop.HUDWindowState{
		X:           171,
		Y:           319,
		Width:       260,
		Height:      70,
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
		X:           500,
		Y:           200,
		Width:       320,
		Height:      84,
		AlwaysOnTop: false,
		AutoHide:    true,
	}
	got := hudWindowOptions(state)
	if got.AlwaysOnTop || !got.HideOnEscape || got.Width != 320 || got.Height != 84 {
		t.Fatalf("HUD presentation flags not propagated: %+v", got)
	}
	if got.MinWidth > got.Width || got.MinHeight > got.Height {
		t.Fatalf("native minimum exceeds user-selected compact size: %+v", got)
	}
}

func TestDetachedHUDWindowOptionsStartHiddenWithSeparateRoute(t *testing.T) {
	state := desktop.HUDWindowState{
		X:              900,
		Y:              700,
		Width:          220,
		Height:         60,
		CompactWidth:   220,
		CompactHeight:  60,
		ExpandedWidth:  380,
		ExpandedHeight: 560,
		AlwaysOnTop:    true,
	}
	capsule := hudWindowOptions(state)
	popup := hudDetachedPopupOptions(state)
	if capsule.Hidden || capsule.Name != desktop.HUDWindowName {
		t.Fatalf("capsule should start visible and keep its existing native identity: %+v", capsule)
	}
	if !popup.Hidden || popup.Name != desktop.HUDPopupWindowName || popup.Name == capsule.Name {
		t.Fatalf("detached popup must be independently named and initially hidden: %+v", popup)
	}
	if !popup.Windows.HiddenOnTaskbar {
		t.Fatal("detached popup must not add a second Windows taskbar entry")
	}
	if popup.URL != "/?nativeHud=1&hudView=popover" || popup.Width != 380 || popup.Height != 560 {
		t.Fatalf("detached popup must load independent React view at expanded size: %+v", popup)
	}
	if !popup.Frameless || popup.BackgroundType != application.BackgroundTypeTransparent ||
		popup.BackgroundColour.Alpha != 0 || !popup.Windows.DisableFramelessWindowDecorations ||
		!popup.Mac.DisableShadow || !popup.Linux.WindowIsTranslucent {
		t.Fatalf("detached popup must preserve transparent chrome across platforms: %+v", popup)
	}
	if capsule.Width != 220 || capsule.Height != 60 {
		t.Fatal("opening the popup must never enlarge the compact capsule")
	}
}

func TestGuardianMascotAndTrayNativeWindowContracts(t *testing.T) {
	state := desktop.HUDWindowState{
		Width:      desktop.HUDMascotWidth,
		Height:     desktop.HUDMascotHeight,
		Appearance: desktop.HUDAppearanceMascot,
	}
	mascot := hudWindowOptions(state)
	if mascot.MinWidth > mascot.Width || mascot.Hidden || !mascot.Frameless {
		t.Fatalf("mascot window must be small, visible and borderless: %+v", mascot)
	}
	if !mascot.Mac.DisableShadow || !mascot.Windows.DisableFramelessWindowDecorations ||
		!mascot.Linux.WindowIsTranslucent {
		t.Fatal("mascot must use cross-platform transparent native chrome")
	}
	state.Appearance = desktop.HUDAppearanceTray
	state.Width, state.Height = 220, 60
	tray := hudWindowOptions(state)
	if !tray.Hidden || tray.Width != 220 || tray.MinWidth != 80 {
		t.Fatalf("tray startup must hide capsule and preserve a recoverable full-size bar: %+v", tray)
	}
	popup := hudDetachedPopupOptions(state)
	if !popup.Hidden || popup.Name != desktop.HUDPopupWindowName {
		t.Fatalf("detached detail window must remain hidden until requested: %+v", popup)
	}
}
