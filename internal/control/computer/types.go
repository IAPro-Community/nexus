package computer

import "time"

type ActionKind string

const (
	ActionClick        ActionKind = "CLICK"
	ActionSetValue     ActionKind = "SET_VALUE"
	ActionTypeText     ActionKind = "TYPE_TEXT"
	ActionPressKey     ActionKind = "PRESS_KEY"
	ActionHotkey       ActionKind = "HOTKEY"
	ActionPasteText    ActionKind = "PASTE_TEXT"
	ActionScroll       ActionKind = "SCROLL"
	ActionDrag         ActionKind = "DRAG"
	ActionFocusWindow  ActionKind = "FOCUS_WINDOW"
	ActionMoveWindow   ActionKind = "MOVE_WINDOW"
	ActionResizeWindow ActionKind = "RESIZE_WINDOW"
)

type Capabilities struct {
	Provider          string `json:"provider"`
	ProviderVersion   string `json:"provider_version,omitempty"`
	ProtocolVersion   int    `json:"protocol_version"`
	Platform          string `json:"platform"`
	ListApps          bool   `json:"list_apps"`
	ListWindows       bool   `json:"list_windows"`
	AccessibilityTree bool   `json:"accessibility_tree"`
	Screenshot        bool   `json:"screenshot"`
	Click             bool   `json:"click"`
	SetValue          bool   `json:"set_value"`
	TypeText          bool   `json:"type_text"`
	PressKey          bool   `json:"press_key"`
	Hotkey            bool   `json:"hotkey"`
	PasteText         bool   `json:"paste_text"`
	Scroll            bool   `json:"scroll"`
	Drag              bool   `json:"drag"`
	WindowManagement  bool   `json:"window_management"`
}

type PermissionStatus struct {
	Available        bool   `json:"available"`
	Accessibility   string `json:"accessibility"`
	ScreenCapture   string `json:"screen_capture"`
	InputMonitoring string `json:"input_monitoring"`
	Reason          string `json:"reason,omitempty"`
}

type AppInfo struct {
	Name     string `json:"name"`
	BundleID string `json:"bundle_id,omitempty"`
	PID      int    `json:"pid"`
}

type WindowInfo struct {
	ID        string `json:"id,omitempty"`
	Index     int    `json:"index,omitempty"`
	Title     string `json:"title"`
	X         int    `json:"x,omitempty"`
	Y         int    `json:"y,omitempty"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Minimized bool   `json:"minimized,omitempty"`
}

type Element struct {
	ID        string   `json:"id"`
	Index     int      `json:"index"`
	Role      string   `json:"role,omitempty"`
	Name      string   `json:"name,omitempty"`
	Value     string   `json:"value,omitempty"`
	Actions   []string `json:"actions,omitempty"`
	Sensitive bool     `json:"sensitive,omitempty"`
	X         int      `json:"x,omitempty"`
	Y         int      `json:"y,omitempty"`
	Width     int      `json:"width,omitempty"`
	Height    int      `json:"height,omitempty"`
}

type SnapshotRequest struct {
	App        string `json:"app"`
	WindowID   string `json:"window_id,omitempty"`
	WindowIndex *int  `json:"window_index,omitempty"`
	Screenshot bool   `json:"screenshot,omitempty"`
	Restore    bool   `json:"restore_window,omitempty"`
}

type Snapshot struct {
	ID          string      `json:"id"`
	App         AppInfo     `json:"app"`
	Window      WindowInfo  `json:"window"`
	Elements    []Element   `json:"elements,omitempty"`
	TreeText    string      `json:"tree_text,omitempty"`
	Screenshot  string      `json:"screenshot_path,omitempty"`
	CapturedAt  time.Time   `json:"captured_at"`
}

type ActionRequest struct {
	Kind        ActionKind `json:"kind"`
	App         string     `json:"app"`
	WindowID    string     `json:"window_id,omitempty"`
	WindowIndex *int       `json:"window_index,omitempty"`
	ElementID   string     `json:"element_id,omitempty"`
	ElementIndex *int      `json:"element_index,omitempty"`
	X           *int       `json:"x,omitempty"`
	Y           *int       `json:"y,omitempty"`
	ToX         *int       `json:"to_x,omitempty"`
	ToY         *int       `json:"to_y,omitempty"`
	Width       *int       `json:"width,omitempty"`
	Height      *int       `json:"height,omitempty"`
	Text        string     `json:"text,omitempty"`
	Key         string     `json:"key,omitempty"`
	Direction   string     `json:"direction,omitempty"`
	Restore     bool       `json:"restore_window,omitempty"`
}

type ActionVerification struct {
	State   string `json:"state"`
	Reason  string `json:"reason,omitempty"`
	Before  string `json:"before_snapshot_id,omitempty"`
	After   string `json:"after_snapshot_id,omitempty"`
}

type ActionResult struct {
	Snapshot     Snapshot           `json:"snapshot"`
	Verification ActionVerification `json:"verification"`
}
