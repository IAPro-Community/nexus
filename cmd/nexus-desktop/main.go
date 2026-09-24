package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/IAPro-Community/nexus/internal/desktop"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

func main() {
	backend, err := desktop.OpenSharedBackend("Nexus Desktop")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize shared Nexus backend: %v\n", err)
		os.Exit(1)
	}

	// Initialize native desktop bridge and window state manager.
	windowManager := desktop.NewWindowStateManager("")
	windowState := windowManager.Load()
	desktopApp := desktop.NewApp(nil, windowManager, desktop.BootstrapInfo{
		ServerURL:    backend.ServerURL,
		SessionToken: backend.Session.ID,
		CSRFToken:    backend.Session.CSRFToken,
	})

	// 5. Launch Wails native shell
	err = wails.Run(&options.App{
		Title: "IAPro Nexus",
		// A second native shell must never create another visible Desktop
		// workspace. Web/Core remains the shared source of truth.
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.iapro.nexus.desktop",
		},
		Width:     windowState.Width,
		Height:    windowState.Height,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets:  backend.Assets,
			Handler: backend.Handler,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 17, B: 23, A: 255},
		OnStartup:        desktopApp.Startup,
		OnShutdown: func(ctx context.Context) {
			desktopApp.Shutdown(ctx)
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = backend.Close(shutdownCtx)
		},
		Bind: []any{
			desktopApp,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHiddenInset(),
			WebviewIsTransparent: true,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   "IAPro Nexus",
				Message: "Autonomous AI Orchestrator and Workspace OS",
			},
		},
		Linux: &linux.Options{
			ProgramName: "nexus-desktop",
		},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Wails application error: %v\n", err)
		os.Exit(1)
	}
}
