package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func (h *NexusHandler) handlePlanRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	planID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/plans/"), "/restore")
	var body struct {
		Revision int `json:"revision"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body) != nil || body.Revision <= 0 {
		writeError(w, http.StatusBadRequest, "revision is required")
		return
	}
	plan, rev, err := h.nexus.RestoreWorkPlanRevision(r.Context(), planID, body.Revision)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plan": plan, "revision": rev})
}

func (h *NexusHandler) handlePlanDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	planID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/plans/"), "/diff")
	from, err1 := strconv.Atoi(r.URL.Query().Get("from"))
	to, err2 := strconv.Atoi(r.URL.Query().Get("to"))
	if err1 != nil || err2 != nil || from <= 0 || to <= 0 {
		writeError(w, http.StatusBadRequest, "from/to revisions are required")
		return
	}
	diff, err := h.nexus.ComparePlanRevisions(r.Context(), planID, from, to)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, diff)
}
