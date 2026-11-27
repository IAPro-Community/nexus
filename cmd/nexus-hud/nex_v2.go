package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/desktop"
	"github.com/IAPro-Community/nexus/internal/hudlink"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func nexV2WindowOptions(state desktop.NexV2PlacementState, rect application.Rect, labMode bool) application.WebviewWindowOptions {
	url := "/nex-v2.html"
	width := rect.Width
	height := rect.Height
	minW := 128
	minH := 48
	maxW := 540
	maxH := 620
	if labMode {
		url = "/nex-v2.html?lab=1"
		if width < 380 {
			width = 380
		}
		if height < 340 {
			height = 340
		}
		minW = width
		minH = height
		maxW = 640
		maxH = 480
	}
	return application.WebviewWindowOptions{
		Name:      desktop.NexV2WindowName,
		Title:     "Nexus · Nex 2.0",
		Width:     width,
		Height:    height,
		MinWidth:  minW,
		MinHeight: minH,
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

type nexV2WindowHost struct {
	mu                  sync.Mutex
	native              *application.App
	stateManager        *desktop.NexV2StateManager
	controller          *desktop.NexV2Controller
	caps                desktop.NexV2PlatformCapabilities
	labMode             bool
	window              application.Window
	closingForHibernate bool
}

func (h *nexV2WindowHost) Current() application.Window {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.window
}

func (h *nexV2WindowHost) IsOpen() bool {
	return h.Current() != nil
}

func (h *nexV2WindowHost) bindWindowEvents(win application.Window) {
	if win == nil {
		return
	}
	var restorer desktop.NexV2StartupRestorer
	win.OnWindowEvent(events.Common.WindowShow, func(_ *application.WindowEvent) {
		restorer.Run(func() {
			if h.native == nil || h.labMode {
				return
			}
			screens := h.native.Screen.GetAll()
			primary := h.native.Screen.GetPrimary()
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
			currentPresence := h.stateManager.Load()
			panelOpen := h.controller != nil && h.controller.GetPanelOpen()
			resolved := desktop.ResolveNexV2SurfaceRect(currentPresence, panelOpen, displays, h.caps.GlobalPositioning)
			win.SetSize(resolved.Width, resolved.Height)
			if h.caps.GlobalPositioning {
				win.SetPosition(resolved.X, resolved.Y)
			}
		})
	})
	win.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		st := h.stateManager.Load()
		h.mu.Lock()
		allowDestroy := h.closingForHibernate || (!h.labMode && desktop.NormalizeHudMemoryMode(st.HudMemoryMode) == desktop.NexV2HudMemoryOnDemand)
		if allowDestroy {
			if h.window == win {
				h.window = nil
			}
			h.closingForHibernate = false
		}
		h.mu.Unlock()
		if allowDestroy {
			return
		}
		win.Hide()
		event.Cancel()
	})
}

func (h *nexV2WindowHost) newWindowDirect() (application.Window, error) {
	if h.native == nil {
		return nil, fmt.Errorf("native application unavailable")
	}
	presence := h.stateManager.Load()
	panelOpen := h.controller != nil && h.controller.GetPanelOpen()
	initialRect := desktop.ResolveNexV2SurfaceRect(presence, panelOpen, nil, h.caps.GlobalPositioning)
	win := h.native.Window.NewWithOptions(nexV2WindowOptions(presence.Placement, initialRect, h.labMode))
	if win == nil {
		return nil, fmt.Errorf("failed to create Nex 2.0 WebviewWindow")
	}
	h.bindWindowEvents(win)
	h.mu.Lock()
	h.window = win
	h.closingForHibernate = false
	h.mu.Unlock()
	return win, nil
}

func (h *nexV2WindowHost) EnsureOpen() error {
	if win := h.Current(); win != nil {
		application.InvokeSync(func() {
			win.Show()
			win.Restore()
			win.Focus()
		})
		return nil
	}
	return application.InvokeSyncWithError(func() error {
		if win := h.Current(); win != nil {
			win.Show()
			win.Restore()
			win.Focus()
			return nil
		}
		win, err := h.newWindowDirect()
		if err != nil {
			return err
		}
		win.Show()
		win.Focus()
		return nil
	})
}

