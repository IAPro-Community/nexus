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
	// A fresh installation starts with Guardian Mascot and an independent
	// popup. Existing user choices are persisted; no environment variable
	// is required to activate the presentation.
	hudController := desktop.NewHUDControllerWithDetachedPopover(stateManager, windowState.DetachedPopover)
	// Wayland uses single-window expansion but keeps the mascot appearance.
	windowState = stateManager.Load()
	if windowState.Appearance == desktop.HUDAppearanceMascot {
		// Match initial Wails dimensions to the mascot, avoiding a frame of
		// the old 220x60 bar when loading a pre-Guardian saved state.
		windowState.Width, windowState.Height = desktop.HUDMascotWidth, desktop.HUDMascotHeight
	}

	var window application.Window
	// Native attention monitoring must not depend on WebView visibility or
	// requestAnimationFrame/timers when the HUD is in tray-only mode.
	notificationCtx, stopNotifications := context.WithCancel(context.Background())
	defer stopNotifications()

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
			stopNotifications()
			_ = hudController.CaptureHUDWindowState()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = backend.Close(shutdownCtx)
		},
	})

	window = native.Window.NewWithOptions(hudWindowOptions(windowState))

	// One factory serves both the startup path and the settings toggle: the
	// popup is a second native Wails window in this same process. Its WebView
	// has its own React tree but shares the authenticated Core.
	createPopup := func() (application.Window, error) {
		if existing, ok := native.Window.GetByName(desktop.HUDPopupWindowName); ok {
			return existing, nil
		}
		popup := native.Window.NewWithOptions(hudDetachedPopupOptions(stateManager.Load()))
		popup.OnWindowEvent(events.Common.WindowLostFocus, func(*application.WindowEvent) {
			// Do not re-enter SetHUDExpanded from a native focus event: Hide()
			// can itself dispatch further focus and hide notifications.
			hudController.NotifyHUDPopupHidden()
			popup.Hide()
		})
		popup.OnWindowEvent(events.Common.WindowHide, func(*application.WindowEvent) {
			// Never Hide() again from WindowHide: avoid reentrant native events.
			hudController.NotifyHUDPopupHidden()
		})
		return popup, nil
	}
	hudController.SetDetachedPopupFactory(createPopup)
	if hudController.GetHUDDetachedPopoverEnabled() {
		if _, err := createPopup(); err != nil {
			fmt.Fprintf(os.Stderr, "Nexus HUD popup: %v\n", err)
		}
	}
	desktop.ConfigureNativeTray(native, window, "Nexus HUD", desktop.HUDTrayAppearanceActions{
		Show: hudController.ShowHUD,
		Hide: hudController.HideHUD,
		// The Mascot works with either a detached popup (preferred) or the
		// single-window fallback on Wayland, without extra configuration.
		MascotAvailable: true,
		Select: func(mode string) {
			if err := hudController.SetHUDAppearance(mode); err != nil {
				fmt.Fprintf(os.Stderr, "Nexus HUD presentation: %v\n", err)
			}
		},
	})

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

	// All supervised Codex/Claude/OpenCode/AGY sessions are read via the
	// canonical registry, even when the frontend WebView is throttled.
	go newHUDAttentionMonitor().run(notificationCtx)

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
