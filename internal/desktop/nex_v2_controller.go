package desktop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// NexV2PresenceState persists user preferences for the Nex 2.0 shell in its
// own file (nex-v2-state.json), never mutating legacy hud-window-state.json.
type NexV2PresenceState struct {
	Placement         NexV2PlacementState `json:"placement"`
	ShowAvatar        bool                `json:"showAvatar"`
	CompactBar        bool                `json:"compactBar"`
	DoNotDisturb      bool                `json:"doNotDisturb"`
	PrivacyMode       bool                `json:"privacyMode"`
	Paused            bool                `json:"paused"`
	ReducedMotion     bool                `json:"reducedMotion"`
	Hotkey            string              `json:"hotkey"`
	RespectFullscreen bool                `json:"respectFullscreen"`
}

// NexV2PlatformCapabilities reports honest OS/compositor capabilities so the UI
// adapts cleanly on Wayland without invasive hacks.
type NexV2PlatformCapabilities struct {
	GlobalPositioning  bool   `json:"globalPositioning"`
	ClickThrough       bool   `json:"clickThrough"`
	GlobalHotkey       bool   `json:"globalHotkey"`
	GlobalHotkeyReason string `json:"globalHotkeyReason,omitempty"`
	SecondaryWindow    bool   `json:"secondaryWindow"`
	Tray               bool   `json:"tray"`
	AutoStart          bool   `json:"autoStart"`
	Platform           string `json:"platform"`
	Session            string `json:"session,omitempty"`
}

func DefaultNexV2PresenceState() NexV2PresenceState {
	return NexV2PresenceState{
		Placement:         DefaultNexV2PlacementState(),
		ShowAvatar:        true,
		CompactBar:        false,
		DoNotDisturb:      false,
		Paused:            false,
		ReducedMotion:     false,
		Hotkey:            "Ctrl+Shift+N",
		RespectFullscreen: true,
	}
}

func normalizeNexV2PresenceState(state NexV2PresenceState) NexV2PresenceState {
	if _, ok := ResolveNexV2Preset(state.Placement.PresetID); !ok {
		state.Placement.PresetID = "standard"
	}
	switch state.Placement.Anchor {
	case NexV2AnchorTopLeft, NexV2AnchorTopRight, NexV2AnchorBottomLeft, NexV2AnchorBottomRight, NexV2AnchorCustom:
	default:
		state.Placement.Anchor = NexV2AnchorBottomRight
	}
	state.Placement.ScaleFactor = NormalizeScaleFactor(state.Placement.ScaleFactor)
	if strings.TrimSpace(state.Hotkey) == "" {
		state.Hotkey = "Ctrl+Shift+N"
	}
	return state
}

// DefaultNexV2Capabilities resolves conservative platform capabilities before
// Wails startup completes.
func DefaultNexV2Capabilities() NexV2PlatformCapabilities {
	caps := NexV2PlatformCapabilities{
		Tray:            true,
		AutoStart:       runtime.GOOS == "linux" || runtime.GOOS == "windows" || runtime.GOOS == "darwin",
		Platform:        runtime.GOOS,
		SecondaryWindow: true,
	}
	switch runtime.GOOS {
	case "windows", "darwin":
		caps.GlobalPositioning = true
		caps.ClickThrough = runtime.GOOS == "darwin" || hudHitTestingSupported()
		caps.GlobalHotkeyReason = "Use tray or in-app shortcut; native global hotkey hook requires platform registration."
	case "linux":
		caps.Session = hudSession()
		if caps.Session == "wayland" {
			caps.GlobalPositioning = false
			caps.ClickThrough = false
			caps.SecondaryWindow = false
			caps.GlobalHotkey = false
			caps.GlobalHotkeyReason = "Global hotkeys, positioning, and click-through are restricted by the Wayland compositor; use tray recovery or anchored panel."
		} else {
			caps.GlobalPositioning = true
			caps.SecondaryWindow = true
			caps.GlobalHotkeyReason = "Use tray menu or window shortcut; global X11 hotkey hook is not registered by default."
		}
	}
	return caps
}

// NexV2StateManager reads and writes nex-v2-state.json atomically.
type NexV2StateManager struct {
	mu       sync.RWMutex
	filePath string
}

