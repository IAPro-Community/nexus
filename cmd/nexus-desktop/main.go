package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/IAPro-Community/nexus/internal/app"
	"github.com/IAPro-Community/nexus/internal/desktop"
	"github.com/IAPro-Community/nexus/internal/hudlink"
	"github.com/wailsapp/wails/v3/pkg/application"
)

var desktopSingleInstanceID = "com.iapro.nexus.desktop"

func main() {
	// Detached SessionHost workers reuse the current executable. Dispatch the
	// internal host command before Wails startup so the Desktop binary can
	// supervise providers without opening a second native window.
	if len(os.Args) > 1 && os.Args[1] == "__control-host" {
		if err := app.Run(os.Args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "nexus desktop control host: %v\n", err)
			os.Exit(1)
		}
		return
	}

	backend, err := desktop.OpenSharedBackend("Nexus Desktop")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize shared Nexus backend: %v\n", err)
		os.Exit(1)
	}

	// Start the linked tray HUD with the Desktop shell and keep both bound to
	// the same lifecycle (NEXUS_HUD_SPAWN=0 opts out).
	hudProc, hudErr := hudlink.Spawn()
	if hudErr != nil {
		fmt.Fprintf(os.Stderr, "nexus desktop: HUD companion unavailable: %v\n", hudErr)
	}
	if hudProc != nil {
		defer hudProc.Close()
	}

	windowManager := desktop.NewWindowStateManager("")
	windowState := windowManager.Load()
	desktopApp := desktop.NewAppForWindow(nil, windowManager, desktop.DesktopWindowName, desktop.BootstrapInfo{
		ServerURL:    backend.ServerURL,
		SessionToken: backend.Session.ID,
		CSRFToken:    backend.Session.CSRFToken,
	})

	var window application.Window

	native := application.New(application.Options{
		Name:        "IAPro Nexus",
		Description: "Autonomous AI Orchestrator and Workspace OS",
		Icon:        desktop.NexusAppIcon(),
		Services:    []application.Service{application.NewService(desktopApp)},
		Assets:      application.AssetOptions{Handler: backend.Handler, DisableLogging: true},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: desktopSingleInstanceID,
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				if window != nil {
					window.Show()
					window.Restore()
					window.Focus()
				}
			},
		},
		Mac: application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: false},
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
			WebviewUserDataPath:           desktop.WindowsWebviewUserDataPath("desktop"),
		},
		Linux: application.LinuxOptions{
			DisableQuitOnLastWindowClosed: true,
			ProgramName:                   "nexus-desktop",
			ApplicationID:                 "com.iapro.nexus.desktop",
		},
		OnShutdown: func() {
			if hudProc != nil {
				hudProc.Close()
			}
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = backend.Close(shutdownCtx)
		},
	})

	window = native.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:                       desktop.DesktopWindowName,
		Title:                      "IAPro Nexus",
		Width:                      windowState.Width,
		Height:                     windowState.Height,
		MinWidth:                   800,
		MinHeight:                  600,
		URL:                        "/",
		BackgroundColour:           application.NewRGBA(15, 17, 23, 255),
		DefaultContextMenuDisabled: true,
		ZoomControlEnabled:         true,
	})
	desktop.ConfigureNativeTray(native, window, "IAPro Nexus")

	if err := native.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Wails v3 application error: %v\n", err)
		os.Exit(1)
	}
}
