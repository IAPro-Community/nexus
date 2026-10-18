package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
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
	AutoStart           bool   `json:"autoStart"`
	DisplayCount        int    `json:"displayCount"`
	Platform            string `json:"platform"`
	Session             string `json:"session,omitempty"`
}

type HUDWindowState struct {
	DisplayID    string  `json:"displayId"`
	DisplayScale float32 `json:"displayScale,omitempty"`
	Placement    string  `json:"placement"`
	X            int     `json:"x"`
	Y            int     `json:"y"`
	Width        int     `json:"width"`
	Height       int     `json:"height"`
	Mode         string  `json:"mode"`
	Opacity      float64 `json:"opacity"`
	AlwaysOnTop  bool    `json:"alwaysOnTop"`
	AutoHide     bool    `json:"autoHide"`
}

type HUDWindowStateManager struct {
	mu       sync.RWMutex
	filePath string
}

func NewHUDWindowStateManager(filePath string) *HUDWindowStateManager {
	if strings.TrimSpace(filePath) == "" {
		dataDir, err := config.DataDir()
		if err != nil || strings.TrimSpace(dataDir) == "" {
			dataDir = filepath.Join(os.TempDir(), "iapro-nexus")
		}
		filePath = filepath.Join(dataDir, "hud-window-state.json")
	}
	return &HUDWindowStateManager{filePath: filePath}
}

