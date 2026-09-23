package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func (h *NexusHandler) costService() (*nexus.CostApplicationService, bool) {
	if h == nil || h.nexus == nil || h.nexus.Store() == nil {
		return nil, false
	}
	return nexus.NewCostApplicationService(h.nexus.Store()), true
}

// handleCostSummary GET /api/v1/cost/summary?days=7&project_id=...
func (h *NexusHandler) handleCostSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	svc, ok := h.costService()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "cost ledger unavailable")
		return
	}
	days := 7
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "days must be a positive integer")
			return
		}
		days = parsed
	}
	summary, err := svc.Summary(r.Context(), r.URL.Query().Get("project_id"), days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// handleCostObservations GET /api/v1/cost/observations with local-only,
// metadata-only attribution. It never returns prompts, terminal output or
// credentials.
func (h *NexusHandler) handleCostObservations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	svc, ok := h.costService()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "cost ledger unavailable")
		return
	}
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = parsed
	}
	days := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "days must be a positive integer")
			return
		}
		days = parsed
	}
	filter := store.CostObservationFilter{
		ProjectID:    r.URL.Query().Get("project_id"),
		MissionRunID: r.URL.Query().Get("mission_run_id"),
		AgentID:      r.URL.Query().Get("agent_id"),
		Provider:     r.URL.Query().Get("provider"),
		Profile:      r.URL.Query().Get("profile"),
		Limit:        limit,
	}
	if days > 0 {
		if days > 365 {
			days = 365
		}
		filter.Since = time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	}
	rows, err := svc.Observations(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rows == nil {
		rows = []store.CostObservation{}
	}
	writeJSON(w, http.StatusOK, rows)
}

func (h *NexusHandler) handleCostPreferences(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.costService()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "cost ledger unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		prefs, err := svc.Preferences(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, prefs)
	case http.MethodPut:
		var prefs store.CostPreferences
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&prefs); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		saved, err := svc.PutPreferences(r.Context(), prefs)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, saved)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