func (h *nexV2WindowHost) Close() error {
	win := h.Current()
	if win == nil {
		return nil
	}
	h.mu.Lock()
	h.closingForHibernate = true
	h.mu.Unlock()
	win.Close()
	h.mu.Lock()
	if h.window == win {
		h.window = nil
	}
	h.closingForHibernate = false
	h.mu.Unlock()
	return nil
}

func shouldOpenNexV2OnSecondInstance(args []string, mode string, labMode bool) bool {
	if !labMode && desktop.NormalizeHudMemoryMode(mode) == desktop.NexV2HudMemoryOnDemand && hudlink.Linked(args, nil) {
		return false
	}
	return true
}

func runNexV2HUD(backend *desktop.SharedBackend, desktopApp *desktop.App, linked bool) {
	labMode := os.Getenv("NEXUS_HUD_LAB") == "1"
	instanceID := hudSingleInstanceID
	if labMode {
		instanceID = fmt.Sprintf("com.iapro.nexus.hud.lab.p%d", os.Getpid())
	}

	stateManager := desktop.NewNexV2StateManager("")
	presence := stateManager.Load()
	caps := desktop.DefaultNexV2Capabilities()

	host := &nexV2WindowHost{
		stateManager: stateManager,
		caps:         caps,
		labMode:      labMode,
	}
	var native *application.App
	collectDisplays := func() []desktop.DisplayDescriptor {
		if native == nil {
			return nil
		}
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
		return displays
	}
	controller := desktop.NewNexV2Controller(stateManager, caps, desktop.NexV2WindowHooks{
		ApplyRect: func(rect application.Rect) {
			window := host.Current()
			if window == nil || labMode {
				return
			}
			window.SetSize(rect.Width, rect.Height)
			if caps.GlobalPositioning {
				window.SetPosition(rect.X, rect.Y)
			}
		},
		SetAlwaysOnTop: func(enabled bool) {
			if window := host.Current(); window != nil {
				window.SetAlwaysOnTop(enabled)
			}
		},
		SetVisible: func(visible bool) {
			if visible {
				_ = host.EnsureOpen()
				return
			}
			if desktop.NormalizeHudMemoryMode(stateManager.Load().HudMemoryMode) == desktop.NexV2HudMemoryOnDemand && !labMode {
				_ = host.Close()
				return
			}
			if window := host.Current(); window != nil {
				window.Hide()
			}
		},
		GetDisplays: collectDisplays,
		GetRect: func() application.Rect {
			window := host.Current()
			if window == nil {
				return application.Rect{}
			}
			x, y := window.Position()
			w, h := window.Size()
			return application.Rect{X: x, Y: y, Width: w, Height: h}
		},
	})
	host.controller = controller

	watcherCtx, stopWatcher := context.WithCancel(context.Background())

	native = application.New(application.Options{
		Name:        "Nexus · Nex 2.0",
		Description: "Nexus Nex 2.0 desktop presence",
		Icon:        desktop.NexusAppIcon(),
		Services: []application.Service{
			application.NewService(desktopApp),
			application.NewService(controller),
		},
		Assets: application.AssetOptions{Handler: backend.Handler, DisableLogging: true},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: instanceID,
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				if shouldOpenNexV2OnSecondInstance(data.Args, stateManager.Load().HudMemoryMode, labMode) {
					_ = host.EnsureOpen()
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
			stopWatcher()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if backend != nil {
				_, reportHub := newCoreHUDClient(backend.ServerURL, backend.Session)
				_ = reportHub(shutdownCtx, false, false, "nex_offline")
				_ = backend.Close(shutdownCtx)
			}
		},
	})
	host.native = native

	if labMode || desktop.NormalizeHudMemoryMode(presence.HudMemoryMode) == desktop.NexV2HudMemoryPersistent {
		if _, err := host.newWindowDirect(); err != nil {
			fmt.Fprintf(os.Stderr, "Nexus Nex 2.0 window init warning: %v\n", err)
		}
	}

	configureNexV2Tray(native, host, controller)

	fetchHUD, reportHub := newCoreHUDClient(backend.ServerURL, backend.Session)
	watcher := newNexV2AlertWatcher(nexV2WatcherConfig{
		StateReader: controller.GetPresenceState,
		InteractionOpen: func() bool {
			return controller.GetPanelOpen() || controller.GetSettingsOpen()
		},
		WindowHost: host,
		FetchHUD:   fetchHUD,
		ReportHub:  reportHub,
	})
	go watcher.Run(watcherCtx)

	if linked && !backend.OwnsEmbeddedCore() {
		go watchLinkedCore(native)
	}

	if err := native.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Nexus Nex 2.0 Wails v3 error: %v\n", err)
		os.Exit(1)
	}
}

