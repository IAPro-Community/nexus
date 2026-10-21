package main

import (
	"github.com/IAPro-Community/nexus/internal/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// hudWindowOptions keeps the transparent HUD chrome configured on every OS.
// Windows and macOS have native shadow controls; Linux uses GTK frameless
// rendering and still requires compositor-specific visual verification.
func hudWindowOptions(state desktop.HUDWindowState) application.WebviewWindowOptions {
	minWidth := 190
	if state.Appearance == desktop.HUDAppearanceMascot {
		minWidth = 80
	}
	return application.WebviewWindowOptions{
		Name:      desktop.HUDWindowName,
		Title:     "Nexus HUD",
		Width:     state.Width,
		Height:    state.Height,
		MinWidth:  minWidth,
		MinHeight: 48,
		MaxWidth:  1200,
		MaxHeight: 900,
		URL:       "/?nativeHud=1",
		Frameless: true,
		Hidden:    state.Appearance == desktop.HUDAppearanceTray,
		Windows: application.WindowsWindow{
			// Remove the DWM's rectangular shadow and native rounded frame.
			DisableFramelessWindowDecorations: true,
		},
		Mac: application.MacWindow{
			// NSWindow shadows the rectangular webview even when its
			// contents have transparent corners; CSS draws the capsule.
			DisableShadow: true,
			CornerType:    application.MacWindowCornerTypeSquare,
		},
		Linux: application.LinuxWindow{
			// GTK still needs the compositor for alpha blending. This is
			// not equivalent to disabling theme/compositor shadows.
			WindowIsTranslucent: true,
		},
		AlwaysOnTop:                state.AlwaysOnTop,
		InitialPosition:            application.WindowXY,
		X:                          state.X,
		Y:                          state.Y,
		BackgroundType:             application.BackgroundTypeTransparent,
		BackgroundColour:           application.NewRGBA(15, 17, 23, 0),
		DefaultContextMenuDisabled: true,
		HideOnEscape:               state.AutoHide,
	}
}

// hudDetachedPopupOptions reuses the exact same native transparency contract
// as the capsule, but starts a distinct larger window hidden until clicked.
func hudDetachedPopupOptions(state desktop.HUDWindowState) application.WebviewWindowOptions {
	options := hudWindowOptions(state)
	options.Name = desktop.HUDPopupWindowName
	options.Title = desktop.HUDPopupWindowName
	options.URL = "/?nativeHud=1&hudView=popover"
	options.Width, options.Height = state.ExpandedWidth, state.ExpandedHeight
	options.MinWidth, options.MinHeight = 280, 240
	options.Hidden = true
	// This transient popup should not create a second Windows taskbar icon.
	options.Windows.HiddenOnTaskbar = true
	options.HideOnEscape = true
	return options
}
