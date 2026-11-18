package web

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/handoff"
)

// GET /api/v1/projects/{id}/maestro/submissions
// The project router enforces authentication and Origin validation before this
// read-only handler is invoked. Caller paths and execution ownership are never
// accepted: the trusted project store chooses the external Maestro workspace.
func (h *NexusHandler) handleProjectMaestroSubmissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required for Maestro submissions")
		return
	}
	for key := range r.URL.Query() {
		if key != "cursor" && key != "limit" && key != "offset" {
			writeError(w, http.StatusBadRequest, "unexpected Maestro polling parameter")
			return
		}
	}
	if len(r.URL.Query()["cursor"]) > 1 || len(r.URL.Query()["limit"]) > 1 ||
		len(r.URL.Query()["offset"]) > 1 {
		writeError(w, http.StatusBadRequest, "ambiguous Maestro polling parameter")
		return
	}
	cursor := r.URL.Query().Get("cursor")
	limit, offset := 50, 0
	var err error
	if value := r.URL.Query().Get("limit"); value != "" {
		limit, err = strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 100 {
			writeError(w, http.StatusBadRequest, "invalid Maestro polling limit")
			return
		}
	}
	if value := r.URL.Query().Get("offset"); value != "" {
		offset, err = strconv.Atoi(value)
		if err != nil || offset < 0 || offset > 1000000 {
			writeError(w, http.StatusBadRequest, "invalid Maestro polling offset")
			return
		}
	}
	if len(cursor) > 128 || (cursor != "" && (len(cursor) != 67 || !strings.HasPrefix(cursor, "s1:"))) {
		writeError(w, http.StatusBadRequest, "invalid Maestro polling cursor")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "Nexus project store unavailable")
		return
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	result, err := handoff.FetchMaestroSubmissionSnapshot(r.Context(), project.CanonicalPath,
		cursor, limit, offset)
	if err != nil {
		// Do not leak executable paths, CLI stderr or provider metadata.
		writeError(w, http.StatusServiceUnavailable, "Maestro submission polling unavailable or incompatible")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/projects/{id}/maestro/submissions/import-usage imports
// measured usage evidence from terminal Maestro submissions into the project
// cost ledger. The body is empty (or {}): the caller chooses nothing — the
// trusted project store picks the workspace and the Maestro run store owns
// the evidence. Errors never leak CLI paths or bridge diagnostics.
func (h *NexusHandler) handleProjectMaestroSubmissionUsageImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required for Maestro usage import")
		return
	}
	if r.Body != nil {
		body, err := io.ReadAll(io.LimitReader(r.Body, 4097))
		if err != nil || len(body) > 4096 {
			writeError(w, http.StatusBadRequest, "invalid Maestro usage import body")
			return
		}
		if trimmed := strings.TrimSpace(string(body)); trimmed != "" && trimmed != "{}" {
			writeError(w, http.StatusBadRequest, "Maestro usage import takes no parameters")
			return
		}
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "Nexus project store unavailable")
		return
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	result, err := h.nexus.ImportMaestroUsageObservations(r.Context(), &project)
	if err != nil {
		// Do not leak executable paths, CLI stderr or bridge diagnostics.
		writeError(w, http.StatusServiceUnavailable, "Maestro usage import unavailable or incompatible")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}
