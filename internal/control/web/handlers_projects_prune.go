package web

import (
	"context"
	"net/http"
	"time"
)

// handleProjectPruneCandidates GET /api/v1/projects/prune-candidates
// Lists projects without usage evidence (no agents, no saved layout, no
// events, never opened). Read-only: the caller decides whether to prune.
func (h *NexusHandler) handleProjectPruneCandidates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	candidates, err := h.projects.PruneCandidates(ctx)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"candidates": candidates})
}
