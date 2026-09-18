package web

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/telemetry"
)

func TestTelemetryEndpoints(t *testing.T) {
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	client, srv, _ := csrfClient(t)

	// Ensure there is at least one test event logged so stats and history have data
	_ = telemetry.LogEvent(telemetry.Event{
		Timestamp:  time.Now(),
		Type:       telemetry.EventSessionStarted,
		ProviderID: "codex",
		ProfileID:  "test-profile",
		Workspace:  "/test/workspace",
		DurationMs: 1200,
	})

	t.Run("GET /api/v1/telemetry/stats", func(t *testing.T) {
		resp, err := client.Get(srv.URL() + "/api/v1/telemetry/stats?days=7")
		if err != nil {
			t.Fatalf("get stats: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var stats telemetry.StatsSummary
		if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
			t.Fatalf("decode stats: %v", err)
		}

		if stats.PeriodDays != 7 {
			t.Errorf("expected PeriodDays 7, got %d", stats.PeriodDays)
		}
		if stats.TotalSessions < 1 {
			t.Errorf("expected at least 1 total session, got %d", stats.TotalSessions)
		}
	})

	t.Run("GET /api/v1/telemetry/history", func(t *testing.T) {
		resp, err := client.Get(srv.URL() + "/api/v1/telemetry/history?limit=10")
		if err != nil {
			t.Fatalf("get history: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var events []telemetry.Event
		if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
			t.Fatalf("decode history: %v", err)
		}

		if len(events) == 0 {
			t.Error("expected at least 1 event in history")
		}
	})

	t.Run("GET /api/v1/telemetry/history with provider filter", func(t *testing.T) {
		resp, err := client.Get(srv.URL() + "/api/v1/telemetry/history?provider=codex")
		if err != nil {
			t.Fatalf("get history with filter: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var events []telemetry.Event
		if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
			t.Fatalf("decode history: %v", err)
		}

		for _, ev := range events {
			if ev.ProviderID != "codex" {
				t.Errorf("expected provider 'codex', got %s", ev.ProviderID)
			}
		}
	})
}
