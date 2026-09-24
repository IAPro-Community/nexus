package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func (h *NexusHandler) handleAgentProjections(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/agent-projections/"), "/")
	switch path {
	case "plan":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		runtimeID, ok := decodeProjectionRuntimeID(w, r)
		if !ok {
			return
		}
		plan, err := h.nexus.PlanRuntimeAgentProjection(r.Context(), runtimeID)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, plan)
		return

	case "apply":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		runtimeID, ok := decodeProjectionRuntimeID(w, r)
		if !ok {
			return
		}
		applied, err := h.nexus.ApplyRuntimeAgentProjection(r.Context(), runtimeID, "web")
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, applied)
		return

	case "discovery":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		runtimeID, ok := decodeProjectionRuntimeID(w, r)
		if !ok {
			return
		}
		discovery, err := h.nexus.ProbeRuntimeAgentProjectionDiscovery(r.Context(), runtimeID)
		if err != nil {
			if discovery == nil {
				writeError(w, http.StatusConflict, err.Error())
			} else {
				writeJSON(w, http.StatusConflict, discovery)
			}
			return
		}
		writeJSON(w, http.StatusOK, discovery)
		return

	case "activation":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		runtimeID, confirm, ok := decodeProjectionActivationRequest(w, r)
		if !ok {
			return
		}
		activation, err := h.nexus.ProbeRuntimeAgentProjectionActivation(r.Context(), runtimeID, confirm)
		if err != nil {
			if activation == nil {
				writeError(w, http.StatusConflict, err.Error())
			} else {
				writeJSON(w, http.StatusConflict, activation)
			}
			return
		}
		writeJSON(w, http.StatusOK, activation)
		return
	}

	parts := strings.Split(path, "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] == "verify" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		result, err := h.nexus.VerifyAgentProjection(r.Context(), parts[0])
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	if len(parts) == 1 && parts[0] != "" {
		if r.Method != http.MethodDelete {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if err := h.nexus.RemoveAgentProjection(r.Context(), parts[0]); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
		return
	}

	writeError(w, http.StatusNotFound, "invalid agent projection path")
}

func decodeProjectionRuntimeID(w http.ResponseWriter, r *http.Request) (string, bool) {
	var body struct {
		RuntimeID string `json:"runtime_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "valid JSON body is required")
		return "", false
	}
	body.RuntimeID = strings.TrimSpace(body.RuntimeID)
	if body.RuntimeID == "" {
		writeError(w, http.StatusBadRequest, "runtime_id is required")
		return "", false
	}
	return body.RuntimeID, true
}

func decodeProjectionActivationRequest(w http.ResponseWriter, r *http.Request) (string, bool, bool) {
	var body struct {
		RuntimeID                string `json:"runtime_id"`
		ConfirmExternalExecution bool   `json:"confirm_external_execution"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "valid JSON body is required")
		return "", false, false
	}
	body.RuntimeID = strings.TrimSpace(body.RuntimeID)
	if body.RuntimeID == "" {
		writeError(w, http.StatusBadRequest, "runtime_id is required")
		return "", false, false
	}
	return body.RuntimeID, body.ConfirmExternalExecution, true
}
