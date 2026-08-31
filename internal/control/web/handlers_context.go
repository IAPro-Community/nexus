package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/kivervinicius/ai-cli/internal/nexus"
)

// handleProjectContextReadiness observes repository metadata and stores only
// the readiness contract. It never performs semantic hydration itself.
func (h *NexusHandler) handleProjectContextReadiness(w http.ResponseWriter, r *http.Request, route string) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	project, err := st.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if route == "context-hydrate" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		record, err := st.SaveContextReadiness(projectID, nexus.ContextHydrating, nexus.ContextFingerprint{ProjectID: projectID}, "Maestro hydration requested")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, record)
		return
	}
	status := nexus.NewMaestroClient().Status()
	version := nexus.MaestroVersion
	maestroState := "UNAVAILABLE"
	if status.Available {
		maestroState = "AVAILABLE"
		if status.Capabilities != nil && status.Capabilities.Version != "" {
			version = status.Capabilities.Version
		}
	}
	current, err := nexus.ObserveContextFingerprint(context.Background(), projectID, project.CanonicalPath, version, maestroState)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	previous, err := st.GetContextReadiness(projectID)
	if route == "context-refresh" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var body struct {
			Hydrated bool `json:"hydrated"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if previous == nil {
			writeJSON(w, http.StatusOK, nexus.EvaluateContextReadiness(nil, &current, false, "context has not been hydrated"))
			return
		}
		var priorFP nexus.ContextFingerprint
		_ = json.Unmarshal([]byte(previous.FingerprintJSON), &priorFP)
		readiness := nexus.EvaluateContextReadiness(&priorFP, &current, previous.State == nexus.ContextHydrating && !body.Hydrated, "")
		if body.Hydrated && readiness.State != nexus.ContextStale {
			readiness.State = nexus.ContextReady
			readiness.Reason = ""
			readiness.Fingerprint = &current
		}
		record, err := st.SaveContextReadiness(projectID, readiness.State, current, readiness.Reason)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, record)
		return
	}
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusOK, nexus.EvaluateContextReadiness(nil, &current, false, "context has not been hydrated"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var priorFP nexus.ContextFingerprint
	if json.Unmarshal([]byte(previous.FingerprintJSON), &priorFP) != nil {
		writeJSON(w, http.StatusOK, nexus.EvaluateContextReadiness(nil, &current, false, "invalid persisted context fingerprint"))
		return
	}
	readiness := nexus.EvaluateContextReadiness(&priorFP, &current, previous.State == nexus.ContextHydrating, previous.Reason)
	writeJSON(w, http.StatusOK, readiness)
}
