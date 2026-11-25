package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

// handleNexMemory manages @personal Nex assistant preferences/facts
// separately from workspace/project intelligence and Maestro semantic memory.
func (h *NexusHandler) handleNexMemory(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.nexMemory == nil {
		writeError(w, http.StatusServiceUnavailable, "Nex personal memory unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"entries": h.nexMemory.List(),
		})
	case http.MethodPost:
		var body struct {
			Key        string `json:"key"`
			Value      string `json:"value"`
			Provenance string `json:"provenance"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid memory payload")
			return
		}
		entry, err := h.nexMemory.Upsert(body.Key, body.Value, body.Provenance, time.Now().UTC())
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"entry": entry})
	case http.MethodDelete:
		if strings.TrimSpace(r.URL.Query().Get("all")) == "1" {
			if err := h.nexMemory.Clear(); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
			return
		}
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			writeError(w, http.StatusBadRequest, "id or all=1 is required")
			return
		}
		deleted := h.nexMemory.Delete(id)
		writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

var _ = nexus.NexPersonalMemoryEntry{}
