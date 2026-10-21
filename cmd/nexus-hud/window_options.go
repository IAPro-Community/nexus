package main

import (
	"github.com/IAPro-Community/nexus/internal/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// hudWindowOptions keeps the transparent HUD chrome consistent on Windows and
// macOS. Platform-specific decoration defaults otherwise cast a rectangular
// shadow around a capsule that is intentionally smaller than its native window.
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
