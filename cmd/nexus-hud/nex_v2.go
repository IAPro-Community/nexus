package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/IAPro-Community/nexus/internal/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func nexV2WindowOptions(state desktop.NexV2PlacementState, rect application.Rect, labMode bool) application.WebviewWindowOptions {
	url := "/nex-v2.html"
	width := rect.Width
	height := rect.Height
	maxW := width
	maxH := height
	if labMode {
		url = "/nex-v2.html?lab=1"
		if width < 380 {
			width = 380
		}
		if height < 340 {
			height = 340
		}
		maxW = 640
		maxH = 480
	}
	return application.WebviewWindowOptions{
		Name:      desktop.NexV2WindowName,
		Title:     "Nexus · Nex 2.0",
		Width:     width,
		Height:    height,
		MinWidth:  width,
		MinHeight: height,
		MaxWidth:  maxW,
		MaxHeight: maxH,
		URL:       url,
		Frameless: true,
		Hidden:    false,
		Windows: application.WindowsWindow{
			DisableFramelessWindowDecorations: true,
		},
		Mac: application.MacWindow{
			DisableShadow: true,
			CornerType:    application.MacWindowCornerTypeSquare,
		},
		Linux: application.LinuxWindow{
			WindowIsTranslucent: true,
		},
		AlwaysOnTop:                state.AlwaysOnTop,
		InitialPosition:            application.WindowXY,
		X:                          rect.X,
		Y:                          rect.Y,
		BackgroundType:             application.BackgroundTypeTransparent,
		BackgroundColour:           application.NewRGBA(15, 17, 23, 0),
		DefaultContextMenuDisabled: true,
	}
}

func runNexV2HUD(backend *desktop.SharedBackend, linked bool) {
	labMode := os.Getenv("NEXUS_HUD_LAB") == "1"
	instanceID := hudSingleInstanceID
	if labMode {
		instanceID = fmt.Sprintf("com.iapro.nexus.hud.lab.p%d", os.Getpid())
	}

	placement := desktop.DefaultNexV2PlacementState()
	caps := desktop.DefaultHUDPlatformCapabilities()
	initialRect, _, _ := desktop.ResolveNexV2WindowRect(placement, nil, caps.GlobalPositioning)

	var window application.Window
	native := application.New(application.Options{
		Name:        "Nexus · Nex 2.0",
		Description: "Nexus Nex 2.0 desktop presence",
		Icon:        desktop.NexusAppIcon(),
		Assets:      application.AssetOptions{Handler: backend.Handler, DisableLogging: true},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: instanceID,
			OnSecondInstanceLaunch: func(_ application.SecondInstanceData) {
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
			WebviewUserDataPath:           desktop.WindowsWebviewUserDataPath("hud-v2"),
		},
		Linux: application.LinuxOptions{
			DisableQuitOnLastWindowClosed: true,
			ProgramName:                   "nexus-hud",
			ApplicationID:                 "com.iapro.nexus.hud",
		},
		OnShutdown: func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = backend.Close(shutdownCtx)
		},
	})

	window = native.Window.NewWithOptions(nexV2WindowOptions(placement, initialRect, labMode))
	desktop.ConfigureNativeTray(native, window, "Nexus · Nex 2.0")

	var restorer desktop.NexV2StartupRestorer
	window.OnWindowEvent(events.Common.WindowShow, func(_ *application.WindowEvent) {
		restorer.Run(func() {
			screens := native.Screen.GetAll()
			primary := native.Screen.GetPrimary()
			displays := make([]desktop.DisplayDescriptor, 0, len(screens))
			for _, s := range screens {
				if s == nil {
					continue
				}
				displays = append(displays, desktop.DisplayDescriptor{
					ID:          s.ID,
					WorkArea:    s.WorkArea,
					ScaleFactor: s.ScaleFactor,
					Primary:     primary != nil && s.ID == primary.ID,
				})
			}
			resolved, _, _ := desktop.ResolveNexV2WindowRect(placement, displays, caps.GlobalPositioning)
			if labMode {
				return
			}
			window.SetSize(resolved.Width, resolved.Height)
			if caps.GlobalPositioning {
				window.SetPosition(resolved.X, resolved.Y)
			}
		})
	})

	if linked && !backend.OwnsEmbeddedCore() {
		go watchLinkedCore(native)
	}

	if err := native.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Nexus Nex 2.0 Wails v3 error: %v\n", err)
		os.Exit(1)
	}
}