func configureNexV2Tray(native *application.App, host *nexV2WindowHost, controller *desktop.NexV2Controller) {
	tray := desktop.NewNativeTray(native, "Nexus · Nex 2.0")
	if tray == nil || native == nil {
		return
	}
	initial := controller.GetPresenceState()
	menu := native.NewMenu()
	menu.Add("Mostrar / Ocultar Avatar").OnClick(func(_ *application.Context) {
		st := controller.GetPresenceState()
		nextShow := !st.ShowAvatar
		_, _ = controller.SetShowAvatar(nextShow)
		if nextShow {
			_ = host.EnsureOpen()
		}
	})
	menu.Add("Abrir Painel").OnClick(func(_ *application.Context) {
		_, _ = controller.SetPanelOpen(true)
		_ = host.EnsureOpen()
	})
	menu.Add("Barra Compacta").OnClick(func(_ *application.Context) {
		st := controller.GetPresenceState()
		nextCompact := !st.CompactBar
		_, _ = controller.SetCompactBar(nextCompact)
		if nextCompact {
			_ = host.EnsureOpen()
		}
	})
	onDemandItem := menu.AddCheckbox(
		"Modo Econômico (Sob Demanda)",
		desktop.NormalizeHudMemoryMode(initial.HudMemoryMode) == desktop.NexV2HudMemoryOnDemand,
	)
	onDemandItem.OnClick(func(_ *application.Context) {
		st := controller.GetPresenceState()
		nextMode := desktop.NexV2HudMemoryPersistent
		if desktop.NormalizeHudMemoryMode(st.HudMemoryMode) == desktop.NexV2HudMemoryPersistent {
			nextMode = desktop.NexV2HudMemoryOnDemand
		}
		updated, err := controller.SetHudMemoryMode(nextMode)
		if err == nil {
			onDemandItem.SetChecked(desktop.NormalizeHudMemoryMode(updated.HudMemoryMode) == desktop.NexV2HudMemoryOnDemand)
			if desktop.NormalizeHudMemoryMode(updated.HudMemoryMode) == desktop.NexV2HudMemoryPersistent {
				_ = host.EnsureOpen()
			}
		}
	})
	dndItem := menu.AddCheckbox("Não Perturbe", initial.DoNotDisturb)
	dndItem.OnClick(func(_ *application.Context) {
		st := controller.GetPresenceState()
		updated, err := controller.SetDoNotDisturb(!st.DoNotDisturb)
		if err == nil {
			dndItem.SetChecked(updated.DoNotDisturb)
		}
	})
	pausedItem := menu.AddCheckbox("Pausar Movimento", initial.Paused)
	pausedItem.OnClick(func(_ *application.Context) {
		st := controller.GetPresenceState()
		updated, err := controller.SetPaused(!st.Paused)
		if err == nil {
			pausedItem.SetChecked(updated.Paused)
		}
	})
	menu.Add("Sair").OnClick(func(_ *application.Context) {
		native.Quit()
	})
	tray.SetMenu(menu)
	tray.OnClick(func() {
		if window := host.Current(); window != nil && window.IsVisible() {
			if desktop.NormalizeHudMemoryMode(controller.GetPresenceState().HudMemoryMode) == desktop.NexV2HudMemoryOnDemand {
				_ = host.Close()
				return
			}
			window.Hide()
			return
		}
		_ = host.EnsureOpen()
	})
}
