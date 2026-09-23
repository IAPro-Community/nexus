package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/IAPro-Community/nexus/internal/app"
	"github.com/IAPro-Community/nexus/internal/control/web"
	"github.com/IAPro-Community/nexus/internal/desktop"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

func main() {
	distFS, err := web.EmbeddedDistFS()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to access embedded web assets: %v\n", err)
		os.Exit(1)
	}

	var core *app.Core
	var backendHandler http.Handler
	var serverURL string
	var desktopSess *web.Session

	if state, stateErr := web.ReadListenState(); stateErr == nil {
		if sess, attachErr := web.AttachLoopbackSession(state); attachErr == nil {
			if proxy, proxyErr := web.NewLoopbackBackendHandler(state.URL, distFS); proxyErr == nil {
				serverURL = state.URL
				desktopSess = sess
				backendHandler = proxy
				fmt.Fprintf(os.Stderr, "Nexus HUD attached to existing Web Core at %s\n", serverURL)
			}
		}
	}

	if backendHandler == nil {
		core, err = app.NewCore(app.CoreConfig{
			Host: "127.0.0.1", Port: 0, NoOpen: true,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize Nexus Core: %v\n", err)
			os.Exit(1)
		}
		coreCtx, coreCancel := context.WithCancel(context.Background())
		defer coreCancel()
		go func() {
			if err := core.Start(coreCtx); err != nil {
				fmt.Fprintf(os.Stderr, "Nexus HUD embedded Core exited: %v\n", err)
			}
		}()
		select {
		case <-core.Ready():
		case <-time.After(10 * time.Second):
			fmt.Fprintln(os.Stderr, "Timeout waiting for Nexus Core to be ready")
			os.Exit(1)
		}
		serverURL = core.URL()
		backendHandler = core.Handler()
		desktopSess, err = core.CreateDesktopSession()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to provision HUD desktop session: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Nexus HUD started an embedded Core at %s\n", serverURL)
	}

	stateManager := desktop.NewHUDWindowStateManager("")
	windowState := stateManager.Load()
	desktopApp := desktop.NewApp(nil, desktop.NewWindowStateManager(""), desktop.BootstrapInfo{
		ServerURL: serverURL, SessionToken: desktopSess.ID, CSRFToken: desktopSess.CSRFToken,
	})
	hudController := desktop.NewHUDController(stateManager)

	err = wails.Run(&options.App{
		Title:          "Nexus HUD",
		SingleInstanceLock: &options.SingleInstanceLock{UniqueId: "com.iapro.nexus.hud"},
		Width:          windowState.Width,
		Height:         windowState.Height,
		MinWidth:       280,
		MinHeight:      48,
		MaxWidth:       1200,
		MaxHeight:      900,
		Frameless:      true,
		AlwaysOnTop:    windowState.AlwaysOnTop,
		DisableResize:  false,
		BackgroundColour: &options.RGBA{R: 15, G: 17, B: 23, A: 0},
		AssetServer: &assetserver.Options{Assets: distFS, Handler: backendHandler},
		OnStartup: func(ctx context.Context) {
			desktopApp.Startup(ctx)
			hudController.Startup(ctx)
		},
		OnDomReady: func(ctx context.Context) {
			hudController.RestoreHUDWindow()
		},
		OnShutdown: func(ctx context.Context) {
			_ = hudController.CaptureHUDWindowState()
			desktopApp.Shutdown(ctx)
			if core != nil {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_ = core.Stop(shutdownCtx)
			}
		},
		Bind: []any{desktopApp, hudController},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			DisableWindowIcon:    true,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHiddenInset(),
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},
		Linux: &linux.Options{
			ProgramName: "nexus-hud",
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Nexus HUD Wails error: %v\n", err)
		os.Exit(1)
	}
}
