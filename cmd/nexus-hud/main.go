package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/app"
	"github.com/IAPro-Community/nexus/internal/control/web"
	"github.com/IAPro-Community/nexus/internal/desktop"
	"github.com/IAPro-Community/nexus/internal/hudlink"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

var hudSingleInstanceID = "com.iapro.nexus.hud"

func main() {
	// HUD can also be the binary that spawned a detached SessionHost. Handle
	// the internal worker command before creating the Wails surface.
	if len(os.Args) > 1 && os.Args[1] == "__control-host" {
		if err := app.Run(os.Args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "nexus HUD control host: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Linked mode: started as a companion of a Nexus workspace surface. The
	// HUD follows its core out (see watchLinkedCore) and stays distinguishable
	// from a standalone or login-autostart launch.
	linked := hudlink.Linked(os.Args[1:], os.Getenv)

	backend, err := desktop.OpenSharedBackend("Nexus HUD")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize shared Nexus backend: %v\n", err)
		os.Exit(1)
	}

	stateManager := desktop.NewHUDWindowStateManager("")
	windowState := stateManager.Load()
	desktopApp := desktop.NewAppForWindow(nil, desktop.NewWindowStateManager(""), desktop.HUDWindowName, desktop.BootstrapInfo{
		ServerURL:    backend.ServerURL,
		SessionToken: backend.Session.ID,
		CSRFToken:    backend.Session.CSRFToken,
	})
	// Detached popover is opt-in until real native input/focus tests pass on
	// Windows, macOS and Linux X11. Wayland retains the legacy single window.
	detachedRequested := os.Getenv("NEXUS_HUD_DETACHED_POPOVER") == "1"
	hudController := desktop.NewHUDControllerWithDetachedPopover(stateManager, detachedRequested)

	var window application.Window

	native := application.New(application.Options{
		Name:        "Nexus HUD",
		Description: "Nexus operational HUD",
		Icon:        desktop.NexusAppIcon(),
		Services: []application.Service{
			application.NewService(desktopApp),
			application.NewService(hudController),
		},
		Assets: application.AssetOptions{Handler: backend.Handler, DisableLogging: true},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: hudSingleInstanceID,
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				if window != nil {
					window.Show()
					window.Restore()
					window.Focus()
				}
			},
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory,
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
			WebviewUserDataPath:           desktop.WindowsWebviewUserDataPath("hud"),
		},
		Linux: application.LinuxOptions{
			DisableQuitOnLastWindowClosed: true,
			ProgramName:                   "nexus-hud",
			ApplicationID:                 "com.iapro.nexus.hud",
		},
		OnShutdown: func() {
			_ = hudController.CaptureHUDWindowState()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = backend.Close(shutdownCtx)
		},
	})

	window = native.Window.NewWithOptions(hudWindowOptions(windowState))
	if hudController.GetHUDDetachedPopoverEnabled() {
		// The popup is a second native Wails window in this same process. Its
		// WebView has its own React tree but shares the authenticated Core.
		popupOptions := hudWindowOptions(windowState)
		popupOptions.Name = desktop.HUDPopupWindowName
		popupOptions.Title = desktop.HUDPopupWindowName
		popupOptions.URL = "/?nativeHud=1&hudView=popover"
		popupOptions.Width, popupOptions.Height = windowState.ExpandedWidth, windowState.ExpandedHeight
		popupOptions.MinWidth, popupOptions.MinHeight = 280, 240
		popupOptions.Hidden = true
		popupOptions.HideOnEscape = true
		popup := native.Window.NewWithOptions(popupOptions)
		popup.OnWindowEvent(events.Common.WindowLostFocus, func(*application.WindowEvent) {
			// Outside click dismisses only the popup, not the capsule.
			_ = hudController.SetHUDExpanded(false)
		})
		popup.OnWindowEvent(events.Common.WindowHide, func(*application.WindowEvent) {
			// Never Hide() again from WindowHide: avoid reentrant native events.
			hudController.NotifyHUDPopupHidden()
		})
	}
	desktop.ConfigureNativeTray(native, window, "Nexus HUD")

	// ServiceStartup can run before the native window exists, which left the
	// first paint at Wails' 190x48 minimum. Re-apply the persisted geometry
	// after the first show so custom sizes are honored immediately, without
	// collapsing an expanded HUD every time it is revealed again.
	var initialRestore sync.Once
	window.OnWindowEvent(events.Common.WindowShow, func(*application.WindowEvent) {
		initialRestore.Do(func() {
			go hudController.RestoreHUDWindow()
		})
	})

	// Only a HUD attached to an external core watches its core: a linked HUD
	// that embedded its own Core is that Core's process, so there is nothing
	// to follow.
	if linked && !backend.OwnsEmbeddedCore() {
		go watchLinkedCore(native)
	}

	if err := native.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Nexus HUD Wails v3 error: %v\n", err)
		os.Exit(1)
	}
}

// watchLinkedCore keeps the HUD bound to the Nexus core that launched it:
// ReadListenState only reports live, healthy loopback cores, so when the
// companion core stops (or dies hard) the HUD follows within a few probes.
func watchLinkedCore(app *application.App) {
	const (
		startGrace    = 20 * time.Second
		probeInterval = 3 * time.Second
		maxFailures   = 4
	)
	time.Sleep(startGrace)
	failures := 0
	for {
		time.Sleep(probeInterval)
		if _, err := web.ReadListenState(); err != nil {
			failures++
			if failures >= maxFailures {
				app.Quit()
				return
			}
			continue
		}
		failures = 0
	}
}
