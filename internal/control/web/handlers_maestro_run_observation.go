package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/handoff"
)

// Read-only proxy: only the external Maestro owns delegated runs, and the
// project-scoped Bridge checks Run/Task ownership before returning lifecycle.
func (h *NexusHandler) handleProjectMaestroRunObservation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required for Maestro run observation")
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/projects/"), "/"), "/")
	if len(parts) != 5 || parts[0] == "" || parts[1] != "maestro" ||
		parts[2] != "runs" || parts[3] == "" || parts[4] != "observe" {
		writeError(w, http.StatusNotFound, "Maestro run observation not found")
		return
	}
	for key := range r.URL.Query() {
		if key != "cursor" && key != "limit" {
			writeError(w, http.StatusBadRequest, "unsupported Maestro run observation query")
			return
		}
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}
		limit = parsed
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "Nexus project store unavailable")
		return
	}
	project, err := st.GetProject(parts[0])
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	observation, err := handoff.FetchMaestroRunObservation(
		r.Context(), project.CanonicalPath, parts[3], r.URL.Query().Get("cursor"), limit,
	)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "Maestro run observation unavailable or outside project scope")
		return
	}
	writeJSON(w, http.StatusOK, observation)
}
