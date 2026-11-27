package desktop

import (
	_ "embed"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/icons"
)

//go:embed assets/brand/nexus-icon-32.png
var nexusTrayIcon []byte

//go:embed assets/brand/nexus-icon-256.png
var nexusAppIcon []byte

//go:embed assets/brand/nex-hud-icon-32.png
var nexHudTrayIcon []byte

//go:embed assets/brand/nex-hud-icon-256.png
var nexHudAppIcon []byte

// NexusAppIcon returns the Nexus brand mark for native shell surfaces (about
// dialogs, Linux application icons and window managers).
func NexusAppIcon() []byte {
	return nexusAppIcon
}

// NexHUDAppIcon returns the NEX agent mascot icon for Nexus HUD native window
// and desktop surfaces.
func NexHUDAppIcon() []byte {
	return nexHudAppIcon
}

// NexHUDTrayIcon returns the NEX agent mascot icon sized for system trays.
func NexHUDTrayIcon() []byte {
	return nexHudTrayIcon
}

// HUDTrayAppearanceActions optionally adds Guardian modes to the native tray.
// Generic desktop windows keep the existing Show/Hide/Quit behavior.
type HUDTrayAppearanceActions struct {
	Select          func(mode string)
	Show            func()
	Hide            func()
	MascotAvailable bool
}

// NewNativeTray creates and brands the platform system tray without requiring
// an initial WebView window (safe for lazy / on-demand HUD hibernation).
func NewNativeTray(app *application.App, tooltip string) *application.SystemTray {
	return NewNativeTrayWithIcon(app, tooltip, nexusTrayIcon)
}

// NewNexHUDTray creates and brands the Nexus HUD system tray using the NEX
// agent mascot icon.
func NewNexHUDTray(app *application.App, tooltip string) *application.SystemTray {
	return NewNativeTrayWithIcon(app, tooltip, nexHudTrayIcon)
}

// NewNativeTrayWithIcon creates and brands the platform system tray with the
// provided PNG icon bytes.
func NewNativeTrayWithIcon(app *application.App, tooltip string, iconBytes []byte) *application.SystemTray {
	if app == nil {
		return nil
	}
	tray := app.SystemTray.New()
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(icons.SystrayMacTemplate)
	} else if len(iconBytes) > 0 {
		// Brand mark: legible on both light and dark taskbars.
		tray.SetIcon(iconBytes)
		tray.SetDarkModeIcon(iconBytes)
	} else {
		tray.SetIcon(icons.SystrayDark)
		tray.SetDarkModeIcon(icons.SystrayLight)
	}
	tray.SetTooltip(tooltip)
	return tray
}

// ConfigureNativeTray installs one cross-platform recovery path for a native
// Nexus window. Closing the window hides it; Quit in the tray is authoritative.
func ConfigureNativeTray(app *application.App, window application.Window, tooltip string, guardian ...HUDTrayAppearanceActions) *application.SystemTray {
	if app == nil || window == nil {
		return nil
	}
	var tray *application.SystemTray
	if len(guardian) > 0 {
		tray = NewNexHUDTray(app, tooltip)
	} else {
		tray = NewNativeTray(app, tooltip)
	}
	if tray == nil {
		return nil
	}

	menu := app.NewMenu()
	show := func() {
		if len(guardian) > 0 && guardian[0].Show != nil {
			guardian[0].Show()
			return
		}
		window.Show()
		window.Restore()
		window.Focus()
	}
	hide := func() {
		if len(guardian) > 0 && guardian[0].Hide != nil {
			guardian[0].Hide()
			return
		}
		window.Hide()
	}
	menu.Add("Mostrar").OnClick(func(_ *application.Context) { show() })
	menu.Add("Ocultar").OnClick(func(_ *application.Context) { hide() })
	if len(guardian) > 0 && guardian[0].Select != nil {
		menu.Add("Guardian: Barra").OnClick(func(_ *application.Context) {
			guardian[0].Select(HUDAppearanceBar)
		})
		if guardian[0].MascotAvailable {
			menu.Add("Guardian: Mascote").OnClick(func(_ *application.Context) {
				guardian[0].Select(HUDAppearanceMascot)
			})
		}
		menu.Add("Guardian: Bandeja").OnClick(func(_ *application.Context) {
			guardian[0].Select(HUDAppearanceTray)
		})
	}
	menu.Add("Sair").OnClick(func(_ *application.Context) {
		app.Quit()
	})
	tray.SetMenu(menu)
	tray.OnClick(func() {
		if window.IsVisible() {
			hide()
			return
		}
		show()
	})

	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		window.Hide()
		event.Cancel()
	})
	return tray
}
