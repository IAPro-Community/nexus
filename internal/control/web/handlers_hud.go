package web

import (
	"net/http"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/flags"
	"github.com/IAPro-Community/nexus/internal/nexus"
)

type hudResponse struct {
	Enabled bool            `json:"enabled"`
	Rollout string          `json:"rollout"`
	State   *nexus.HUDState `json:"state,omitempty"`
}

// handleHUDState exposes one read-only projection over the existing Nexus
// domains. The handler never starts provider probes, schedulers, Agents, or
// Missions. When the rollout flag is disabled it returns a stable envelope so
// clients can hide the experimental surface without treating the state as an
// operational failure. The projection API defaults on for native HUD;
// the Web widget has an independent, user-controlled visibility preference.
func (h *NexusHandler) handleHUDState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !flags.Enabled(flags.HUD) {
		writeJSON(w, http.StatusOK, hudResponse{Enabled: false, Rollout: "experimental"})
		return
	}
	if h == nil || h.hud == nil {
		writeError(w, http.StatusServiceUnavailable, "HUD projection unavailable")
		return
	}
	state, err := h.hud.State(r.Context(), strings.TrimSpace(r.URL.Query().Get("project_id")))
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, hudResponse{
		Enabled: true,
		Rollout: "experimental",
		State:   &state,
	})
}