func NewNexV2StateManager(filePath string) *NexV2StateManager {
	if strings.TrimSpace(filePath) == "" {
		dataDir, err := config.DataDir()
		if err != nil || strings.TrimSpace(dataDir) == "" {
			dataDir = filepath.Join(os.TempDir(), "iapro-nexus")
		}
		filePath = filepath.Join(dataDir, "nex-v2-state.json")
	}
	return &NexV2StateManager{filePath: filePath}
}

func (m *NexV2StateManager) Load() NexV2PresenceState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	raw, err := os.ReadFile(m.filePath)
	if err != nil {
		return DefaultNexV2PresenceState()
	}
	var state NexV2PresenceState
	if err := json.Unmarshal(raw, &state); err != nil {
		return DefaultNexV2PresenceState()
	}
	return normalizeNexV2PresenceState(state)
}

func (m *NexV2StateManager) Save(state NexV2PresenceState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	state = normalizeNexV2PresenceState(state)
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.filePath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, m.filePath)
}

// NexV2WindowHooks decouples controller logic from live Wails windows for
// deterministic unit and concurrency testing.
type NexV2WindowHooks struct {
	ApplyRect      func(rect application.Rect)
	SetAlwaysOnTop func(enabled bool)
	SetVisible     func(visible bool)
	GetDisplays    func() []DisplayDescriptor
	GetRect        func() application.Rect
}

// NexV2Controller manages Nex 2.0 shell presence, geometry, and settings
// separately from React model updates.
type NexV2Controller struct {
	mu           sync.Mutex
	manager      *NexV2StateManager
	caps         NexV2PlatformCapabilities
	panelOpen    bool
	settingsOpen bool
	overlayOpen  bool
	focused      bool
	hooks        NexV2WindowHooks
}

func NewNexV2Controller(manager *NexV2StateManager, caps NexV2PlatformCapabilities, hooks NexV2WindowHooks) *NexV2Controller {
	if manager == nil {
		manager = NewNexV2StateManager("")
	}
	return &NexV2Controller{
		manager: manager,
		caps:    caps,
		hooks:   hooks,
	}
}

func (c *NexV2Controller) GetPresenceState() NexV2PresenceState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.manager.Load()
}

func (c *NexV2Controller) GetCapabilities() NexV2PlatformCapabilities {
	c.mu.Lock()
	defer c.mu.Unlock()
	caps := c.caps
	if c.hooks.GetDisplays != nil {
		_ = c.hooks.GetDisplays()
	}
	return caps
}

func (c *NexV2Controller) GetPanelOpen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.panelOpen
}

// ResolveNexV2SurfaceRect computes the bounded native window rect for collapsed,
// compact-bar, or expanded-panel states.
func ResolveNexV2SurfaceRect(
	state NexV2PresenceState,
	panelOpen bool,
	displays []DisplayDescriptor,
	globalPositioning bool,
) application.Rect {
	return ResolveNexV2SurfaceRectForState(state, panelOpen, false, false, displays, globalPositioning)
}

// ResolveNexV2SurfaceRectForState computes the bounded native window rect across
// all Nex 2.0 shell states (collapsed square avatar, compact bar, overlay
// banner/notification, settings drawer, or expanded panel).
func ResolveNexV2SurfaceRectForState(
	state NexV2PresenceState,
	panelOpen bool,
	settingsOpen bool,
	overlayOpen bool,
	displays []DisplayDescriptor,
	globalPositioning bool,
) application.Rect {
	baseRect, display, _ := ResolveNexV2WindowRect(state.Placement, displays, globalPositioning)
	effectiveCompactBar := state.CompactBar || !state.ShowAvatar
	hasOverlay := overlayOpen && state.ShowAvatar
	if !panelOpen && !settingsOpen && !effectiveCompactBar && !hasOverlay {
		return baseRect
	}

	var targetW, targetH int
	switch {
	case panelOpen || settingsOpen:
		targetW, targetH = 440, 540
		if state.ShowAvatar && baseRect.Height > 160 {
			targetH += baseRect.Height - 160
		}
		if state.ShowAvatar && state.CompactBar {
			targetH += 48
		}
		if targetH > 620 {
			targetH = 620
		}
	case effectiveCompactBar:
		targetW = 360
		if state.ShowAvatar {
			targetH = baseRect.Height + 64
		} else {
			targetH = 56
		}
	default:
		// Collapsed avatar with notification bubble or offline banner.
		targetW = 360
		targetH = baseRect.Height + 96
	}

	if !globalPositioning {
		return application.Rect{
			X:      0,
			Y:      0,
			Width:  targetW,
			Height: targetH,
		}
	}

	work := display.WorkArea
	x := baseRect.X
	y := baseRect.Y
	switch state.Placement.Anchor {
	case NexV2AnchorTopRight, NexV2AnchorBottomRight:
		x = baseRect.X + baseRect.Width - targetW
	}
	switch state.Placement.Anchor {
	case NexV2AnchorBottomLeft, NexV2AnchorBottomRight:
		y = baseRect.Y + baseRect.Height - targetH
	}
	if work.Width > 0 && x+targetW > work.X+work.Width {
		x = baseRect.X + baseRect.Width - targetW
	}
	if work.Height > 0 && y+targetH > work.Y+work.Height {
		y = baseRect.Y + baseRect.Height - targetH
	}
	return clampRectToWorkArea(application.Rect{
		X:      x,
		Y:      y,
		Width:  targetW,
		Height: targetH,
	}, work)
}

