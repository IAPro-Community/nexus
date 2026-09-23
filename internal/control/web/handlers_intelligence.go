package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
)

// handleIntelligence GET/PUT /api/v1/intelligence exposes secret-free Intelligence routing configuration.
func (h *NexusHandler) handleIntelligence(w http.ResponseWriter, r *http.Request) {
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	switch r.Method {
	case http.MethodGet:
		status := h.nexus.IntelligenceStatus(r.Context(), projectID)
		writeJSON(w, http.StatusOK, status)
	case http.MethodPut:
		var body coreconfig.IntelligenceConfig
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if err := h.nexus.SetIntelligenceConfig(body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		status := h.nexus.IntelligenceStatus(r.Context(), projectID)
		writeJSON(w, http.StatusOK, status)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleIntelligenceProbe POST /api/v1/intelligence/probe runs a short round-trip
// so Composer can refuse Refinar before a long auto_plan hang.
func (h *NexusHandler) handleIntelligenceProbe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	result, err := h.nexus.ProbeIntelligence(r.Context(), projectID)
	if err != nil {
		if errors.Is(err, intelligence.ErrIntelligenceUnavailable) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"ok":       false,
				"error":    "intelligence_unavailable",
				"detail":   err.Error(),
				"provider": result.Provider,
			})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"ok":       false,
			"error":    "intelligence_probe_failed",
			"detail":   err.Error(),
			"provider": result.Provider,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"provider": result.Provider,
	})
}

// handleIntelligenceMemories GET/POST /api/v1/intelligence/memories
func (h *NexusHandler) handleIntelligenceMemories(w http.ResponseWriter, r *http.Request) {
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	switch r.Method {
	case http.MethodGet:
		rows, err := h.nexus.ListIntelligenceMemories(r.Context(), projectID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if rows == nil {
			rows = []nexus.MergedIntelligenceMemory{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"memories": rows})
	case http.MethodPost:
		var body struct {
			Scope     string `json:"scope"`
			ProjectID string `json:"project_id"`
			Key       string `json:"key"`
			Value     string `json:"value"`
			Source    string `json:"source"`
			Force     bool   `json:"force"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if body.ProjectID == "" {
			body.ProjectID = projectID
		}
		scope := strings.ToUpper(strings.TrimSpace(body.Scope))
		if scope == "" {
			scope = nexus.MemoryScopeProject
		}
		if body.Force || body.Source == nexus.MemorySourceExplicit {
			row, err := h.nexus.UpsertIntelligenceMemory(r.Context(), scope, body.ProjectID, body.Key, body.Value, firstNonEmptyWeb(body.Source, nexus.MemorySourceExplicit))
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, row)
			return
		}
		if err := h.nexus.PromoteHabitMemory(r.Context(), body.ProjectID, body.Key, body.Value, firstNonEmptyWeb(body.Source, nexus.MemorySourceComposer), false); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		rows, err := h.nexus.ListIntelligenceMemories(r.Context(), body.ProjectID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"memories": rows})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *NexusHandler) handleIntelligenceMemoryRecall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body nexus.MemoryRecallRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.ProjectID == "" {
		body.ProjectID = strings.TrimSpace(r.URL.Query().Get("project_id"))
	}
	result, err := h.nexus.RecallIntelligenceMemories(r.Context(), body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *NexusHandler) handleIntelligenceMemoryReflect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body nexus.MemoryRecallRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.ProjectID == "" {
		body.ProjectID = strings.TrimSpace(r.URL.Query().Get("project_id"))
	}
	result, err := h.nexus.ReflectIntelligenceMemories(r.Context(), body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// handleIntelligenceMemoryDelete DELETE /api/v1/intelligence/memories/{scope}/{id}?project_id=
func (h *NexusHandler) handleIntelligenceMemoryDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/intelligence/memories/"), "/")
	parts := strings.Split(rest, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		writeError(w, http.StatusBadRequest, "expected /intelligence/memories/{scope}/{id}")
		return
	}
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if err := h.nexus.DeleteIntelligenceMemory(r.Context(), parts[0], projectID, parts[1]); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "deleted"})
}

func firstNonEmptyWeb(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// handleClarification GET /api/v1/clarifications/{id}
// POST /api/v1/clarifications/{id}/resolve continues the exact persisted analysis.
func (h *NexusHandler) handleClarification(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/clarifications/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "missing clarification id")
		return
	}
	id := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		checkpoint, err := h.nexus.GetClarification(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, checkpoint)
		return
	}
	if len(parts) == 2 && parts[1] == "resolve" && r.Method == http.MethodPost {
		var body struct {
			Answers map[string]string `json:"answers"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		plan, checkpoint, err := h.nexus.ResolveClarificationAndGeneratePlan(r.Context(), id, body.Answers)
		if err != nil {
			var clarificationErr *nexus.ClarificationRequiredError
			if errors.As(err, &clarificationErr) {
				writeJSON(w, http.StatusConflict, map[string]any{
					"error":         "clarification_required",
					"clarification": clarificationErr.Checkpoint,
				})
				return
			}
			if errors.Is(err, intelligence.ErrIntelligenceUnavailable) {
				writeJSON(w, http.StatusServiceUnavailable, map[string]any{
					"error":         "intelligence_unavailable",
					"detail":        err.Error(),
					"clarification": checkpoint,
				})
				return
			}
			var contextErr *nexus.ComposerContextNotReadyError
			if errors.As(err, &contextErr) {
				writeJSON(w, http.StatusConflict, map[string]any{
					"error": "context_not_ready", "detail": err.Error(),
					"readiness": contextErr.Readiness, "clarification": checkpoint,
				})
				return
			}
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"plan": plan, "clarification": checkpoint})
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
