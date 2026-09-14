package web

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

// handleConductorPolicy handles GET/PUT /api/v1/conductor/policy
func (h *NexusHandler) handleConductorPolicy(w http.ResponseWriter, r *http.Request) {
	n := h.nexus
	if n == nil {
		writeError(w, http.StatusServiceUnavailable, "nexus unavailable")
		return
	}
	st, err := n.OpenProject()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet:
		policy, err := nexus.GetConductorPolicy(st)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, policy)

	case http.MethodPut:
		var policy nexus.ConductorPolicy
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&policy); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if err := nexus.PutConductorPolicy(st, &policy); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Return the persisted policy.
		saved, err := nexus.GetConductorPolicy(st)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, saved)

	default:
		w.Header().Set("Allow", "GET, PUT")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleConductorDispatch handles POST /api/v1/conductor/dispatch
func (h *NexusHandler) handleConductorDispatch(w http.ResponseWriter, r *http.Request) {
	n := h.nexus
	if n == nil {
		writeError(w, http.StatusServiceUnavailable, "nexus unavailable")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req nexus.DispatchRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	result, err := n.Dispatch(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