func (c *NexV2Controller) applySurfaceRectLocked(state NexV2PresenceState, displays []DisplayDescriptor) {
	if c.hooks.ApplyRect == nil {
		return
	}
	rect := ResolveNexV2SurfaceRectForState(
		state,
		c.panelOpen,
		c.settingsOpen,
		c.overlayOpen,
		displays,
		c.caps.GlobalPositioning,
	)
	c.hooks.ApplyRect(rect)
}

// SetPanelOpen serializes panel open/close requests to prevent toggle races.
func (c *NexV2Controller) SetPanelOpen(open bool) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.panelOpen = open
	c.applySurfaceRectLocked(c.manager.Load(), c.currentDisplaysLocked())
	return c.panelOpen, nil
}

// SetSettingsOpen expands or restores the native surface when the Nex 2.0
// settings drawer opens or closes.
func (c *NexV2Controller) SetSettingsOpen(open bool) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.settingsOpen = open
	c.applySurfaceRectLocked(c.manager.Load(), c.currentDisplaysLocked())
	return c.settingsOpen, nil
}

// SetOverlayVisible expands or restores the collapsed surface for transient
// overlays (notification bubbles or offline banners).
func (c *NexV2Controller) SetOverlayVisible(visible bool) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.overlayOpen = visible
	c.applySurfaceRectLocked(c.manager.Load(), c.currentDisplaysLocked())
	return c.overlayOpen, nil
}

func (c *NexV2Controller) SetPreset(presetID string) (NexV2PresenceState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := ResolveNexV2Preset(presetID); !ok {
		return c.manager.Load(), fmt.Errorf("unknown Nex 2.0 preset %q", presetID)
	}
	state := c.manager.Load()
	displays := c.currentDisplaysLocked()
	currentRect, display, _ := ResolveNexV2WindowRect(state.Placement, displays, c.caps.GlobalPositioning)
	collapsedAvatar := !c.panelOpen && !c.settingsOpen && !c.overlayOpen && !state.CompactBar && state.ShowAvatar
	if collapsedAvatar && c.hooks.GetRect != nil {
		r := c.hooks.GetRect()
		if r.Width > 0 && r.Height > 0 {
			currentRect = r
		}
	}
	_, nextPlacement, _ := SwitchNexV2Preset(currentRect, presetID, display, c.caps.GlobalPositioning)
	nextPlacement.AlwaysOnTop = state.Placement.AlwaysOnTop
	state.Placement = nextPlacement
	if err := c.manager.Save(state); err != nil {
		return state, err
	}
	c.applySurfaceRectLocked(state, displays)
	return state, nil
}

func (c *NexV2Controller) SetAlwaysOnTop(enabled bool) (NexV2PresenceState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.manager.Load()
	state.Placement.AlwaysOnTop = enabled
	if err := c.manager.Save(state); err != nil {
		return state, err
	}
	if c.hooks.SetAlwaysOnTop != nil {
		c.hooks.SetAlwaysOnTop(enabled)
	}
	return state, nil
}

func (c *NexV2Controller) SetShowAvatar(visible bool) (NexV2PresenceState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.manager.Load()
	state.ShowAvatar = visible
	if err := c.manager.Save(state); err != nil {
		return state, err
	}
	c.applySurfaceRectLocked(state, c.currentDisplaysLocked())
	if c.hooks.SetVisible != nil {
		c.hooks.SetVisible(visible || state.CompactBar || c.panelOpen || c.settingsOpen)
	}
	return state, nil
}

