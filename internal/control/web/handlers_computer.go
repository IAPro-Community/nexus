package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/computer"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const computerUseRequestBodyLimit = 1 << 20

func (h *NexusHandler) handleProjectComputerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	if _, err := st.GetProject(projectID); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	status, err := h.nexus.ComputerUseStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *NexusHandler) handleProjectComputerSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	if err := h.requireProject(projectID); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var body struct {
		Contract runner.AutonomyContract  `json:"contract"`
		Request  computer.SnapshotRequest `json:"request"`
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, computerUseRequestBodyLimit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid computer snapshot body")
		return
	}
	body.Request.App = strings.TrimSpace(body.Request.App)
	snapshot, err := h.nexus.ComputerSnapshot(r.Context(), body.Contract, body.Request)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (h *NexusHandler) handleProjectComputerAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	if err := h.requireProject(projectID); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var body struct {
		Contract runner.AutonomyContract `json:"contract"`
		Request  computer.ActionRequest  `json:"request"`
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, computerUseRequestBodyLimit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid computer action body")
		return
	}
	body.Request.App = strings.TrimSpace(body.Request.App)
	result, err := h.nexus.ComputerAct(r.Context(), body.Contract, body.Request)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *NexusHandler) requireProject(projectID string) error {
	st, err := h.nexus.OpenProject()
	if err != nil {
		return err
	}
	_, err = st.GetProject(strings.TrimSpace(projectID))
	return err
}