func defaultHUDWindowState() HUDWindowState {
	return HUDWindowState{
		DisplayID:   "current",
		Placement:   HUDPlacementBottomRight,
		X:           24,
		Y:           24,
		Width:       220,
		Height:      60,
		Mode:        "AUTO",
		Opacity:     0.96,
		AlwaysOnTop: true,
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
	if state.Width < 190 || state.Width > 1200 {
		state.Width = defaults.Width
	}
	if state.Height < 48 || state.Height > 900 {
		state.Height = defaults.Height
	}
	if state.Opacity <= 0 || state.Opacity > 1 {
		state.Opacity = defaults.Opacity
	}
	if strings.TrimSpace(state.Mode) == "" {
		state.Mode = defaults.Mode
	}
	return state
}

func clampHUDWindowStateToWorkArea(state HUDWindowState, work application.Rect) HUDWindowState {
	state = normalizeHUDWindowState(state)
	if work.Width <= 0 || work.Height <= 0 {
		return state
	}
	if state.Width > work.Width {
		state.Width = work.Width
	}
	if state.Height > work.Height {
		state.Height = work.Height
	}
	maxX := work.X + work.Width - state.Width
	maxY := work.Y + work.Height - state.Height
	if state.X < work.X {
		state.X = work.X
	}
	if state.Y < work.Y {
		state.Y = work.Y
	}
	if state.X > maxX {
		state.X = maxX
	}
	if state.Y > maxY {
		state.Y = maxY
	}
	return state
}

// Kept for pure geometry tests and callers that work with an origin-zero screen.
func clampHUDWindowStateToScreen(state HUDWindowState, screenWidth, screenHeight int) HUDWindowState {
	return clampHUDWindowStateToWorkArea(state, application.Rect{Width: screenWidth, Height: screenHeight})
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

func hudSession() string {
	session := strings.ToLower(strings.TrimSpace(os.Getenv("XDG_SESSION_TYPE")))
	if session == "" && strings.TrimSpace(os.Getenv("WAYLAND_DISPLAY")) != "" {
		session = "wayland"
	}
	if session == "" && strings.TrimSpace(os.Getenv("DISPLAY")) != "" {
		session = "x11"
	}
	return session
}

func DefaultHUDPlatformCapabilities() HUDPlatformCapabilities {
	caps := HUDPlatformCapabilities{
		FramelessWindow:     true,
		AlwaysOnTop:         true,
		NativeNotifications: DefaultCapabilities().Notifications,
		Platform:            runtime.GOOS,
	}
	switch runtime.GOOS {
	case "windows", "darwin":
		caps.TransparentWindow = true
		caps.GlobalPositioning = true
	case "linux":
		caps.Session = hudSession()
		caps.GlobalPositioning = caps.Session != "wayland"
	}
	return caps
}

// HUDController exposes presentation-only native window operations.
type HUDController struct {
	mu      sync.RWMutex
	manager *HUDWindowStateManager
	caps    HUDPlatformCapabilities
	ready   bool
}

func NewHUDController(manager *HUDWindowStateManager) *HUDController {
	if manager == nil {
		manager = NewHUDWindowStateManager("")
	}
	return &HUDController{manager: manager, caps: DefaultHUDPlatformCapabilities()}
}

func (h *HUDController) nativeWindow() (application.Window, bool) {
	app := application.Get()
	if app == nil {
		return nil, false
	}
	return app.Window.GetByName(HUDWindowName)
}

func (h *HUDController) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	h.mu.Lock()
	h.ready = true
	h.mu.Unlock()
	if window, ok := h.nativeWindow(); ok {
		window.OnWindowEvent(events.Common.WindowLostFocus, func(_ *application.WindowEvent) {
			if h.manager.Load().AutoHide {
				_ = h.CaptureHUDWindowState()
				window.Hide()
			}
		})
	}
	h.RestoreHUDWindow()
	return nil
}

func (h *HUDController) ServiceShutdown() error {
	_ = h.CaptureHUDWindowState()
	h.mu.Lock()
	h.ready = false
	h.mu.Unlock()
	return nil
}

func (h *HUDController) GetHUDCapabilities() HUDPlatformCapabilities {
	h.mu.RLock()
	caps := h.caps
	ready := h.ready
	h.mu.RUnlock()
	app := application.Get()
	if ready && app != nil {
		screens := app.Screen.GetAll()
		caps.MultiMonitor = len(screens) > 0
		caps.DisplayCount = len(screens)
		caps.Tray = true
		caps.AutoHide = true
		caps.AutoStart = runtime.GOOS == "linux" || runtime.GOOS == "windows" || runtime.GOOS == "darwin"
	}
	return caps
}

func (h *HUDController) GetHUDWindowState() HUDWindowState {
	return h.manager.Load()
}

func (h *HUDController) screenForState(state HUDWindowState, window application.Window) *application.Screen {
	app := application.Get()
	if app == nil {
		return nil
	}
	if state.DisplayID != "" && state.DisplayID != "current" {
		if screen := app.Screen.GetByID(state.DisplayID); screen != nil {
			return screen
		}
	}
	if window != nil {
		x, y := window.Position()
		w, height := window.Size()
		if screen := app.Screen.ScreenNearestDipRect(application.Rect{X: x, Y: y, Width: w, Height: height}); screen != nil {
			return screen
		}
	}
	return app.Screen.GetPrimary()
}

func (h *HUDController) SaveHUDWindowState(state HUDWindowState) error {
	state = normalizeHUDWindowState(state)
	window, ok := h.nativeWindow()
	if ok {
		screen := h.screenForState(state, window)
		if screen != nil {
			state = clampHUDWindowStateToWorkArea(state, screen.WorkArea)
			state.DisplayID = screen.ID
			state.DisplayScale = screen.ScaleFactor
		}
		window.SetSize(state.Width, state.Height)
		if h.GetHUDCapabilities().GlobalPositioning {
			window.SetPosition(state.X, state.Y)
		}
		window.SetAlwaysOnTop(state.AlwaysOnTop)
	}
	return h.manager.Save(state)
}

// The React HUD initializes collapsed. Persisted expanded geometry must never
// survive a process restart as an invisible, oversized click-capturing window.
// Preserve the bottom-right anchor for custom placements.
func compactHUDWindowState(state HUDWindowState) HUDWindowState {
  state = normalizeHUDWindowState(state)
  constWidth, constHeight := 220, 60
  state.X += state.Width - constWidth
  state.Y += state.Height - constHeight
  state.Width, state.Height = constWidth, constHeight
  return state
}

func (h *HUDController) RestoreHUDWindow() {
	window, ok := h.nativeWindow()
	if !ok {
		return
	}
	state := compactHUDWindowState(h.manager.Load())
	screen := h.screenForState(state, window)
	if screen != nil {
		// Named placements are resolved against the usable screen area, not
		// absolute desktop coordinates. This keeps the capsule above taskbars.
		if state.Placement == HUDPlacementBottomRight {
			state.X = screen.WorkArea.X + screen.WorkArea.Width - state.Width - 12
			state.Y = screen.WorkArea.Y + screen.WorkArea.Height - state.Height - 8
		}
		state = clampHUDWindowStateToWorkArea(state, screen.WorkArea)
		state.DisplayID = screen.ID
		state.DisplayScale = screen.ScaleFactor
		_ = h.manager.Save(state)
	}
	window.SetSize(state.Width, state.Height)
	if h.GetHUDCapabilities().GlobalPositioning {
		window.SetPosition(state.X, state.Y)
	}
	window.SetAlwaysOnTop(state.AlwaysOnTop)
}

func (h *HUDController) CaptureHUDWindowState() error {
	window, ok := h.nativeWindow()
	if !ok {
		return nil
	}
	state := h.manager.Load()
	state.Width, state.Height = window.Size()
	state.X, state.Y = window.Position()
	state.Placement = HUDPlacementCustom
	if screen := h.screenForState(state, window); screen != nil {
		state = clampHUDWindowStateToWorkArea(state, screen.WorkArea)
		state.DisplayID = screen.ID
		state.DisplayScale = screen.ScaleFactor
	}
	return h.manager.Save(state)
}

func (h *HUDController) SetHUDExpanded(expanded bool) error {
	window, ok := h.nativeWindow()
	state := h.manager.Load()
	targetWidth, targetHeight := 220, 60
	if expanded {
		targetWidth, targetHeight = 334, 496
	}

	// Native size is authoritative: a blur/capture may have persisted a
	// different size between two user clicks. Always grow up/left from the
	// *actual* window bottom-right rather than shifting from a stale snapshot.
	oldWidth, oldHeight := state.Width, state.Height
	if ok {
		state.X, state.Y = window.Position()
		oldWidth, oldHeight = window.Size()
	}
	state.X += oldWidth - targetWidth
	state.Y += oldHeight - targetHeight
	state.Width, state.Height = targetWidth, targetHeight
	if ok {
		if screen := h.screenForState(state, window); screen != nil {
			state = clampHUDWindowStateToWorkArea(state, screen.WorkArea)
			state.DisplayID = screen.ID
			state.DisplayScale = screen.ScaleFactor
		}
		window.SetSize(state.Width, state.Height)
		if h.GetHUDCapabilities().GlobalPositioning {
			window.SetPosition(state.X, state.Y)
		}
	}
	state.Placement = HUDPlacementCustom
	return h.manager.Save(state)
}

func (h *HUDController) SetHUDAutoHide(enabled bool) error {
	state := h.manager.Load()
	state.AutoHide = enabled
	return h.manager.Save(state)
}

func (h *HUDController) HideHUD() {
	_ = h.CaptureHUDWindowState()
	if window, ok := h.nativeWindow(); ok {
		window.Hide()
	}
}

func (h *HUDController) ShowHUD() {
	if window, ok := h.nativeWindow(); ok {
		window.Show()
		window.Restore()
		window.Focus()
	}
}

func (h *HUDController) GetHUDAutoStartEnabled() bool {
	app := application.Get()
	if app == nil {
		return false
	}
	enabled, err := app.Autostart.IsEnabled()
	return err == nil && enabled
}

func (h *HUDController) SetHUDAutoStartEnabled(enabled bool) error {
	app := application.Get()
	if app == nil {
		return fmt.Errorf("HUD autostart is unavailable before Wails startup")
	}
	if enabled {
		return app.Autostart.Enable()
	}
	return app.Autostart.Disable()
}
