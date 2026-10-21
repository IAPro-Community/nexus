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
	HUDPopupWindowName       = "Nexus HUD Popup"
	HUDAppearanceBar         = "BAR"
	HUDAppearanceMascot      = "MASCOT"
	HUDAppearanceTray        = "TRAY"
	HUDMascotWidth           = 108
	HUDMascotHeight          = 112
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
	DisplayID      string  `json:"displayId"`
	DisplayScale   float32 `json:"displayScale,omitempty"`
	Placement      string  `json:"placement"`
	X              int     `json:"x"`
	Y              int     `json:"y"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	CompactWidth   int     `json:"compactWidth"`
	CompactHeight  int     `json:"compactHeight"`
	ExpandedWidth  int     `json:"expandedWidth"`
	ExpandedHeight int     `json:"expandedHeight"`
	Mode           string  `json:"mode"`
	Appearance     string  `json:"appearance"`
	Opacity        float64 `json:"opacity"`
	AlwaysOnTop    bool    `json:"alwaysOnTop"`
	AutoHide       bool    `json:"autoHide"`
}

// HUDSizePreset is the native window geometry contract exposed to the HUD
// settings surface. Compact and expanded sizes travel together so a user can
// choose one preset without producing a popover that no longer fits.
type HUDSizePreset struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	CompactWidth   int    `json:"compactWidth"`
	CompactHeight  int    `json:"compactHeight"`
	ExpandedWidth  int    `json:"expandedWidth"`
	ExpandedHeight int    `json:"expandedHeight"`
}

var hudSizePresets = []HUDSizePreset{
	{ID: "compact", Label: "Compacto", CompactWidth: 220, CompactHeight: 60, ExpandedWidth: 334, ExpandedHeight: 496},
	{ID: "comfortable", Label: "Confortável", CompactWidth: 260, CompactHeight: 70, ExpandedWidth: 380, ExpandedHeight: 560},
	{ID: "large", Label: "Grande", CompactWidth: 320, CompactHeight: 84, ExpandedWidth: 440, ExpandedHeight: 640},
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
		DisplayID:      "current",
		Placement:      HUDPlacementBottomRight,
		X:              24,
		Y:              24,
		Width:          220,
		Height:         60,
		CompactWidth:   220,
		CompactHeight:  60,
		ExpandedWidth:  334,
		ExpandedHeight: 496,
		Mode:           "AUTO",
		Appearance:     HUDAppearanceBar,
		Opacity:        0.96,
		AlwaysOnTop:    true,
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
	if state.Width < 80 || state.Width > 1200 {
		state.Width = defaults.Width
	}
	if state.Height < 48 || state.Height > 900 {
		state.Height = defaults.Height
	}
	if state.CompactWidth < 190 || state.CompactWidth > 600 {
		state.CompactWidth = defaults.CompactWidth
	}
	if state.CompactHeight < 48 || state.CompactHeight > 300 {
		state.CompactHeight = defaults.CompactHeight
	}
	if state.ExpandedWidth < 280 || state.ExpandedWidth > 1200 {
		state.ExpandedWidth = defaults.ExpandedWidth
	}
	if state.ExpandedHeight < 240 || state.ExpandedHeight > 900 {
		state.ExpandedHeight = defaults.ExpandedHeight
	}
	if state.ExpandedWidth < state.CompactWidth {
		state.ExpandedWidth = state.CompactWidth
	}
	if state.ExpandedHeight < state.CompactHeight {
		state.ExpandedHeight = state.CompactHeight
	}
	if state.Opacity <= 0 || state.Opacity > 1 {
		state.Opacity = defaults.Opacity
	}
	if strings.TrimSpace(state.Mode) == "" {
		state.Mode = defaults.Mode
	}
	if !validHUDAppearance(state.Appearance) {
		state.Appearance = defaults.Appearance
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
	// Detached mode is configured at process startup and is never toggled
	// from the browser. The legacy single-window path remains the fallback.
	detached bool
	expanded bool
}

func NewHUDController(manager *HUDWindowStateManager) *HUDController {
	return NewHUDControllerWithDetachedPopover(manager, false)
}

// NewHUDControllerWithDetachedPopover enables a second Wails window without
// altering the legacy HUD when the feature is disabled. Wayland has no portable
// global positioning protocol, so the split mode remains disabled there.
func NewHUDControllerWithDetachedPopover(manager *HUDWindowStateManager, detached bool) *HUDController {
	if manager == nil {
		manager = NewHUDWindowStateManager("")
	}
	if runtime.GOOS == "linux" && hudSession() == "wayland" {
		detached = false
	}
	// If detached mode is disabled (including Wayland), restore a usable bar
	// instead of rendering 220 DIP content inside a 108 DIP mascot window.
	if !detached {
		state := manager.Load()
		if state.Appearance == HUDAppearanceMascot {
			state.Appearance = HUDAppearanceBar
			state.Width, state.Height = state.CompactWidth, state.CompactHeight
			_ = manager.Save(state)
		}
	}
	return &HUDController{manager: manager, caps: DefaultHUDPlatformCapabilities(), detached: detached}
}

func (h *HUDController) GetHUDDetachedPopoverEnabled() bool {
	return h.detached
}

func (h *HUDController) GetHUDExpanded() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.expanded
}

// NotifyHUDPopupHidden keeps the controller in sync when the OS dismisses the
// popup (for example Escape). It intentionally does not call Hide again:
// WindowHide must never recurse into itself.
func (h *HUDController) NotifyHUDPopupHidden() {
	if h.detached {
		h.setHUDExpandedState(false)
	}
}

func (h *HUDController) setHUDExpandedState(expanded bool) {
	h.mu.Lock()
	h.expanded = expanded
	h.mu.Unlock()
}

// closeDetachedHUD is called on popup focus loss. It must never collapse or
// hide the compact capsule, even if auto-hide is configured.
func (h *HUDController) closeDetachedHUD() {
	if !h.detached {
		return
	}
	h.setHUDExpandedState(false)
	if popup, ok := h.detachedPopupWindow(); ok {
		popup.Hide()
	}
}

func (h *HUDController) detachedPopupWindow() (application.Window, bool) {
	if !h.detached {
		return nil, false
	}
	app := application.Get()
	if app == nil {
		return nil, false
	}
	return app.Window.GetByName(HUDPopupWindowName)
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
			// Switching focus from capsule to its detached popup must not
			// make the capsule disappear.
			if h.manager.Load().AutoHide && !h.GetHUDExpanded() {
				_ = h.CaptureHUDWindowState()
				window.Hide()
			}
		})
	}
	h.RestoreHUDWindow()
	return nil
}

func (h *HUDController) ServiceShutdown() error {
	h.closeDetachedHUD()
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
	compactWidth, compactHeight := hudCompactDimensions(state)
	state.X += state.Width - compactWidth
	state.Y += state.Height - compactHeight
	state.Width, state.Height = compactWidth, compactHeight
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
	if h.detached {
		return h.setHUDDetachedExpanded(expanded)
	}
	window, ok := h.nativeWindow()
	if !ok {
		// The React popup must not be marked expanded inside a 220x60 window.
		return fmt.Errorf("native Nexus HUD window is unavailable")
	}
	state := h.manager.Load()
	targetWidth, targetHeight := state.CompactWidth, state.CompactHeight
	if expanded {
		targetWidth, targetHeight = state.ExpandedWidth, state.ExpandedHeight
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

// GetHUDSizePresets returns the supported native window geometries. The
// frontend only renders these presets; it never invents dimensions.
func (h *HUDController) GetHUDSizePresets() []HUDSizePreset {
	presets := make([]HUDSizePreset, len(hudSizePresets))
	copy(presets, hudSizePresets)
	return presets
}

// SetHUDSizePreset persists and applies a named geometry while keeping the
// bottom-right anchor stable. It is used by the native HUD settings surface.
func (h *HUDController) SetHUDSizePreset(preset string, expanded bool) error {
	if h.detached {
		return h.setDetachedHUDSizePreset(preset)
	}
	window, ok := h.nativeWindow()
	if !ok {
		return fmt.Errorf("native Nexus HUD window is unavailable")
	}
	selected, ok := resolveHUDSizePreset(preset)
	if !ok {
		return fmt.Errorf("unknown HUD size preset %q", preset)
	}
	state := h.manager.Load()
	state.CompactWidth, state.CompactHeight = selected.CompactWidth, selected.CompactHeight
	state.ExpandedWidth, state.ExpandedHeight = selected.ExpandedWidth, selected.ExpandedHeight
	targetWidth, targetHeight := selected.CompactWidth, selected.CompactHeight
	if expanded {
		targetWidth, targetHeight = selected.ExpandedWidth, selected.ExpandedHeight
	}
	state.X, state.Y = window.Position()
	oldWidth, oldHeight := window.Size()
	state.X += oldWidth - targetWidth
	state.Y += oldHeight - targetHeight
	state.Width, state.Height = targetWidth, targetHeight
	if screen := h.screenForState(state, window); screen != nil {
		state = clampHUDWindowStateToWorkArea(state, screen.WorkArea)
		state.DisplayID = screen.ID
		state.DisplayScale = screen.ScaleFactor
	}
	window.SetSize(state.Width, state.Height)
	if h.GetHUDCapabilities().GlobalPositioning {
		window.SetPosition(state.X, state.Y)
	}
	state.Placement = HUDPlacementCustom
	return h.manager.Save(state)
}

// MoveHUD moves the native window by a keyboard-sized delta and clamps it to
// the current work area. Pointer dragging is handled by Wails' native drag
// region; this method keeps the operation accessible from the keyboard.
func (h *HUDController) MoveHUD(dx, dy int) error {
	window, ok := h.nativeWindow()
	if !ok {
		return fmt.Errorf("native Nexus HUD window is unavailable")
	}
	if !h.GetHUDCapabilities().GlobalPositioning {
		return fmt.Errorf("native Nexus HUD positioning is unavailable on this platform")
	}
	state := h.manager.Load()
	state.Width, state.Height = window.Size()
	state.X, state.Y = window.Position()
	state.X += dx
	state.Y += dy
	if screen := h.screenForState(state, window); screen != nil {
		state = clampHUDWindowStateToWorkArea(state, screen.WorkArea)
		state.DisplayID = screen.ID
		state.DisplayScale = screen.ScaleFactor
	}
	window.SetPosition(state.X, state.Y)
	state.Placement = HUDPlacementCustom
	if err := h.manager.Save(state); err != nil {
		return err
	}
	if h.detached && h.GetHUDExpanded() {
		return h.positionDetachedHUDPopup(window)
	}
	return nil
}

func resolveHUDSizePreset(preset string) (HUDSizePreset, bool) {
	for _, item := range hudSizePresets {
		if item.ID == preset {
			return item, true
		}
	}
	return HUDSizePreset{}, false
}

func (h *HUDController) SetHUDAutoHide(enabled bool) error {
	state := h.manager.Load()
	state.AutoHide = enabled
	return h.manager.Save(state)
}

func (h *HUDController) HideHUD() {
	h.closeDetachedHUD()
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
