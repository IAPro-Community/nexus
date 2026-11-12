package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// workspaceShares is the process-wide grant registry. Phase 2 keeps the
// registry as the authorization authority (immediate revoke, no decision
// cache) and adds a durable mirror: grants are written through to the store
// and hydrated back into the registry on first use after a restart.
var (
	workspaceShares        = nexus.NewObservationShareRegistry()
	workspaceShareHydrated = struct {
		mu      sync.Mutex
		storeID *store.Store
		done    bool
	}{}
)

// hydrateWorkspaceShares loads persisted grants into the process registry
// exactly once per store instance. A failed load is retried on the next
// request: losing the durable mirror to a transient store error must not
// permanently blind the authorization path.
func hydrateWorkspaceShares(h *NexusHandler) (*store.Store, error) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		return nil, err
	}
	workspaceShareHydrated.mu.Lock()
	defer workspaceShareHydrated.mu.Unlock()
	if workspaceShareHydrated.done && workspaceShareHydrated.storeID == st {
		return st, nil
	}
	grants, err := st.ListObservationShareGrants()
	if err != nil {
		return nil, err
	}
	for _, rec := range grants {
		grant := nexus.ObservationShareGrant{
			ID:        rec.ID,
			ProjectID: rec.ProjectID,
			AgentID:   rec.AgentID,
			Role:      rec.Role,
			CreatedAt: rec.CreatedAt,
			ExpiresAt: rec.ExpiresAt,
			RevokedAt: rec.RevokedAt,
		}
		workspaceShares.Restore(grant)
	}
	workspaceShareHydrated.storeID = st
	workspaceShareHydrated.done = true
	return st, nil
}

// persistShareGrant mirrors one registry grant into the store. Persistence
// failure is surfaced so the caller can retire the un-persisted grant
// instead of silently keeping a grant that would vanish on restart.
func persistShareGrant(st *store.Store, grant *nexus.ObservationShareGrant) error {
	return st.UpsertObservationShareGrant(store.ObservationShareGrantRecord{
		ID:        grant.ID,
		ProjectID: grant.ProjectID,
		AgentID:   grant.AgentID,
		Role:      grant.Role,
		CreatedAt: grant.CreatedAt,
		ExpiresAt: grant.ExpiresAt,
		RevokedAt: grant.RevokedAt,
	})
}

// handleProjectShares dispatches /api/v1/projects/{id}/shares[/{grantID}].
//
// Grant management mutates state, so it keeps the standard authenticated +
// CSRF path. Grants are viewer-only, optionally TTL-bounded (ttl_seconds,
// max 7d) and are consumed through the unauthenticated shared read endpoint
// below. Every grant is persisted before it is returned: a grant the store
// refused is revoked in the registry, never kept half-alive.
func (h *NexusHandler) handleProjectShares(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/projects/"), "/")
	trimmed := strings.TrimSuffix(path, "/shares")
	rest := ""
	if idx := strings.Index(trimmed, "/shares/"); idx >= 0 {
		rest = strings.TrimSpace(trimmed[idx+len("/shares/"):])
		trimmed = trimmed[:idx]
	}
	parts := strings.Split(trimmed, "/")
	projectID := parts[0]
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	st, err := hydrateWorkspaceShares(h)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	if _, err := st.GetProject(projectID); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	switch {
	case rest != "":
		if r.Method != http.MethodDelete {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if !workspaceShares.Revoke(rest, time.Now()) {
			writeError(w, http.StatusNotFound, "grant not found or already revoked")
			return
		}
		if grant, ok := workspaceShares.Get(rest); ok {
			if err := persistShareGrant(st, &grant); err != nil {
				writeError(w, http.StatusServiceUnavailable, "share grant persistence unavailable")
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "revoked", "id": rest})
	case r.Method == http.MethodPost:
		var body struct {
			AgentID    string `json:"agent_id"`
			TTLSeconds int64  `json:"ttl_seconds"`
		}
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
				writeError(w, http.StatusBadRequest, "invalid body")
				return
			}
		}
		var (
			grant *nexus.ObservationShareGrant
		)
		if body.TTLSeconds > 0 {
			g, err := workspaceShares.GrantWithTTL(projectID, body.AgentID, time.Now(), time.Duration(body.TTLSeconds)*time.Second)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			grant = g
		} else {
			grant = workspaceShares.Grant(projectID, body.AgentID, time.Now())
		}
		if err := persistShareGrant(st, grant); err != nil {
			// Retire the un-persisted grant: an authorization that would
			// silently vanish on restart must not be handed out.
			workspaceShares.Revoke(grant.ID, time.Now())
			writeError(w, http.StatusServiceUnavailable, "share grant persistence unavailable")
			return
		}
		writeJSON(w, http.StatusCreated, grant)
	case r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"grants": workspaceShares.List(projectID)})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleSharedObservations GET /api/v1/shared/observations?grant=<id>&agent_id=<id>
//
// Viewer-only read path for shared observation grants. It serves cost
// observations scoped to the granted project (and agent when the grant is
// narrowed). No session is required — the grant IS the authorization — and
// every request is checked against the current registry state, so a revoke
// or expiry denies the very next call.
func (h *NexusHandler) handleSharedObservations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, err := hydrateWorkspaceShares(h); err != nil {
		writeError(w, http.StatusServiceUnavailable, "share registry unavailable")
		return
	}
	grantID := strings.TrimSpace(r.URL.Query().Get("grant"))
	if grantID == "" {
		writeError(w, http.StatusBadRequest, "missing grant")
		return
	}
	grant, ok := workspaceShares.Get(grantID)
	if !ok {
		writeError(w, http.StatusForbidden, "unknown grant")
		return
	}
	reason, allowed := workspaceShares.Authorize(grantID, grant.ProjectID, r.URL.Query().Get("agent_id"), nexus.ShareActionRead)
	if !allowed {
		writeError(w, http.StatusForbidden, reason)
		return
	}
	svc, ok := h.costService()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "cost ledger unavailable")
		return
	}
	observations, err := svc.Observations(r.Context(), store.CostObservationFilter{
		ProjectID: grant.ProjectID,
		AgentID:   strings.TrimSpace(r.URL.Query().Get("agent_id")),
		Limit:     200,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "observations unavailable")
		return
	}
	// Viewer payload: observations only. No preferences, no accounts,
	// no provider identities.
	writeJSON(w, http.StatusOK, map[string]any{
		"project_id":   grant.ProjectID,
		"role":         grant.Role,
		"observations": observations,
	})
}
