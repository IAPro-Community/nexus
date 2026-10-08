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
	hudController := desktop.NewHUDController(stateManager)

	native := application.New(application.Options{
		Name:        "Nexus HUD",
		Description: "Nexus operational HUD",
		Services: []application.Service{
			application.NewService(desktopApp),
			application.NewService(hudController),
		},
		Assets: application.AssetOptions{Handler: backend.Handler, DisableLogging: true},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: hudSingleInstanceID,
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

	window := native.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:                       desktop.HUDWindowName,
		Title:                      "Nexus HUD",
		Width:                      windowState.Width,
		Height:                     windowState.Height,
		MinWidth:                   280,
		MinHeight:                  48,
		MaxWidth:                   1200,
		MaxHeight:                  900,
		URL:                        "/?nativeHud=1",
		Frameless:                  true,
		AlwaysOnTop:                windowState.AlwaysOnTop,
		InitialPosition:            application.WindowXY,
		X:                          windowState.X,
		Y:                          windowState.Y,
		BackgroundType:             application.BackgroundTypeTransparent,
		BackgroundColour:           application.NewRGBA(15, 17, 23, 0),
		DefaultContextMenuDisabled: true,
		HideOnEscape:               windowState.AutoHide,
	})
	desktop.ConfigureNativeTray(native, window, "Nexus HUD")

	if err := native.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Nexus HUD Wails v3 error: %v\n", err)
		os.Exit(1)
	}
}
