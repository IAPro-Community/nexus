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

func (h *NexusHandler) capacityService() (*nexus.CapacityApplicationService, bool) {
	if h == nil || h.nexus == nil {
		return nil, false
	}
	st := h.nexus.Store()
	if st == nil {
		return nil, false
	}
	return nexus.NewCapacityApplicationService(st), true
}

// handleCapacityStatus GET /api/v1/capacity/status — monitored resources with
// triage, lease ownership and preferences. Unknown stays unknown: no rows
// project as an empty list, never as healthy.
func (h *NexusHandler) handleCapacityStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	svc, ok := h.capacityService()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "capacity store unavailable")
		return
	}
	status, err := svc.Status(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// handleCapacityHistory GET /api/v1/capacity/history?limit=N — durable alert
// events with triage, newest first.
func (h *NexusHandler) handleCapacityHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	svc, ok := h.capacityService()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "capacity store unavailable")
		return
	}
	limit := 50
	if l := strings.TrimSpace(r.URL.Query().Get("limit")); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	events, err := svc.History(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if events == nil {
		events = []nexus.CapacityHistoryEvent{}
	}
	writeJSON(w, http.StatusOK, events)
}

// handleCapacityPreferences GET|PUT /api/v1/capacity/preferences.
func (h *NexusHandler) handleCapacityPreferences(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.capacityService()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "capacity store unavailable")
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
		var prefs store.CapacityPreferences
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&prefs); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		saved, err := svc.UpdatePreferences(r.Context(), prefs)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, saved)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleCapacityAck POST /api/v1/capacity/ack {"fingerprint": "..."}.
func (h *NexusHandler) handleCapacityAck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	svc, ok := h.capacityService()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "capacity store unavailable")
		return
	}
	var body struct {
		Fingerprint string `json:"fingerprint"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	st, err := svc.Ack(r.Context(), body.Fingerprint)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// handleCapacitySnooze POST /api/v1/capacity/snooze
// {"fingerprint": "...", "until": "RFC3339"|""} — empty clears snooze.
func (h *NexusHandler) handleCapacitySnooze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	svc, ok := h.capacityService()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "capacity store unavailable")
		return
	}
	var body struct {
		Fingerprint string `json:"fingerprint"`
		Until       string `json:"until"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	var until time.Time
	if strings.TrimSpace(body.Until) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(body.Until))
		if err != nil {
			writeError(w, http.StatusBadRequest, "until must be RFC3339")
			return
		}
		until = parsed
	}
	st, err := svc.Snooze(r.Context(), body.Fingerprint, until)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, st)
}
