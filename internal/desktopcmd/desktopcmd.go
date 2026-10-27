// Package desktopcmd carries UI actions requested by another surface (for
// example the HUD) to the Desktop window that owns the webview.
//
// The command travels through the Core HTTP API because the HUD may run as a
// separate process. Only a Core embedded in the Desktop registers a handler;
// when no Desktop window exists the request fails instead of being queued, so
// the caller can explain that the app is closed.
package desktopcmd

import (
	"strings"
	"sync"
)

// ActionNewAgent opens the new agent dialog in the Desktop window.
const ActionNewAgent = "new-agent"

// EventName is the Wails event emitted to the Desktop webview after a command
// is accepted.
const EventName = "nexus:desktop-command"

var (
	mu      sync.RWMutex
	handler func(action string) bool
)

// SetHandler installs the Desktop callback. Passing nil unregisters it.
func SetHandler(fn func(action string) bool) {
	mu.Lock()
	handler = fn
	mu.Unlock()
}

// Dispatch delivers an action to the Desktop window. It reports whether a
// handler was registered and accepted the action.
func Dispatch(action string) bool {
	action = strings.TrimSpace(action)
	if action == "" {
		return false
	}
	mu.RLock()
	fn := handler
	mu.RUnlock()
	if fn == nil {
		return false
	}
	return fn(action)
}
