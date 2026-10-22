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

// HUDTrayAppearanceActions optionally adds Guardian modes to the native tray.
// Generic desktop windows keep the existing Show/Hide/Quit behavior.
type HUDTrayAppearanceActions struct {
	Select          func(mode string)
	Show            func()
	Hide            func()
	MascotAvailable bool
}

// ConfigureNativeTray installs one cross-platform recovery path for a native
// Nexus window. Closing the window hides it; Quit in the tray is authoritative.
func ConfigureNativeTray(app *application.App, window application.Window, tooltip string, guardian ...HUDTrayAppearanceActions) *application.SystemTray {
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
