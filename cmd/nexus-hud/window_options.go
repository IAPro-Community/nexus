package main

import (
	"github.com/IAPro-Community/nexus/internal/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// hudWindowOptions keeps the transparent HUD chrome configured on every OS.
// Windows and macOS have native shadow controls; Linux uses GTK frameless
// rendering and still requires compositor-specific visual verification.
func hudWindowOptions(state desktop.HUDWindowState) application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
		Name:                       desktop.HUDWindowName,
		Title:                      "Nexus HUD",
		Width:                      state.Width,
		Height:                     state.Height,
		MinWidth:                   190,
		MinHeight:                  48,
		MaxWidth:                   1200,
		MaxHeight:                  900,
		URL:                        "/?nativeHud=1",
		Frameless:                  true,
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
