package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// handleProjectsList GET|POST /api/v1/projects
func (h *NexusHandler) handleProjectsList(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := h.projects.List(r.Context())
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var body struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil || strings.TrimSpace(body.Path) == "" {
			writeError(w, http.StatusBadRequest, "path is required")
			return
		}
		proj, err := h.projects.Create(r.Context(), body.Name, body.Path)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, proj)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleProjectSummaries GET /api/v1/projects/summaries
// Returns lightweight per-project agent counts for the project rail outline.
func (h *NexusHandler) handleProjectSummaries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	list, err := h.agents.ListSummaries(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleProjectDetail GET/PATCH/DELETE /api/v1/projects/{id}
func (h *NexusHandler) handleProjectDetail(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/projects/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	id := parts[0]

	switch r.Method {
	case http.MethodGet:
		proj, record, err := h.projects.Get(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"project": proj, "layout": record.Layout, "revision": record.Revision, "record": record})

	case http.MethodPatch:
		var body struct {
			Name             *string `json:"name"`
			MaestroMode      *string `json:"maestro_mode"`
			DefaultIsolation *string `json:"default_isolation"`
			DefaultBranch    *string `json:"default_branch"`
			ResourcePolicy   *string `json:"resource_policy"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		proj, err := h.projects.Update(r.Context(), id, nexus.ProjectPatch{
			Name: body.Name, MaestroMode: body.MaestroMode, DefaultIsolation: body.DefaultIsolation,
			DefaultBranch: body.DefaultBranch, ResourcePolicy: body.ResourcePolicy,
		})
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, proj)

	case http.MethodDelete:
		if err := h.projects.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleProjectEvents GET /api/v1/projects/{id}/events returns recorded activity events.
func (h *NexusHandler) handleProjectEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := projectIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}

	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if parsed, err := strconv.Atoi(lStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	agentID := r.URL.Query().Get("agent_id")

	eventsList, err := h.projects.Events(r.Context(), id, agentID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for i := range eventsList {
		eventsList[i].Summary = security.Redact(eventsList[i].Summary)
	}
	writeJSON(w, http.StatusOK, eventsList)
}

// handleProjectContext GET /api/v1/projects/{id}/context returns the persisted
// five-state readiness contract with live source-drift observation.
func (h *NexusHandler) handleProjectContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	readiness, err := h.nexus.ObserveContextReadiness(projectIDFromPath(r.URL.Path))
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, readiness)
}

// handleProjectContextPrepare POST /api/v1/projects/{id}/context/prepare checks
// durable context artifacts; semantic hydration itself remains owned by Maestro.
func (h *NexusHandler) handleProjectContextPrepare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		CreateContext bool `json:"create_context"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body)
	var readiness *nexus.ContextReadiness
	var err error
	if body.CreateContext {
		readiness, err = h.nexus.PrepareContextWithBootstrap(projectIDFromPath(r.URL.Path))
	} else {
		readiness, err = h.nexus.PrepareContext(projectIDFromPath(r.URL.Path))
	}
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, readiness)
}

// handleProjectShell POST /api/v1/projects/{id}/shell launches a normal
// supervised project shell without creating an Agent.
func (h *NexusHandler) handleProjectShell(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	sess, err := h.nexus.StartProjectShell(r.Context(), projectID)
	if err != nil {
		// A shell launch failure is an unavailable local runtime, not a
		// resource conflict. Preserve the concrete error for the UI while
		// allowing clients to distinguish it from optimistic-concurrency 409s.
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"runtime": sess})
}

// handleProjectLayout GET/PUT /api/v1/projects/{id}/layout
func (h *NexusHandler) handleProjectLayout(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/projects/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	id := parts[0]
	switch r.Method {
	case http.MethodGet:
		rec, err := h.projects.Layout(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"layout":   rec.Layout,
			"revision": rec.Revision,
			"record":   rec,
		})
	case http.MethodPut:
		var body struct {
			Layout   string `json:"layout"`
			Revision *int64 `json:"revision"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		var expectedRev int64
		if body.Revision != nil {
			expectedRev = *body.Revision
		}
		newRec, err := h.projects.SaveLayout(r.Context(), id, body.Layout, expectedRev)
		if errors.Is(err, store.ErrRevisionConflict) {
			cur, _ := h.projects.Layout(r.Context(), id)
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":            "layout revision conflict",
				"code":             "REVISION_CONFLICT",
				"current_revision": cur.Revision,
			})
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "saved",
			"revision": newRec.Revision,
			"layout":   newRec.Layout,
		})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
