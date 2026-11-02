package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/handoff"
)

// handleProjectMaestroRoutePreview is an authenticated, CSRF-protected read-only
// preview. It never submits a Run or reads a caller-selected artifact's content
// from another project. The trusted PromptArtifact lives in the Nexus store.
func (h *NexusHandler) handleProjectMaestroRoutePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required for Maestro routing preview")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if strings.TrimSpace(projectID) == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}

	var body struct {
		ArtifactID  string                          `json:"artifact_id"`
		Proposal    json.RawMessage                 `json:"proposal"`
		Assignments []handoff.MaestroWorkAssignment `json:"assignments"`
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, 128*1024+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || body.ArtifactID == "" ||
		len(body.Proposal) == 0 || len(body.Assignments) == 0 {
		writeError(w, http.StatusBadRequest, "artifact_id, proposal and assignments are required")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "exactly one request object is required")
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
	artifact, err := st.GetPromptArtifact(body.ArtifactID)
	if err != nil {
		writeError(w, http.StatusNotFound, "PromptArtifact not found")
		return
	}
	session, err := st.GetComposerSession(artifact.SessionID)
	if err != nil || session.ProjectID != projectID {
		writeError(w, http.StatusNotFound, "PromptArtifact does not belong to project")
		return
	}
	// Never accept content/hash/version from the browser. The project-bound
	// persisted artifact is the only source of those immutable fields.
	preview, err := handoff.FetchMaestroRoutePreview(r.Context(), project.CanonicalPath, handoff.MaestroRoutePreviewRequest{
		ArtifactID: artifact.ID,
		Version: artifact.Version,
		Content: artifact.Content,
		SHA256: artifact.Hash,
		Proposal: body.Proposal,
		Assignments: body.Assignments,
	})
	if err != nil {
		// Do not leak external CLI stderr, local paths or model diagnostics.
		writeError(w, http.StatusServiceUnavailable, "Maestro routing preflight unavailable or rejected; verify CLI/capabilities and task assignments")
		return
	}
	writeJSON(w, http.StatusOK, preview)
}