func (c *NexV2Controller) SetCompactBar(enabled bool) (NexV2PresenceState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.manager.Load()
	state.CompactBar = enabled
	if err := c.manager.Save(state); err != nil {
		return state, err
	}
	c.applySurfaceRectLocked(state, c.currentDisplaysLocked())
	if c.hooks.SetVisible != nil {
		c.hooks.SetVisible(state.ShowAvatar || enabled || c.panelOpen || c.settingsOpen)
	}
	return state, nil
}

// SetDisplayMode atomically switches between Avatar (showAvatar=true, compactBar=false),
// Bar (showAvatar=false, compactBar=true), and Both (showAvatar=true, compactBar=true)
// without an intermediate hidden frame or size mismatch.
func (c *NexV2Controller) SetDisplayMode(showAvatar bool, compactBar bool) (NexV2PresenceState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.manager.Load()
	state.ShowAvatar = showAvatar
	state.CompactBar = compactBar
	if err := c.manager.Save(state); err != nil {
		return state, err
	}
	c.applySurfaceRectLocked(state, c.currentDisplaysLocked())
	if c.hooks.SetVisible != nil {
		c.hooks.SetVisible(state.ShowAvatar || state.CompactBar || c.panelOpen || c.settingsOpen)
	}
	return state, nil
}

func (c *NexV2Controller) SetDoNotDisturb(enabled bool) (NexV2PresenceState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.manager.Load()
	state.DoNotDisturb = enabled
	return state, c.manager.Save(state)
}

func (c *NexV2Controller) SetPrivacyMode(enabled bool) (NexV2PresenceState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.manager.Load()
	state.PrivacyMode = enabled
	return state, c.manager.Save(state)
}

func (c *NexV2Controller) SetPaused(paused bool) (NexV2PresenceState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.manager.Load()
	state.Paused = paused
	return state, c.manager.Save(state)
}

func (c *NexV2Controller) SetReducedMotion(enabled bool) (NexV2PresenceState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.manager.Load()
	state.ReducedMotion = enabled
	return state, c.manager.Save(state)
}

func (c *NexV2Controller) MoveByDelta(dx, dy int) (NexV2PlacementState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.manager.Load()
	if !c.caps.GlobalPositioning {
		return state.Placement, fmt.Errorf("global positioning unavailable on %s/%s", c.caps.Platform, c.caps.Session)
	}
	displays := c.currentDisplaysLocked()
	rect, display, _ := ResolveNexV2WindowRect(state.Placement, displays, true)
	rect.X += dx
	rect.Y += dy
	rect = clampRectToWorkArea(rect, display.WorkArea)
	state.Placement.DisplayID = display.ID
	state.Placement.Anchor = NexV2AnchorCustom
	state.Placement.OffsetX = rect.X - display.WorkArea.X
	state.Placement.OffsetY = rect.Y - display.WorkArea.Y
	state.Placement.ScaleFactor = display.ScaleFactor
	if err := c.manager.Save(state); err != nil {
		return state.Placement, err
	}
	if c.hooks.ApplyRect != nil {
		c.hooks.ApplyRect(rect)
	}
	return state.Placement, nil
}

// HandleDisplayChange re-resolves and clamps window geometry after hotplug or
// suspend/resume without losing the user's preferred preset.
func (c *NexV2Controller) HandleDisplayChange(displays []DisplayDescriptor) NexV2PresenceState {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.manager.Load()
	rect, display, fellBack := ResolveNexV2WindowRect(state.Placement, displays, c.caps.GlobalPositioning)
	if fellBack || state.Placement.DisplayID == "" {
		state.Placement.DisplayID = display.ID
	}
	state.Placement.ScaleFactor = display.ScaleFactor
	if c.caps.GlobalPositioning {
		state.Placement.OffsetX = rect.X - display.WorkArea.X
		state.Placement.OffsetY = rect.Y - display.WorkArea.Y
	}
	_ = c.manager.Save(state)
	c.applySurfaceRectLocked(state, displays)
	return state
}

// HandleFocusChange records focus transitions without triggering blur-time
// placement corruption or forced hiding.
func (c *NexV2Controller) HandleFocusChange(focused bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.focused = focused
}

func (c *NexV2Controller) currentDisplaysLocked() []DisplayDescriptor {
	if c.hooks.GetDisplays != nil {
		return c.hooks.GetDisplays()
	}
	return nil
}
