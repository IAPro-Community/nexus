package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/handoff"
)

type maestroComposeHTTPBody struct {
	ArtifactID       string `json:"artifact_id"`
	Model            string `json:"model,omitempty"`
	ApprovedModelUse bool   `json:"approved_model_use"`
}

// The body supplies no arbitrary workspace, artifact content, hash, approval
// state or execution target. Those are server-side project-scoped records.
func parseMaestroComposeHTTPBody(reader io.Reader) (maestroComposeHTTPBody, error) {
	var body maestroComposeHTTPBody
	decoder := json.NewDecoder(io.LimitReader(reader, 4096+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		return body, err
	}
	if strings.TrimSpace(body.ArtifactID) == "" || !body.ApprovedModelUse {
		return body, errors.New("finalized artifact and explicit model-use consent required")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return body, errors.New("exactly one JSON request object required")
	}
	return body, nil
}

// POST /api/v1/projects/{id}/maestro/compose-preview
// The route uses the existing session/CSRF middleware. A generated task graph
// is a non-executable proposal until edited/saved/approved by the operator.
func (h *NexusHandler) handleProjectMaestroComposePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required for Maestro composition preview")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if strings.TrimSpace(projectID) == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	body, err := parseMaestroComposeHTTPBody(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Finalized PromptArtifact and explicit model-use consent are required")
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
	if err != nil || session.ProjectID != projectID || session.State != "FINALIZED" {
		writeError(w, http.StatusConflict, "PromptArtifact must belong to a finalized Composer session in this project")
		return
	}

	preview, err := handoff.FetchMaestroComposePreview(r.Context(), project.CanonicalPath, handoff.MaestroComposePreviewRequest{
		ArtifactID: artifact.ID, Version: artifact.Version, Content: artifact.Content,
		SHA256: artifact.Hash, Model: body.Model, ApprovedModelUse: true,
	})
	if err != nil {
		// External provider errors can contain local paths or credentials.
		writeError(w, http.StatusServiceUnavailable, "Maestro composition unavailable or rejected; check external Maestro/Codex and explicit approval")
		return
	}
	writeJSON(w, http.StatusOK, preview)
}
