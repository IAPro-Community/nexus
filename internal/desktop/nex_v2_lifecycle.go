package desktop

import (
	"sync"
	"time"
)

const (
	NexV2LifecycleStandalone = "standalone"
	NexV2LifecycleLinked     = "linked"

	nexV2MaxRendererRestarts = 3
	nexV2BaseBackoff         = 500 * time.Millisecond
	nexV2MaxBackoff          = 2 * time.Second
	nexV2HealthyResetWindow  = 5 * time.Minute
)

// NexV2LifecycleStatus describes the ownership relationship between the desktop
// shell and the shared Go Core.
type NexV2LifecycleStatus struct {
	Mode               string `json:"mode"`
	OwnsEmbeddedCore   bool   `json:"ownsEmbeddedCore"`
	MainCloseKeepsCore bool   `json:"mainCloseKeepsCore"`
	RendererCrashSafe  bool   `json:"rendererCrashSafe"`
}

// ResolveNexV2Lifecycle documents the explicit standalone vs linked lifecycle.
// Closing the main window never shuts down a standalone instance or an attached
// Web Core, and renderer restarts never terminate user Agents.
func ResolveNexV2Lifecycle(linked, ownsEmbeddedCore bool) NexV2LifecycleStatus {
	mode := NexV2LifecycleStandalone
	if linked && !ownsEmbeddedCore {
		mode = NexV2LifecycleLinked
	}
	return NexV2LifecycleStatus{
		Mode:               mode,
		OwnsEmbeddedCore:   ownsEmbeddedCore,
		MainCloseKeepsCore: mode == NexV2LifecycleStandalone || ownsEmbeddedCore,
		RendererCrashSafe:  true,
	}
}

// ShouldTerminateCoreOnRendererError enforces that WebView/renderer crashes
// recover presentation only without stopping the Core or user Agents.
func ShouldTerminateCoreOnRendererError() bool {
	return false
}

// RendererRecoverySupervisor enforces bounded renderer restarts with
// exponential backoff and falls back to a static icon once exhausted.
type RendererRecoverySupervisor struct {
	mu           sync.Mutex
	now          func() time.Time
	restarts     int
	lastHealthy  time.Time
	staticBackup bool
}

// NewRendererRecoverySupervisor constructs a supervisor using the provided clock
// (or time.Now when nil).
func NewRendererRecoverySupervisor(now func() time.Time) *RendererRecoverySupervisor {
	if now == nil {
		now = time.Now
	}
	return &RendererRecoverySupervisor{
		now:         now,
		lastHealthy: now(),
	}
}

// RecordFailure records a renderer crash/unresponsive event and returns whether
// presentation restart should be attempted along with its bounded delay.
func (s *RendererRecoverySupervisor) RecordFailure() (bool, time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.restarts >= nexV2MaxRendererRestarts {
		s.staticBackup = true
		return false, 0
	}

	delay := nexV2BaseBackoff << s.restarts
	if delay > nexV2MaxBackoff {
		delay = nexV2MaxBackoff
	}
	s.restarts++
	return true, delay
}

// RecordHealthy marks a healthy renderer heartbeat and resets the failure budget
// once the healthy window has elapsed.
func (s *RendererRecoverySupervisor) RecordHealthy() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if s.restarts > 0 && now.Sub(s.lastHealthy) >= nexV2HealthyResetWindow {
		s.restarts = 0
		s.staticBackup = false
	}
	s.lastHealthy = now
}

// StaticFallbackActive reports whether the bounded restart budget was exhausted
// and the shell should stay on the static tray/icon fallback.
func (s *RendererRecoverySupervisor) StaticFallbackActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.staticBackup
}

// RestartCount returns the number of renderer restarts attempted in the current
// window.
func (s *RendererRecoverySupervisor) RestartCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.restarts
}
