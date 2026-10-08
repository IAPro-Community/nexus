package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/IAPro-Community/nexus/internal/app"
	"github.com/IAPro-Community/nexus/internal/desktop"
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

	windowManager := desktop.NewWindowStateManager("")
	windowState := windowManager.Load()
	desktopApp := desktop.NewAppForWindow(nil, windowManager, desktop.DesktopWindowName, desktop.BootstrapInfo{
		ServerURL:    backend.ServerURL,
		SessionToken: backend.Session.ID,
		CSRFToken:    backend.Session.CSRFToken,
	})

	native := application.New(application.Options{
		Name:        "IAPro Nexus",
		Description: "Autonomous AI Orchestrator and Workspace OS",
		Services:    []application.Service{application.NewService(desktopApp)},
		Assets:      application.AssetOptions{Handler: backend.Handler, DisableLogging: true},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: desktopSingleInstanceID,
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
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = backend.Close(shutdownCtx)
		},
	})

	window := native.Window.NewWithOptions(application.WebviewWindowOptions{
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
