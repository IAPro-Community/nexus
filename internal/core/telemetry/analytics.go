package telemetry

import (
	"os"
	"strings"
	"sync"
)

// ProductAnalytics is the beta product-measurement boundary, architecturally
// separate from LocalTelemetry (LogEvent/ReadRecentEvents/ComputeStats).
//
// Privacy contract (enforced by construction, not policy):
//   - Only allowlisted event NAMES are recorded; free-form content (prompts,
//     code, files, tokens, terminal output, paths, PII) has no field to flow
//     through — Track takes counts and enum-like attributes only.
//   - Disabled by default; enabled only via explicit opt-in
//     (NEXUS_ANALYTICS=1 or settings opt-in). NEXUS_ANALYTICS=0 always wins.
//   - Offline-resilient: Track never fails the caller and never blocks on
//     network; without a configured sink events are counted in memory only.
//   - The app behaves identically whether analytics is on or off.
//
// Event names worth tracking (callers pick from this list):
// app_started, app_version, os, install_method, provider_detected,
// provider_auth_state_changed, mission_started, mission_completed,
// mission_failed, composer_started, update_checked, update_installed,
// crash_recovery, feature_used.
var (
	analyticsMu     sync.Mutex
	analyticsCounts = map[string]int64{}
	analyticsSink   func(name string, attrs map[string]string)
)

// AnalyticsEventName allowlists product event names.
var AnalyticsEventName = map[string]bool{
	"app_started": true, "app_version": true, "os": true,
	"install_method": true, "provider_detected": true,
	"provider_auth_state_changed": true, "mission_started": true,
	"mission_completed": true, "mission_failed": true,
	"composer_started": true, "update_checked": true,
	"update_installed": true, "crash_recovery": true,
	"feature_used": true,
}

// AnalyticsEnabled reports the opt-in state. Explicit opt-out always wins;
// otherwise explicit opt-in (env) enables. Default is disabled.
func AnalyticsEnabled() bool {
	if v := strings.TrimSpace(os.Getenv("NEXUS_ANALYTICS")); v != "" {
		switch strings.ToLower(v) {
		case "0", "false", "no", "off":
			return false
		case "1", "true", "yes", "on":
			return true
		}
	}
	return false
}

// SetAnalyticsSink installs an optional export sink (error-reporting or
// metrics backends plug in here behind an interface). Nil clears it.
func SetAnalyticsSink(sink func(name string, attrs map[string]string)) {
	analyticsMu.Lock()
	defer analyticsMu.Unlock()
	analyticsSink = sink
}

// Track records one privacy-safe product event. attrs values must be short
// enum-like tokens (os name, provider id, version); Track drops the event
// when disabled or when the name is not allowlisted, and never returns an
// error so callers stay simple and offline-safe.
func Track(name string, attrs map[string]string) {
	if !AnalyticsEnabled() || !AnalyticsEventName[name] {
		return
	}
	safe := make(map[string]string, len(attrs))
	for k, v := range attrs {
		if len(k) > 64 || len(v) > 128 {
			continue
		}
		safe[strings.ToLower(k)] = v
	}
	analyticsMu.Lock()
	analyticsCounts[name]++
	sink := analyticsSink
	analyticsMu.Unlock()
	if sink != nil {
		sink(name, safe)
	}
}

// AnalyticsCounts returns in-memory per-event counters (observability/tests).
func AnalyticsCounts() map[string]int64 {
	analyticsMu.Lock()
	defer analyticsMu.Unlock()
	out := make(map[string]int64, len(analyticsCounts))
	for k, v := range analyticsCounts {
		out[k] = v
	}
	return out
}
