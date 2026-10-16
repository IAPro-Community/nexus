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

// NexusAppIcon returns the Nexus brand mark for native shell surfaces (about
// dialogs, Linux application icons and window managers).
func NexusAppIcon() []byte {
	return nexusAppIcon
}

// ConfigureNativeTray installs one cross-platform recovery path for a native
// Nexus window. Closing the window hides it; Quit in the tray is authoritative.
func ConfigureNativeTray(app *application.App, window application.Window, tooltip string) *application.SystemTray {
	if app == nil || window == nil {
		return nil
	}
	tray := app.SystemTray.New()
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(icons.SystrayMacTemplate)
	} else if len(nexusTrayIcon) > 0 {
		// Brand mark: legible on both light and dark taskbars.
		tray.SetIcon(nexusTrayIcon)
		tray.SetDarkModeIcon(nexusTrayIcon)
	} else {
		tray.SetIcon(icons.SystrayDark)
		tray.SetDarkModeIcon(icons.SystrayLight)
	}
	tray.SetTooltip(tooltip)

	menu := app.NewMenu()
	menu.Add("Mostrar").OnClick(func(_ *application.Context) {
		window.Show()
		window.Restore()
		window.Focus()
	})
	menu.Add("Ocultar").OnClick(func(_ *application.Context) {
		window.Hide()
	})
	menu.Add("Sair").OnClick(func(_ *application.Context) {
		app.Quit()
	})
	tray.SetMenu(menu)
	tray.OnClick(func() {
		if window.IsVisible() {
			window.Hide()
			return
		}
		window.Show()
		window.Restore()
		window.Focus()
	})

	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		window.Hide()
		event.Cancel()
	})
	return tray
}
