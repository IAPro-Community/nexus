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
	// Prefer the already-running loopback Web Control Center. This makes the
	// Desktop and browser surfaces share the same Core, registry and PTY hosts.
	// If Web is not running, fall back to an embedded Desktop-owned Core.
	var core *app.Core
	var backendHandler http.Handler
	var serverURL string
	var desktopSess *web.Session
	// Obtain the embedded frontend before selecting the backend. The Wails asset
	// server owns SPA navigation; only API/WebSocket requests cross to Web Core.
	distFS, err := web.EmbeddedDistFS()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to access embedded web assets: %v\n", err)
		os.Exit(1)
	}
	if state, stateErr := web.ReadListenState(); stateErr == nil {
		if sess, attachErr := web.AttachLoopbackSession(state); attachErr == nil {
			if proxy, proxyErr := web.NewLoopbackBackendHandler(state.URL, distFS); proxyErr == nil {
				serverURL = state.URL
				desktopSess = sess
				backendHandler = proxy
				fmt.Fprintf(os.Stderr, "Nexus Desktop attached to existing Web Core at %s\n", serverURL)
			}
		}
	}

	if backendHandler == nil {
		var err error
		core, err = app.NewCore(app.CoreConfig{
			Host:   "127.0.0.1",
			Port:   0,
			NoOpen: true,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize Nexus Core: %v\n", err)
			os.Exit(1)
		}

		coreCtx, coreCancel := context.WithCancel(context.Background())
		defer coreCancel()

		go func() {
			if err := core.Start(coreCtx); err != nil {
				fmt.Fprintf(os.Stderr, "Nexus Core exited: %v\n", err)
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
			fmt.Fprintf(os.Stderr, "Failed to provision desktop session: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Nexus Desktop started an embedded Core at %s\n", serverURL)
	}

	// Initialize native desktop bridge and window state manager.
	windowManager := desktop.NewWindowStateManager("")
	windowState := windowManager.Load()
	desktopApp := desktop.NewApp(nil, windowManager, desktop.BootstrapInfo{
		ServerURL:    serverURL,
		SessionToken: desktopSess.ID,
		CSRFToken:    desktopSess.CSRFToken,
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
			Assets:  distFS,
			Handler: backendHandler,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 17, B: 23, A: 255},
		OnStartup:        desktopApp.Startup,
		OnShutdown: func(ctx context.Context) {
			desktopApp.Shutdown(ctx)
			if core != nil {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_ = core.Stop(shutdownCtx)
			}
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
