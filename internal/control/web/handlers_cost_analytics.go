package web

import (
	"net/http"
	"strconv"
	"strings"
)

// handleCostAnalytics GET /api/v1/cost/analytics?days=7&project_id=...
// Projects and time ranges are filtered by the existing local CostApplicationService.
// This is read-only: no new collector, tariff table or billing semantics.
func (h *NexusHandler) handleCostAnalytics(w http.ResponseWriter, r *http.Request) {
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
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 365 {
			writeError(w, http.StatusBadRequest, "days must be between 1 and 365")
			return
		}
		days = value
	}
	report, err := svc.Analytics(r.Context(), r.URL.Query().Get("project_id"), days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to read cost analytics")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, report)
}
