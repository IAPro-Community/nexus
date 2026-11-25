package desktop

import (
	"math"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	// NexV2WindowName identifies the lightweight Nex 2.0 native avatar surface.
	NexV2WindowName = "Nexus Nex 2.0"

	NexV2AnchorBottomRight = "BOTTOM_RIGHT"
	NexV2AnchorBottomLeft  = "BOTTOM_LEFT"
	NexV2AnchorTopRight    = "TOP_RIGHT"
	NexV2AnchorTopLeft     = "TOP_LEFT"
	NexV2AnchorCustom      = "CUSTOM"

	nexV2DefaultMarginDIP = 16
)

// DisplayDescriptor captures display-relative bounds and DPI scaling for one
// physical monitor. WorkArea is in logical DIPs and may contain negative X/Y.
type DisplayDescriptor struct {
	ID          string           `json:"id"`
	WorkArea    application.Rect `json:"workArea"`
	ScaleFactor float32          `json:"scaleFactor"`
	Primary     bool             `json:"primary"`
}

// NexV2ViewportPreset defines a fixed square viewport that encloses the entire
// Nex motion envelope without resizing during animation or state transitions.
type NexV2ViewportPreset struct {
	ID     string `json:"id"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

var nexV2Presets = []NexV2ViewportPreset{
	{ID: "compact", Width: 128, Height: 128},
	{ID: "standard", Width: 160, Height: 160},
	{ID: "large", Width: 192, Height: 192},
}

// NexV2PlacementState stores display-relative coordinates in logical DIPs so
// hotplug, mixed DPI, and preset changes preserve intent without clipping.
type NexV2PlacementState struct {
	DisplayID   string  `json:"displayId"`
	Anchor      string  `json:"anchor"`
	OffsetX     int     `json:"offsetX"`
	OffsetY     int     `json:"offsetY"`
	PresetID    string  `json:"presetId"`
	AlwaysOnTop bool    `json:"alwaysOnTop"`
	ScaleFactor float32 `json:"scaleFactor,omitempty"`
}

// DefaultNexV2PlacementState returns the default bottom-right placement using
// the standard 160x160 DIP viewport.
func DefaultNexV2PlacementState() NexV2PlacementState {
	return NexV2PlacementState{
		Anchor:      NexV2AnchorBottomRight,
		PresetID:    "standard",
		AlwaysOnTop: true,
		ScaleFactor: 1.0,
	}
}

// NexV2Presets returns a copy of the supported fixed avatar viewport presets.
func NexV2Presets() []NexV2ViewportPreset {
	out := make([]NexV2ViewportPreset, len(nexV2Presets))
	copy(out, nexV2Presets)
	return out
}

// ResolveNexV2Preset returns the named viewport preset or falls back to false.
func ResolveNexV2Preset(id string) (NexV2ViewportPreset, bool) {
	needle := strings.ToLower(strings.TrimSpace(id))
	for _, preset := range nexV2Presets {
		if preset.ID == needle {
			return preset, true
		}
	}
	return NexV2ViewportPreset{}, false
}

// NormalizeScaleFactor clamps missing or non-finite DPI scale values to 1.0.
func NormalizeScaleFactor(scale float32) float32 {
	if scale <= 0 || math.IsNaN(float64(scale)) || math.IsInf(float64(scale), 0) {
		return 1.0
	}
	return scale
}

// LogicalToPhysical converts logical DIPs to physical pixels for a display DPI.
func LogicalToPhysical(dip int, scale float32) int {
	return int(math.Round(float64(dip) * float64(NormalizeScaleFactor(scale))))
}

// PhysicalToLogical converts physical pixels back to logical DIPs.
func PhysicalToLogical(px int, scale float32) int {
	return int(math.Round(float64(px) / float64(NormalizeScaleFactor(scale))))
}

func selectDisplayForState(displayID string, displays []DisplayDescriptor) (DisplayDescriptor, bool) {
	trimmed := strings.TrimSpace(displayID)
	if trimmed != "" && trimmed != "current" {
		for _, d := range displays {
			if d.ID == trimmed {
				return d, false
			}
		}
	}
	for _, d := range displays {
		if d.Primary {
			return d, trimmed != "" && trimmed != "current"
		}
	}
	if len(displays) > 0 {
		return displays[0], trimmed != "" && trimmed != "current"
	}
	return DisplayDescriptor{
		ID:          "fallback",
		WorkArea:    application.Rect{X: 0, Y: 0, Width: 1280, Height: 720},
		ScaleFactor: 1.0,
		Primary:     true,
	}, true
}

func clampRectToWorkArea(rect, work application.Rect) application.Rect {
	if work.Width <= 0 || work.Height <= 0 {
		return rect
	}
	if rect.Width > work.Width {
		rect.Width = work.Width
	}
	if rect.Height > work.Height {
		rect.Height = work.Height
	}
	maxX := work.X + work.Width - rect.Width
	maxY := work.Y + work.Height - rect.Height
	if rect.X < work.X {
		rect.X = work.X
	}
	if rect.Y < work.Y {
		rect.Y = work.Y
	}
	if rect.X > maxX {
		rect.X = maxX
	}
	if rect.Y > maxY {
		rect.Y = maxY
	}
	return rect
}

// ResolveNexV2WindowRect computes the safe, clamped logical window rect for the
// Nex 2.0 avatar from display-relative placement state, handling negative monitor
// origins, mixed DPI, monitor hotplug fallback, and Wayland capability limits.
func ResolveNexV2WindowRect(
	state NexV2PlacementState,
	displays []DisplayDescriptor,
	globalPositioning bool,
) (application.Rect, DisplayDescriptor, bool) {
	preset, ok := ResolveNexV2Preset(state.PresetID)
	if !ok {
		preset = nexV2Presets[1] // standard 160x160
	}
	display, fellBack := selectDisplayForState(state.DisplayID, displays)
	display.ScaleFactor = NormalizeScaleFactor(display.ScaleFactor)

	if !globalPositioning {
		// Wayland and capability-restricted compositors cannot accept global
		// coordinates; keep fixed viewport bounds so animation never clips.
		return application.Rect{
			X:      0,
			Y:      0,
			Width:  preset.Width,
			Height: preset.Height,
		}, display, fellBack
	}

	work := display.WorkArea
	var x, y int
	switch state.Anchor {
	case NexV2AnchorTopLeft:
		x = work.X + nexV2DefaultMarginDIP
		y = work.Y + nexV2DefaultMarginDIP
	case NexV2AnchorTopRight:
		x = work.X + work.Width - preset.Width - nexV2DefaultMarginDIP
		y = work.Y + nexV2DefaultMarginDIP
	case NexV2AnchorBottomLeft:
		x = work.X + nexV2DefaultMarginDIP
		y = work.Y + work.Height - preset.Height - nexV2DefaultMarginDIP
	case NexV2AnchorCustom:
		x = work.X + state.OffsetX
		y = work.Y + state.OffsetY
	default: // NexV2AnchorBottomRight
		x = work.X + work.Width - preset.Width - nexV2DefaultMarginDIP
		y = work.Y + work.Height - preset.Height - nexV2DefaultMarginDIP
	}

	clamped := clampRectToWorkArea(application.Rect{
		X:      x,
		Y:      y,
		Width:  preset.Width,
		Height: preset.Height,
	}, work)
	return clamped, display, fellBack
}

// SwitchNexV2Preset preserves the display-relative bottom-right anchor when the
// user selects a new fixed preset size, keeping the rect clamped to workArea.
func SwitchNexV2Preset(current application.Rect, newPresetID string, display DisplayDescriptor, globalPositioning bool) (application.Rect, NexV2PlacementState, bool) {
	preset, ok := ResolveNexV2Preset(newPresetID)
	if !ok {
		return current, DefaultNexV2PlacementState(), false
	}
	next := application.Rect{
		X:      current.X,
		Y:      current.Y,
		Width:  preset.Width,
		Height: preset.Height,
	}
	if globalPositioning {
		next.X = current.X + current.Width - preset.Width
		next.Y = current.Y + current.Height - preset.Height
		next = clampRectToWorkArea(next, display.WorkArea)
	}
	state := NexV2PlacementState{
		DisplayID:   display.ID,
		Anchor:      NexV2AnchorCustom,
		OffsetX:     next.X - display.WorkArea.X,
		OffsetY:     next.Y - display.WorkArea.Y,
		PresetID:    preset.ID,
		AlwaysOnTop: true,
		ScaleFactor: NormalizeScaleFactor(display.ScaleFactor),
	}
	return next, state, true
}

// NexV2StartupRestorer guarantees initial geometry restoration runs at most once,
// eliminating startup race conditions across WindowShow and ServiceStartup hooks.
type NexV2StartupRestorer struct {
	once sync.Once
}

// Run executes fn at most once across all goroutines.
func (r *NexV2StartupRestorer) Run(fn func()) {
	if fn == nil {
		return
	}
	r.once.Do(fn)
}
