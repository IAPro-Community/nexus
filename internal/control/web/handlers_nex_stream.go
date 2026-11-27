package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/flags"
	"github.com/IAPro-Community/nexus/internal/nexus"
)

type nexEventsResponse struct {
	Revision            uint64                   `json:"revision"`
	ReplayTruncated     bool                     `json:"replay_truncated"`
	Events              []nexus.NexEventEnvelope `json:"events"`
	ModelWatchFreshness string                   `json:"model_watch_freshness"`
}

// handleNexEvents exposes a read-only, cursor-indexed event stream derived from
// canonical HUD projection and ModelCatalogWatchService.
func (h *NexusHandler) handleNexEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var cursor uint64
	if raw := strings.TrimSpace(r.URL.Query().Get("since")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid since cursor")
			return
		}
		cursor = parsed
	}
	if h == nil || h.nexEvents == nil || h.modelWatch == nil {
		writeError(w, http.StatusServiceUnavailable, "Nex event stream unavailable")
		return
	}

	now := time.Now().UTC()
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if flags.Enabled(flags.HUD) && h.hud != nil {
		if state, err := h.hud.State(r.Context(), projectID); err == nil {
			state.RuntimeAttention = nexus.BuildHUDRuntimeAttention(registry.DefaultRegistry().List(), projectID)
			changes := h.modelWatch.ObserveHUDState(state, now)
			h.nexEvents.IngestHUDState(state, changes, now)
		}
	}

	page := h.nexEvents.Since(cursor)
	snap := h.modelWatch.Snapshot(now)
	recordNexHubFromRequest(r)
	writeJSON(w, http.StatusOK, nexEventsResponse{
		Revision:            page.Revision,
		ReplayTruncated:     page.ReplayTruncated,
		Events:              page.Events,
		ModelWatchFreshness: snap.Freshness,
	})
}

// handleNexModelWatch exposes the canonical read-only model catalog watch
// snapshot without launching any external provider probes.
func (h *NexusHandler) handleNexModelWatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h == nil || h.modelWatch == nil {
		writeError(w, http.StatusServiceUnavailable, "model catalog watch unavailable")
		return
	}
	now := time.Now().UTC()
	if flags.Enabled(flags.HUD) && h.hud != nil {
		if state, err := h.hud.State(r.Context(), strings.TrimSpace(r.URL.Query().Get("project_id"))); err == nil {
			changes := h.modelWatch.ObserveHUDState(state, now)
			if h.nexEvents != nil && len(changes) > 0 {
				h.nexEvents.IngestHUDState(state, changes, now)
			}
		}
	}
	writeJSON(w, http.StatusOK, h.modelWatch.Snapshot(now))
}
