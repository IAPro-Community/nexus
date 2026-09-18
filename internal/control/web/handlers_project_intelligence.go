package web

import (
	"net/http"
)

// handleProjectIntelligence returns the snapshot/scan read model for the
// current repository identity. It is deliberately separate from the existing
// /api/v1/intelligence provider configuration endpoint.
func (h *NexusHandler) handleProjectIntelligence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	view, err := h.nexus.GetProjectIntelligence(r.Context(), projectIDFromPath(r.URL.Path))
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// handleProjectIntelligenceScan queues a bounded static scan. The scan runs
// outside the request and the returned attempt can be polled through the read
// model endpoint.
func (h *NexusHandler) handleProjectIntelligenceScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	scan, err := h.nexus.RequestProjectIntelligenceScan(r.Context(), projectIDFromPath(r.URL.Path))
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, scan)
}
