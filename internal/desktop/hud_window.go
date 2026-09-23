package desktop

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	HUDPlacementTopLeft      = "TOP_LEFT"
	HUDPlacementTopCenter    = "TOP_CENTER"
	HUDPlacementTopRight     = "TOP_RIGHT"
	HUDPlacementLeftCenter   = "LEFT_CENTER"
	HUDPlacementRightCenter  = "RIGHT_CENTER"
	HUDPlacementBottomLeft   = "BOTTOM_LEFT"
	HUDPlacementBottomCenter = "BOTTOM_CENTER"
	HUDPlacementBottomRight  = "BOTTOM_RIGHT"
	HUDPlacementCustom       = "CUSTOM"
)

type HUDPlatformCapabilities struct {
	TransparentWindow   bool   `json:"transparentWindow"`
	FramelessWindow     bool   `json:"framelessWindow"`
	AlwaysOnTop         bool   `json:"alwaysOnTop"`
	GlobalPositioning   bool   `json:"globalPositioning"`
	MultiMonitor        bool   `json:"multiMonitor"`
	AutoHide            bool   `json:"autoHide"`
	NativeNotifications bool   `json:"nativeNotifications"`
	Tray                bool   `json:"tray"`
	Platform            string `json:"platform"`
	Session             string `json:"session,omitempty"`
}

