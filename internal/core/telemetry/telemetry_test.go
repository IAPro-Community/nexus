package telemetry

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestTelemetryLoggingAndStats(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("AI_CLI_STATE_DIR", stateDir)

	now := time.Now()

	// Log several events
	_ = LogEvent(Event{
		Timestamp:  now.Add(-2 * time.Hour),
		Type:       EventSessionStarted,
		ProviderID: "codex",
		ProfileID:  "personal",
		Workspace:  "/projects/omega",
	})
	_ = LogEvent(Event{
		Timestamp:   now.Add(-1 * time.Hour),
		Type:        EventRateLimitDetected,
		ProviderID:  "codex",
		ProfileID:   "work",
		FailureKind: model.FailureRateLimit,
	})
	_ = LogEvent(Event{
		Timestamp:       now.Add(-1 * time.Hour),
		Type:            EventFallbackTriggered,
		ProviderID:      "codex",
		ProfileID:       "work",
		FallbackProfile: "personal",
	})

	events, err := ReadRecentEvents(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	stats, err := ComputeStats(24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalSessions != 1 {
		t.Fatalf("expected 1 session, got %d", stats.TotalSessions)
	}
	if stats.TotalFallbacks != 1 {
		t.Fatalf("expected 1 fallback, got %d", stats.TotalFallbacks)
	}
	if stats.TotalRateLimits != 1 {
		t.Fatalf("expected 1 rate limit, got %d", stats.TotalRateLimits)
	}
}
