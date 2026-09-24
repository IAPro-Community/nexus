package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// handleAgentModes GET /api/v1/agent-modes returns the canonical behavioral
// presets. Icons/labels stay presentation concerns in the Web client.
func (h *NexusHandler) handleAgentModes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, nexus.CanonicalAgentModes())
}

func (h *NexusHandler) handleExecutionRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, nexus.CanonicalExecutionRoles())
}

func (h *NexusHandler) handleRuntimeExecutionRole(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/runtimes/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "execution-role" {
		writeError(w, http.StatusNotFound, "invalid runtime execution-role path")
		return
	}
	runtimeID := parts[0]

	switch r.Method {
	case http.MethodGet:
		view, err := h.nexus.EffectiveRuntimeExecutionRole(runtimeID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, view)
	case http.MethodPost:
		var body struct {
			RoleID string `json:"role_id"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil || strings.TrimSpace(body.RoleID) == "" {
			writeError(w, http.StatusBadRequest, "role_id is required")
			return
		}
		view, err := h.nexus.SetRuntimeExecutionRole(r.Context(), runtimeID, body.RoleID, "web")
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, view)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleAgentsList GET /api/v1/projects/{projectID}/agents
func (h *NexusHandler) handleAgentsList(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	agents, err := h.agents.List(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, agents)
}

// handleAgentCreate POST /api/v1/projects/{projectID}/agents  {name}
func (h *NexusHandler) handleAgentCreate(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	var body struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	agent, err := h.agents.Create(r.Context(), projectID, body.Name, body.Role)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, agent)
}

// handleAgentDetail GET/PATCH/DELETE /api/v1/agents/{id}
func (h *NexusHandler) handleAgentDetail(w http.ResponseWriter, r *http.Request) {
	id := agentIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing agent id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		detail, err := h.agents.Detail(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"agent":           detail.Agent,
			"generations":     detail.Generations,
			"lineage":         detail.Lineage,
			"revisions":       detail.Revisions,
			"effective_state": detail.EffectiveState,
			"recoverable":     detail.EffectiveState == store.AgentRecoverable,
		})
	case http.MethodPatch:
		var body struct {
			Name *string `json:"name"`
			Role *string `json:"role"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		agent, err := h.agents.Update(r.Context(), id, nexus.AgentPatch{Name: body.Name, Role: body.Role})
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, agent)
	case http.MethodDelete:
		detail, err := h.agents.Detail(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if err := h.agents.Delete(r.Context(), id, detail.Agent.ProjectID); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleAgentStart POST /api/v1/agents/{id}/start. Provider selection is
// intentionally read only here: it must be persisted through Resources first.
func (h *NexusHandler) handleAgentStart(w http.ResponseWriter, r *http.Request) {
	id := agentIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing agent id")
		return
	}
	var body struct {
		Provider string `json:"provider"`
		Profile  string `json:"profile"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body)

	provider, profile, err := h.nexus.ResolveStartParams(id, body.Provider, body.Profile)
	if err != nil {
		writeAgentConflict(w, err)
		return
	}

	sess, err := h.nexus.StartAgent(r.Context(), id, provider, profile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runtime": sess})
}

// handleAgentAsk POST /api/v1/agents/{id}/ask submits work to the existing
// persistent Agent. It never creates a new Agent.
func (h *NexusHandler) handleAgentAsk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := agentIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing agent id")
		return
	}
	var body struct {
		Prompt               string   `json:"prompt"`
		SkillIDs             []string `json:"skill_ids,omitempty"`
		Scope                string   `json:"scope,omitempty"`
		ExecutionRoleID      string   `json:"execution_role,omitempty"`
		StartIfNeeded        bool     `json:"start_if_needed"`
		ProjectID            string   `json:"project_id,omitempty"`
		ContextFingerprintID string   `json:"context_fingerprint_id,omitempty"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil || strings.TrimSpace(body.Prompt) == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}
	if strings.TrimSpace(body.ProjectID) != "" || strings.TrimSpace(body.ContextFingerprintID) != "" {
		if strings.TrimSpace(body.ProjectID) == "" || strings.TrimSpace(body.ContextFingerprintID) == "" {
			writeError(w, http.StatusConflict, "prepared task requires project_id and context_fingerprint_id")
			return
		}
		st, storeErr := h.nexus.OpenProject()
		if storeErr != nil {
			writeError(w, http.StatusServiceUnavailable, storeErr.Error())
			return
		}
		agent, agentErr := st.GetAgent(id, body.ProjectID)
		if agentErr != nil {
			writeError(w, http.StatusNotFound, "agent does not belong to the prepared project")
			return
		}
		readiness, readinessErr := h.nexus.ObserveContextReadiness(agent.ProjectID)
		if readinessErr != nil || readiness.State != nexus.ContextReady || readiness.CurrentFingerprintID != body.ContextFingerprintID {
			writeError(w, http.StatusConflict, "prepared task context is missing, stale, or changed")
			return
		}
	}

	compiled, err := h.nexus.CompileAgentPromptForProject(body.ProjectID, body.Prompt, body.SkillIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.nexus.AskAgentWithOptions(r.Context(), id, compiled.CompiledPrompt, nexus.AskAgentOptions{
		StartIfNeeded:         body.StartIfNeeded,
		ExecutionRoleID:       body.ExecutionRoleID,
		CompileSpecialization: false,
	})
	if err != nil {
		writeAgentConflict(w, err)
		return
	}

	// Persist immutable prompt audit receipt
	if st, stErr := h.nexus.OpenProject(); stErr == nil && result.RuntimeID != "" {
		_, _ = st.RecordAgentPromptReceipt(store.AgentPromptReceipt{
			AgentID:    id,
			RuntimeID:  result.RuntimeID,
			SkillIDs:   compiled.ValidatedSkills,
			PromptHash: compiled.PromptHash,
			Source:     "terminal_ask",
		})
	}

	writeJSON(w, http.StatusOK, result)
}

func writeAgentConflict(w http.ResponseWriter, err error) {
	message := err.Error()
	code := "CONFLICT"
	if strings.Contains(message, "REQUIRED_RESOURCE_SELECTION") {
		code = "REQUIRED_RESOURCE_SELECTION"
	}
	writeJSON(w, http.StatusConflict, map[string]string{"error": message, "code": code})
}

// handleAgentStop POST /api/v1/agents/{id}/stop
func (h *NexusHandler) handleAgentStop(w http.ResponseWriter, r *http.Request) {
	id := agentIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing agent id")
		return
	}
	if err := h.nexus.StopAgent(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

// handleAgentRecover POST /api/v1/agents/{id}/recover
func (h *NexusHandler) handleAgentRecover(w http.ResponseWriter, r *http.Request) {
	id := agentIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing agent id")
		return
	}
	sess, err := h.nexus.RecoverAgent(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runtime": sess})
}

// resolveAgentRuntimeID maps an agent to its current live runtime ID.
// A DB generation alone is not enough: after nexus web restart the generation
// may still point at a dead id. Only return ids present in the live registry.
func (h *NexusHandler) resolveAgentRuntimeID(agentID string) (string, error) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		return "", err
	}
	gen, err := st.CurrentGeneration(agentID)
	if err != nil {
		return "", err
	}
	if gen.RuntimeID == "" {
		return "", fmt.Errorf("empty runtime generation")
	}
	sess, ok := h.nexus.RuntimeRegistry().Get(gen.RuntimeID)
	if !ok || !sess.HostLive() {
		return "", fmt.Errorf("runtime generation %s is not live", gen.RuntimeID)
	}
	return gen.RuntimeID, nil
}

// handleAgentConfigGet GET /api/v1/agents/{id}/config
func (h *NexusHandler) handleAgentConfigGet(w http.ResponseWriter, r *http.Request) {
	id := agentIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing agent id")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	agent, err := st.GetAgent(id, "")
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var cfg nexus.AgentConfig
	if agent.CurrentRevisionID != "" {
		rev, rerr := st.GetRevision(agent.CurrentRevisionID)
		if rerr == nil {
			cfg, _ = nexus.ParseAgentConfig(rev.Config)
		}
	}
	revs, _ := st.ListRevisions(id)
	writeJSON(w, http.StatusOK, map[string]any{
		"config":    cfg,
		"revision":  agent.CurrentRevisionID,
		"revisions": revs,
	})
}

// handleAgentConfigApply POST /api/v1/agents/{id}/config/apply
func (h *NexusHandler) handleAgentConfigApply(w http.ResponseWriter, r *http.Request) {
	id := agentIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing agent id")
		return
	}
	var cfg nexus.AgentConfig
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid config body")
		return
	}
	if strings.TrimSpace(cfg.Provider) != "" {
		if strings.TrimSpace(cfg.Profile) == "" {
			cfg.Profile = "default"
		}
		if _, err := h.nexus.ValidateResource(cfg.Provider, cfg.Profile); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	}
	impact, err := h.nexus.SafeApply(r.Context(), id, cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"impact": impact})
}

// handleAgentConfigImpact POST /api/v1/agents/{id}/config/impact
func (h *NexusHandler) handleAgentConfigImpact(w http.ResponseWriter, r *http.Request) {
	id := agentIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing agent id")
		return
	}
	var cfg nexus.AgentConfig
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid config body")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	agent, err := st.GetAgent(id, "")
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var current nexus.AgentConfig
	if agent.CurrentRevisionID != "" {
		rev, rerr := st.GetRevision(agent.CurrentRevisionID)
		if rerr == nil {
			current, _ = nexus.ParseAgentConfig(rev.Config)
		}
	}
	current = nexus.NormalizeAgentSpec(agent, current)
	cfg = nexus.NormalizeAgentSpec(agent, cfg)
	impact := nexus.AnalyzeImpact(current, cfg)
	writeJSON(w, http.StatusOK, map[string]any{"impact": impact})
}
