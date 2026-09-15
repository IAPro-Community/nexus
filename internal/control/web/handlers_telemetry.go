package web

import (
	"net/http"
	"strconv"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/telemetry"
)

// handleTelemetryStats returns aggregated stats for the specified period (default 7 days).
func (h *NexusHandler) handleTelemetryStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 90 {
			days = parsed
		}
	}

	stats, err := telemetry.ComputeStats(time.Duration(days) * 24 * time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to compute stats: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// handleTelemetryHistory returns recent telemetry events with optional limit and provider filter.
func (h *NexusHandler) handleTelemetryHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	events, err := telemetry.ReadRecentEvents(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read telemetry history: "+err.Error())
		return
	}
	if events == nil {
		events = []telemetry.Event{}
	}

	provider := r.URL.Query().Get("provider")
	if provider != "" {
		filtered := make([]telemetry.Event, 0, len(events))
		for _, ev := range events {
			if ev.ProviderID == provider {
				filtered = append(filtered, ev)
			}
		}
		events = filtered
	}

	writeJSON(w, http.StatusOK, events)
}