type HUDWindowState struct {
	DisplayID   string  `json:"displayId"`
	Placement   string  `json:"placement"`
	X           int     `json:"x"`
	Y           int     `json:"y"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	Mode        string  `json:"mode"`
	Opacity     float64 `json:"opacity"`
	AlwaysOnTop bool    `json:"alwaysOnTop"`
	AutoHide    bool    `json:"autoHide"`
}

type HUDWindowStateManager struct {
	mu       sync.RWMutex
	filePath string
}

func NewHUDWindowStateManager(filePath string) *HUDWindowStateManager {
	if strings.TrimSpace(filePath) == "" {
		home, _ := os.UserHomeDir()
		filePath = filepath.Join(home, ".local", "share", "iapro-nexus", "hud-window-state.json")
	}
	return &HUDWindowStateManager{filePath: filePath}
}

func defaultHUDWindowState() HUDWindowState {
	return HUDWindowState{
		DisplayID:   "current",
		Placement:   HUDPlacementTopRight,
		X:           24,
		Y:           24,
		Width:       430,
		Height:      64,
		Mode:        "AUTO",
		Opacity:     0.96,
		AlwaysOnTop: true,
		AutoHide:    false,
	}
}

func normalizeHUDWindowState(state HUDWindowState) HUDWindowState {
	defaults := defaultHUDWindowState()
	switch state.Placement {
	case HUDPlacementTopLeft, HUDPlacementTopCenter, HUDPlacementTopRight,
		HUDPlacementLeftCenter, HUDPlacementRightCenter, HUDPlacementBottomLeft,
		HUDPlacementBottomCenter, HUDPlacementBottomRight, HUDPlacementCustom:
	default:
		state.Placement = defaults.Placement
	}
	if strings.TrimSpace(state.DisplayID) == "" {
		state.DisplayID = defaults.DisplayID
	}
	if state.Width < 280 || state.Width > 1200 {
		state.Width = defaults.Width
	}
	if state.Height < 48 || state.Height > 900 {
		state.Height = defaults.Height
	}
	if state.X < 0 {
		state.X = 0
	}
	if state.Y < 0 {
		state.Y = 0
	}
	if state.Opacity <= 0 || state.Opacity > 1 {
		state.Opacity = defaults.Opacity
	}
	if strings.TrimSpace(state.Mode) == "" {
		state.Mode = defaults.Mode
	}
	return state
}

func (m *HUDWindowStateManager) Load() HUDWindowState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return defaultHUDWindowState()
	}
	var state HUDWindowState
	if err := json.Unmarshal(data, &state); err != nil {
		return defaultHUDWindowState()
	}
	return normalizeHUDWindowState(state)
}

func (m *HUDWindowStateManager) Save(state HUDWindowState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	state = normalizeHUDWindowState(state)
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.filePath, data, 0o600)
}

func DefaultHUDPlatformCapabilities() HUDPlatformCapabilities {
	caps := HUDPlatformCapabilities{
		FramelessWindow:     true,
		AlwaysOnTop:         true,
		NativeNotifications: DefaultCapabilities().Notifications,
		Platform:            runtime.GOOS,
	}

	switch runtime.GOOS {
	case "windows":
		caps.TransparentWindow = true
		caps.GlobalPositioning = true
		// Wails v2 does not expose stable display identity/work-area/DPI through
		// the application contract used by Nexus HUD. Keep multi-monitor
		// fail-closed until native same-SHA certification adds an adapter.
		caps.MultiMonitor = false
	case "darwin":
		caps.TransparentWindow = true
		caps.GlobalPositioning = true
		caps.MultiMonitor = false
	case "linux":
		session := strings.ToLower(strings.TrimSpace(os.Getenv("XDG_SESSION_TYPE")))
		if session == "" && strings.TrimSpace(os.Getenv("WAYLAND_DISPLAY")) != "" {
			session = "wayland"
		}
		if session == "" && strings.TrimSpace(os.Getenv("DISPLAY")) != "" {
			session = "x11"
		}
		caps.Session = session
		caps.TransparentWindow = false
		caps.GlobalPositioning = session != "wayland"
		caps.MultiMonitor = false
		caps.AutoHide = false
	}
	return caps
}

// HUDController exposes presentation-only native window operations.
type HUDController struct {
	mu      sync.RWMutex
	ctx     context.Context
	manager *HUDWindowStateManager
	caps    HUDPlatformCapabilities
}

func NewHUDController(manager *HUDWindowStateManager) *HUDController {
	if manager == nil {
		manager = NewHUDWindowStateManager("")
	}
	return &HUDController{manager: manager, caps: DefaultHUDPlatformCapabilities()}
}

func (h *HUDController) Startup(ctx context.Context) {
	h.mu.Lock()
	h.ctx = ctx
	h.mu.Unlock()
}

func (h *HUDController) GetHUDCapabilities() HUDPlatformCapabilities {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.caps
}

func (h *HUDController) GetHUDWindowState() HUDWindowState {
	return h.manager.Load()
}

func (h *HUDController) SaveHUDWindowState(state HUDWindowState) error {
	state = normalizeHUDWindowState(state)
	h.mu.RLock()
	ctx := h.ctx
	caps := h.caps
	h.mu.RUnlock()
	if ctx != nil {
		wailsruntime.WindowSetSize(ctx, state.Width, state.Height)
		if caps.GlobalPositioning {
			wailsruntime.WindowSetPosition(ctx, state.X, state.Y)
		}
		wailsruntime.WindowSetAlwaysOnTop(ctx, state.AlwaysOnTop)
	}
	return h.manager.Save(state)
}

func (h *HUDController) RestoreHUDWindow() {
	h.mu.RLock()
	ctx := h.ctx
	caps := h.caps
	h.mu.RUnlock()
	if ctx == nil {
		return
	}
	state := h.manager.Load()
	wailsruntime.WindowSetSize(ctx, state.Width, state.Height)
	if caps.GlobalPositioning {
		wailsruntime.WindowSetPosition(ctx, state.X, state.Y)
	}
	wailsruntime.WindowSetAlwaysOnTop(ctx, state.AlwaysOnTop)
}

func (h *HUDController) CaptureHUDWindowState() error {
	h.mu.RLock()
	ctx := h.ctx
	caps := h.caps
	h.mu.RUnlock()
	if ctx == nil {
		return nil
	}
	state := h.manager.Load()
	width, height := wailsruntime.WindowGetSize(ctx)
	state.Width = width
	state.Height = height
	if caps.GlobalPositioning {
		x, y := wailsruntime.WindowGetPosition(ctx)
		state.X = x
		state.Y = y
		state.Placement = HUDPlacementCustom
	}
	return h.manager.Save(state)
}

func (h *HUDController) SetHUDExpanded(expanded bool) error {
	h.mu.RLock()
	ctx := h.ctx
	h.mu.RUnlock()
	if ctx == nil {
		return nil
	}
	state := h.manager.Load()
	if expanded {
		state.Height = 520
	} else {
		state.Height = 64
	}
	wailsruntime.WindowSetSize(ctx, state.Width, state.Height)
	return h.manager.Save(state)
}

func (h *HUDController) HideHUD() {
	_ = h.CaptureHUDWindowState()
	h.mu.RLock()
	ctx := h.ctx
	h.mu.RUnlock()
	if ctx != nil {
		wailsruntime.WindowHide(ctx)
	}
}

func (h *HUDController) ShowHUD() {
	h.mu.RLock()
	ctx := h.ctx
	h.mu.RUnlock()
	if ctx != nil {
		wailsruntime.WindowShow(ctx)
	}
}
