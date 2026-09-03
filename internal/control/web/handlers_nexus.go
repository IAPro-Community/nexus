package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/buildinfo"
	coreconfig "github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/nexus"
	"github.com/kivervinicius/ai-cli/internal/nexus/intelligence"
	"github.com/kivervinicius/ai-cli/internal/nexus/runner"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"

	"time"

	"strconv"
)

// NexusHandler serves the Nexus product API (projects, agents, generations,
// lineage, layouts) on top of the shared control core.
type NexusHandler struct {
	auth                  *AuthManager
	nexus                 *nexus.Nexus
	hostFilesystemEnabled bool
}

func NewNexusHandler(auth *AuthManager) *NexusHandler {
	n := nexus.Default()
	// Wire runtime change notifications to the terminal broker (Gate 4).
	broker := DefaultBroker()
	n.SetRuntimeObservers(
		broker.NotifyRuntimeChanged,
		broker.NotifyAgentState,
		broker.NotifyContinuity,
	)
	return &NexusHandler{auth: auth, nexus: n, hostFilesystemEnabled: true}
}

func (h *NexusHandler) setHostFilesystemEnabled(enabled bool) {
	h.hostFilesystemEnabled = enabled
}

// handleProjectsList GET|POST /api/v1/projects
func (h *NexusHandler) handleProjectsList(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	switch r.Method {
	case http.MethodGet:
		list, err := st.ListProjects()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var body struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Path) == "" {
			writeError(w, http.StatusBadRequest, "path is required")
			return
		}
		canonical, err := store.CanonicalPath(body.Path)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		proj, err := st.CreateProject(store.Project{Name: body.Name, CanonicalPath: canonical})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, proj)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleProjectDetail GET/PATCH/DELETE /api/v1/projects/{id}
func (h *NexusHandler) handleProjectDetail(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/projects/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	id := parts[0]

	switch r.Method {
	case http.MethodGet:
		proj, err := st.GetProject(id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		// Touch MRU on access (P1 Project MRU).
		_ = st.TouchProject(id)
		layout, _ := st.GetLayout(id)
		writeJSON(w, http.StatusOK, map[string]any{"project": proj, "layout": layout})

	case http.MethodPatch:
		proj, err := st.GetProject(id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		var body struct {
			Name             *string `json:"name"`
			MaestroMode      *string `json:"maestro_mode"`
			DefaultIsolation *string `json:"default_isolation"`
			DefaultBranch    *string `json:"default_branch"`
			ResourcePolicy   *string `json:"resource_policy"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if body.Name != nil {
			proj.Name = *body.Name
			proj.Slug = store.Slugify(*body.Name)
		}
		if body.MaestroMode != nil {
			proj.MaestroMode = *body.MaestroMode
		}
		if body.DefaultIsolation != nil {
			proj.DefaultIsolation = *body.DefaultIsolation
		}
		if body.DefaultBranch != nil {
			proj.DefaultBranch = *body.DefaultBranch
		}
		if body.ResourcePolicy != nil {
			proj.ResourcePolicy = *body.ResourcePolicy
		}
		if err := st.UpdateProject(proj); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, proj)

	case http.MethodDelete:
		if err := h.nexus.DeleteProject(id); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
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
	readiness, err := h.nexus.PrepareContext(projectIDFromPath(r.URL.Path))
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
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"runtime": sess})
}

// handleProjectLayout GET/PUT /api/v1/projects/{id}/layout
func (h *NexusHandler) handleProjectLayout(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/projects/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	id := parts[0]
	switch r.Method {
	case http.MethodGet:
		layout, err := st.GetLayout(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"layout": layout})
	case http.MethodPut:
		var body struct {
			Layout string `json:"layout"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if err := st.SaveLayout(id, body.Layout); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleAgentsList GET /api/v1/projects/{projectID}/agents
func (h *NexusHandler) handleAgentsList(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	agents, err := st.ListAgents(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Reconcile effective live state for every agent in the list (§29 honest state).
	for i := range agents {
		if eff, err := h.nexus.EffectiveAgentState(agents[i].ID); err == nil && eff != "" {
			agents[i].Status = eff
		}
	}
	writeJSON(w, http.StatusOK, agents)
}

// handleAgentCreate POST /api/v1/projects/{projectID}/agents  {name}
func (h *NexusHandler) handleAgentCreate(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	var body struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: projectID, Name: body.Name, Role: body.Role})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, agent)
}

// handleAgentDetail GET/PATCH/DELETE /api/v1/agents/{id}
func (h *NexusHandler) handleAgentDetail(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	id := agentIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusNotFound, "missing agent id")
		return
	}
	agent, err := st.GetAgent(id, "")
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet:
		generations, _ := st.ListGenerations(id)
		lineage, _ := st.ListLineage(id)
		revisions, _ := st.ListRevisions(id)
		effectiveState, _ := h.nexus.EffectiveAgentState(id)
		if effectiveState != "" {
			agent.Status = effectiveState
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"agent":           agent,
			"generations":     generations,
			"lineage":         lineage,
			"revisions":       revisions,
			"effective_state": effectiveState,
			"recoverable":     effectiveState == store.AgentRecoverable,
		})
	case http.MethodPatch:
		var body struct {
			Name *string `json:"name"`
			Role *string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if body.Name != nil {
			agent.Name = *body.Name
		}
		if body.Role != nil {
			agent.Role = *body.Role
		}
		if err := st.UpdateAgent(agent); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, agent)
	case http.MethodDelete:
		if err := h.nexus.DeleteAgent(id, agent.ProjectID); err != nil {
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
	_ = json.NewDecoder(r.Body).Decode(&body)

	provider, profile, err := h.nexus.ResolveStartParams(id, body.Provider, body.Profile)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
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
		Prompt        string `json:"prompt"`
		StartIfNeeded bool   `json:"start_if_needed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Prompt) == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}
	result, err := h.nexus.AskAgent(r.Context(), id, body.Prompt, body.StartIfNeeded)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
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

// resolveAgentRuntimeID maps an agent to its current runtime generation ID.
func (h *NexusHandler) resolveAgentRuntimeID(agentID string) (string, error) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		return "", err
	}
	gen, err := st.CurrentGeneration(agentID)
	if err != nil {
		return "", err
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
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
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
	impact, err := h.nexus.SafeApply(context.Background(), id, cfg)
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
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
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
	impact := nexus.AnalyzeImpact(current, cfg)
	writeJSON(w, http.StatusOK, map[string]any{"impact": impact})
}

// handleResourcesList GET /api/v1/resources — returns available provider accounts
// and the scheduler recommendation for the current context.
func (h *NexusHandler) handleResourcesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	accounts, err := h.nexus.ListResources()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if accounts == nil {
		accounts = []nexus.ProviderAccount{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"accounts": accounts,
		"policy":   "BALANCED",
	})
}

// handleResourceSelect POST /api/v1/resources/select — persist a manually
// selected, eligible resource for an Agent.
func (h *NexusHandler) handleResourceSelect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Provider string `json:"provider"`
		Profile  string `json:"profile"`
		Policy   string `json:"policy"`
		AgentID  string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if strings.TrimSpace(body.AgentID) == "" || strings.TrimSpace(body.Provider) == "" || strings.TrimSpace(body.Profile) == "" {
		writeError(w, http.StatusBadRequest, "agent_id, provider and profile are required")
		return
	}
	if body.Policy != "" && nexus.SchedulerPolicy(body.Policy) != nexus.PolicyManual {
		writeError(w, http.StatusBadRequest, "resource allocation requires MANUAL policy; use a recommendation endpoint for automatic selection")
		return
	}
	allocation, err := h.nexus.AllocateResource(context.Background(), body.AgentID, body.Provider, body.Profile, nexus.PolicyManual)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, allocation)
}

// handleResourceRecommend POST /api/v1/resources/recommend — evaluate accounts against TaskRequirements
func (h *NexusHandler) handleResourceRecommend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Requirements nexus.TaskRequirements `json:"requirements"`
		Policy       string                 `json:"policy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	accounts, err := h.nexus.ListResources()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	policy := nexus.SchedulerPolicy(strings.ToUpper(req.Policy))
	if policy == "" {
		policy = nexus.PolicyBalanced
	}
	result := nexus.RecommendResources(accounts, req.Requirements, policy)
	writeJSON(w, http.StatusOK, result)
}

// handleMaestroStatus GET /api/v1/maestro — returns Maestro integration status.
func (h *NexusHandler) handleMaestroStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	client := nexus.NewMaestroClient()
	status := client.Status()
	writeJSON(w, http.StatusOK, status)
}

// handleMaestroAdvice POST /api/v1/maestro/advice — request Maestro recommendations.
func (h *NexusHandler) handleMaestroAdvice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		ProjectID string `json:"project_id"`
		AgentID   string `json:"agent_id"`
		Intent    string `json:"intent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	client := nexus.NewMaestroClient()
	ctx := nexus.AdviceContext{
		ProjectID: body.ProjectID,
		AgentID:   body.AgentID,
	}
	resp, err := client.GetAdvice(ctx, body.Intent)
	if err != nil {
		// Return degraded response, not 500.
		writeJSON(w, http.StatusOK, map[string]any{
			"mode":     "OFF",
			"error":    err.Error(),
			"degraded": true,
		})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleSystemUpdates GET /api/v1/system/updates — returns status of Nexus & Maestro versions.
func (h *NexusHandler) handleSystemUpdates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	client := nexus.NewMaestroClient()
	mStatus := client.Status()
	maestroVer := "unknown"
	if mStatus.Capabilities != nil {
		maestroVer = mStatus.Capabilities.Version
	}

	// Check npm registry for latest maestro version if available
	latestMaestroVer := maestroVer
	updateAvailable := false
	if npmPath, err := exec.LookPath("npm"); err == nil {
		cmd := exec.Command(npmPath, "view", "@iapro/orquestrador-maestro-cli", "version")
		if out, err := cmd.Output(); err == nil {
			latestMaestroVer = strings.TrimSpace(string(out))
			if latestMaestroVer != "" && maestroVer != "unknown" && latestMaestroVer != maestroVer {
				updateAvailable = true
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"nexus_version":          buildinfo.Version,
		"nexus_commit":           buildinfo.Commit,
		"nexus_build_date":       buildinfo.BuildDate,
		"maestro_version":        maestroVer,
		"maestro_latest_version": latestMaestroVer,
		"maestro_available":      mStatus.Available,
		"update_available":       updateAvailable,
	})
}

// handleSystemUpdate POST /api/v1/system/update — triggers update routine for Nexus & Maestro.
func (h *NexusHandler) handleSystemUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeError(w, http.StatusNotImplemented, "system update not yet implemented")
}

// handleMissionsList GET /api/v1/projects/{id}/missions — list missions for a project.
func (h *NexusHandler) handleMissionsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	missions, err := st.ListMissions(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"missions": missions})
}

// handleMissionCreate POST /api/v1/projects/{id}/missions — create a mission.
func (h *NexusHandler) handleMissionCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Goal        string `json:"goal"`
		Scope       string `json:"scope"`
		RiskLevel   string `json:"risk_level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	m := &store.Mission{
		ProjectID:   projectID,
		Name:        body.Name,
		Description: body.Description,
		Goal:        body.Goal,
		Scope:       body.Scope,
		RiskLevel:   body.RiskLevel,
	}
	if err := st.CreateMission(m); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

// handleMissionDetail GET/PATCH/DELETE /api/v1/missions/{id}
func (h *NexusHandler) handleMissionDetail(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	missionID := missionIDFromPath(r.URL.Path)
	if missionID == "" {
		writeError(w, http.StatusNotFound, "missing mission id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		m, err := st.GetMission(missionID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		tasks, _ := st.ListTasks(missionID)
		assignments, _ := st.ListAssignments(missionID)
		total, pending, active, completed, failed, _ := st.MissionStats(missionID)
		writeJSON(w, http.StatusOK, map[string]any{
			"mission":     m,
			"tasks":       tasks,
			"assignments": assignments,
			"stats":       map[string]int{"total": total, "pending": pending, "active": active, "completed": completed, "failed": failed},
		})

	case http.MethodPatch:
		m, err := st.GetMission(missionID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		var body struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
			Status      *string `json:"status"`
			Goal        *string `json:"goal"`
			Scope       *string `json:"scope"`
			RiskLevel   *string `json:"risk_level"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if body.Name != nil {
			m.Name = *body.Name
		}
		if body.Description != nil {
			m.Description = *body.Description
		}
		if body.Status != nil {
			m.Status = *body.Status
		}
		if body.Goal != nil {
			m.Goal = *body.Goal
		}
		if body.Scope != nil {
			m.Scope = *body.Scope
		}
		if body.RiskLevel != nil {
			m.RiskLevel = *body.RiskLevel
		}
		if err := st.UpdateMission(m); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, m)

	case http.MethodDelete:
		if err := st.DeleteMission(missionID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleMissionTaskCreate POST /api/v1/missions/{id}/tasks — add a task to a mission.
func (h *NexusHandler) handleMissionTaskCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	missionID := missionIDFromPath(r.URL.Path)
	if missionID == "" {
		writeError(w, http.StatusNotFound, "missing mission id")
		return
	}
	var body struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		Kind         string `json:"kind"`
		Priority     int    `json:"priority"`
		Dependencies string `json:"dependencies"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	t := &store.MissionTask{
		MissionID:    missionID,
		Name:         body.Name,
		Description:  body.Description,
		Kind:         body.Kind,
		Priority:     body.Priority,
		Dependencies: body.Dependencies,
	}
	if err := st.CreateTask(t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// handleMissionAssign POST /api/v1/missions/{id}/assign — assign an agent to a task.
func (h *NexusHandler) handleMissionAssign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	missionID := missionIDFromPath(r.URL.Path)
	if missionID == "" {
		writeError(w, http.StatusNotFound, "missing mission id")
		return
	}
	var body struct {
		TaskID  string `json:"task_id"`
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	a := &store.MissionAssignment{
		MissionID: missionID,
		TaskID:    body.TaskID,
		AgentID:   body.AgentID,
	}
	if err := st.CreateAssignment(a); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func projectIDFromPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/projects/"), "/")
	if len(parts) >= 1 && strings.HasSuffix(path, "/agents") {
		return parts[0]
	}
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func agentIDFromPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/agents/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func missionIDFromPath(path string) string {
	// Handle /api/v1/missions/{id} and /api/v1/missions/{id}/tasks and /api/v1/missions/{id}/assign
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/missions/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// ProjectGitBranchesResponse is returned by GET /api/v1/projects/:id/git/branches
type ProjectGitBranchesResponse struct {
	ProjectID      string   `json:"project_id"`
	CanonicalPath  string   `json:"canonical_path"`
	CurrentBranch  string   `json:"current_branch"`
	DefaultBranch  string   `json:"default_branch"`
	Branches       []string `json:"branches"`
	RemoteBranches []string `json:"remote_branches"`
	IsClean        bool     `json:"is_clean"`
	ModifiedCount  int      `json:"modified_count"`
}

// handleProjectGitBranches GET /api/v1/projects/{projectID}/git/branches
func (h *NexusHandler) handleProjectGitBranches(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	proj, err := st.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	dir := proj.CanonicalPath
	current := getGitBranch(dir)
	if current == "" {
		current = proj.DefaultBranch
	}
	if current == "" {
		current = "main"
	}

	// 1. List local branches
	var branches []string
	cmdList := exec.Command("git", "branch", "--list", "--no-color")
	cmdList.Dir = dir
	if out, err := cmdList.Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		for _, l := range lines {
			name := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "*"))
			if name != "" && !strings.Contains(name, "->") {
				branches = append(branches, name)
			}
		}
	}
	if len(branches) == 0 {
		branches = []string{current}
	}

	// 2. List remote branches
	var remoteBranches []string
	cmdRemote := exec.Command("git", "branch", "-r", "--no-color")
	cmdRemote.Dir = dir
	if out, err := cmdRemote.Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		for _, l := range lines {
			name := strings.TrimSpace(l)
			if name != "" && !strings.Contains(name, "->") {
				remoteBranches = append(remoteBranches, name)
			}
		}
	}

	// 3. Status check for uncommitted changes
	isClean := true
	modifiedCount := 0
	cmdStatus := exec.Command("git", "status", "--porcelain")
	cmdStatus.Dir = dir
	if out, err := cmdStatus.Output(); err == nil {
		trimmed := strings.TrimSpace(string(out))
		if trimmed != "" {
			lines := strings.Split(trimmed, "\n")
			modifiedCount = len(lines)
			isClean = false
		}
	}

	resp := ProjectGitBranchesResponse{
		ProjectID:      proj.ID,
		CanonicalPath:  proj.CanonicalPath,
		CurrentBranch:  current,
		DefaultBranch:  proj.DefaultBranch,
		Branches:       branches,
		RemoteBranches: remoteBranches,
		IsClean:        isClean,
		ModifiedCount:  modifiedCount,
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleProjectGitCheckout POST /api/v1/projects/{projectID}/git/checkout
func (h *NexusHandler) handleProjectGitCheckout(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	proj, err := st.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var body struct {
		Branch string `json:"branch"`
		Create bool   `json:"create"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Branch) == "" {
		writeError(w, http.StatusBadRequest, "branch is required")
		return
	}
	targetBranch := strings.TrimSpace(body.Branch)
	// Basic sanitization against flags, illegal chars, and shell metacharacters
	blockedChars := " ~^:?*[\\\"'$(" + "`)"
	if strings.HasPrefix(targetBranch, "-") || strings.ContainsAny(targetBranch, blockedChars) {
		writeError(w, http.StatusBadRequest, "invalid branch name")
		return
	}

	// Safety policy: Check if any agent is actively working in the canonical project tree (A10).
	agents, _ := st.ListAgents(projectID)
	for _, a := range agents {
		eff, _ := h.nexus.EffectiveAgentState(a.ID)
		if eff == store.AgentWorking || eff == store.AgentStarting || eff == store.AgentRecovering {
			isDirectCanonical := true
			if a.CurrentRevisionID != "" {
				if rev, rerr := st.GetRevision(a.CurrentRevisionID); rerr == nil {
					if cfg, perr := nexus.ParseAgentConfig(rev.Config); perr == nil {
						if cfg.Isolation == "worktree" {
							isDirectCanonical = false
						}
					}
				}
			}
			if isDirectCanonical {
				writeError(w, http.StatusConflict, fmt.Sprintf("cannot checkout branch: agent %q (%s) is actively running in the project workspace (stop agent or migrate to worktree isolation first)", a.Name, a.ID))
				return
			}
		}
	}

	dir := proj.CanonicalPath
	var cmd *exec.Cmd
	if body.Create {
		cmd = exec.Command("git", "checkout", "-b", targetBranch)
	} else {
		cmd = exec.Command("git", "checkout", targetBranch)
	}
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("git checkout failed: %s (%s)", strings.TrimSpace(string(out)), err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":        true,
		"current_branch": targetBranch,
		"output":         strings.TrimSpace(string(out)),
	})
}

// handleProjectPlans GET/POST /api/v1/projects/{projectID}/plans
func (h *NexusHandler) handleProjectPlans(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		plans, err := h.nexus.ListWorkPlans(r.Context(), projectID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if plans == nil {
			plans = []store.WorkPlan{}
		}
		writeJSON(w, http.StatusOK, plans)

	case http.MethodPost:
		var body struct {
			Title       string            `json:"title"`
			Description string            `json:"description"`
			Goal        string            `json:"goal"`
			AutoPlan    bool              `json:"auto_plan"`
			Phases      []store.PlanPhase `json:"phases"`
			Facts       map[string]string `json:"facts"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}

		if body.AutoPlan && strings.TrimSpace(body.Goal) != "" {
			plan, err := h.nexus.GeneratePlanFromIntent(r.Context(), projectID, body.Goal)
			if err != nil {
				var clarificationErr *nexus.ClarificationRequiredError
				switch {
				case errors.As(err, &clarificationErr):
					writeJSON(w, http.StatusConflict, map[string]any{
						"error":         "clarification_required",
						"clarification": clarificationErr.Checkpoint,
					})
				case errors.Is(err, intelligence.ErrIntelligenceUnavailable):
					writeJSON(w, http.StatusServiceUnavailable, map[string]any{
						"error":  "intelligence_unavailable",
						"detail": err.Error(),
					})
				default:
					var contextErr *nexus.ComposerContextNotReadyError
					if errors.As(err, &contextErr) {
						writeJSON(w, http.StatusConflict, map[string]any{"error": "context_not_ready", "detail": err.Error(), "readiness": contextErr.Readiness})
					} else {
						writeError(w, http.StatusBadGateway, err.Error())
					}
				}
				return
			}
			writeJSON(w, http.StatusCreated, plan)
			return
		}

		if strings.TrimSpace(body.Title) == "" {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}

		plan, err := h.nexus.CreateWorkPlan(r.Context(), projectID, body.Title, body.Description, body.Phases, body.Facts)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, plan)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handlePlanDetail GET/PUT/DELETE /api/v1/plans/{id}
func (h *NexusHandler) handlePlanDetail(w http.ResponseWriter, r *http.Request) {
	planID := strings.TrimPrefix(r.URL.Path, "/api/v1/plans/")
	slashIdx := strings.Index(planID, "/")
	if slashIdx != -1 {
		planID = planID[:slashIdx]
	}
	if planID == "" {
		writeError(w, http.StatusNotFound, "missing plan id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		plan, err := h.nexus.GetWorkPlan(r.Context(), planID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		revisions, _ := h.nexus.ListPlanRevisions(r.Context(), planID)
		writeJSON(w, http.StatusOK, map[string]any{
			"plan":      plan,
			"revisions": revisions,
		})

	case http.MethodPut:
		var body struct {
			Plan          store.WorkPlan `json:"plan"`
			ChangeSummary string         `json:"change_summary"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		body.Plan.ID = planID
		updated, rev, err := h.nexus.UpdateWorkPlan(r.Context(), body.Plan, body.ChangeSummary)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"plan":     updated,
			"revision": rev,
		})

	case http.MethodDelete:
		if err := h.nexus.DeleteWorkPlan(r.Context(), planID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handlePlanCompile POST /api/v1/plans/{id}/compile
func (h *NexusHandler) handlePlanCompile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	planID := strings.TrimPrefix(r.URL.Path, "/api/v1/plans/")
	planID = strings.TrimSuffix(planID, "/compile")
	if planID == "" {
		writeError(w, http.StatusNotFound, "missing plan id")
		return
	}

	var body struct {
		PhaseID   string `json:"phase_id"`
		PackageID string `json:"package_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PackageID == "" {
		writeError(w, http.StatusBadRequest, "package_id is required")
		return
	}

	compiled, err := h.nexus.CompilePackagePrompt(r.Context(), planID, body.PhaseID, body.PackageID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, compiled)
}

// handlePlanRun POST /api/v1/plans/{id}/run
func (h *NexusHandler) handlePlanRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	planID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/plans/"), "/run")
	if planID == "" {
		writeError(w, http.StatusBadRequest, "plan id is required")
		return
	}
	var body struct {
		AgentID               string   `json:"agent_id"`
		PlanRevision          int      `json:"plan_revision"`
		MaxRetry              int      `json:"max_retries"`
		MaxTotalIterations    int      `json:"max_total_iterations"`
		PackageTimeoutSeconds int      `json:"package_timeout_seconds"`
		VerificationCommands  []string `json:"verification_commands"`
		Autonomous            *bool    `json:"autonomous"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.PlanRevision <= 0 {
		writeError(w, http.StatusBadRequest, "plan_revision is required")
		return
	}
	contract := runner.DefaultAutonomyContract()
	if body.MaxRetry > 0 {
		contract.MaxRetries = body.MaxRetry
	}
	if body.MaxTotalIterations > 0 {
		contract.MaxTotalIterations = body.MaxTotalIterations
	}
	if body.PackageTimeoutSeconds > 0 {
		contract.PackageTimeoutSeconds = body.PackageTimeoutSeconds
	}
	if len(body.VerificationCommands) > 0 {
		contract.VerificationCommands = body.VerificationCommands
	}
	autonomous := true
	if body.Autonomous != nil {
		autonomous = *body.Autonomous
	}
	run, err := h.nexus.StartMissionRunApproved(r.Context(), planID, body.PlanRevision, body.AgentID, contract, autonomous)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

// handleRunsList GET /api/v1/runs
func (h *NexusHandler) handleRunsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	runs, err := h.nexus.Runner().ListRuns(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

// handleRunDetail exposes durable MissionRun state and explicit control actions.
// POST /step exists for diagnostics; normal product execution uses the background worker.
func (h *NexusHandler) handleRunDetail(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/runs/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	runID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = strings.ToLower(parts[1])
	}

	if r.Method == http.MethodGet && action == "" {
		run, err := h.nexus.Runner().GetRun(r.Context(), runID)
		if err != nil {
			writeError(w, http.StatusNotFound, "run not found")
			return
		}
		writeJSON(w, http.StatusOK, run)
		return
	}
	if r.Method == http.MethodGet && action == "evidence" {
		h.handleRunEvidence(w, r, runID)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	var (
		run *runner.MissionRun
		err error
	)
	switch action {
	case "pause":
		run, err = h.nexus.PauseMissionRun(r.Context(), runID, firstNonEmpty(body.Reason, "paused by user"))
	case "take-control":
		run, err = h.nexus.TakeControlMissionRun(r.Context(), runID, firstNonEmpty(body.Reason, "manual takeover"))
	case "resume":
		run, err = h.nexus.ResumeMissionRun(r.Context(), runID)
	case "return-to-mission":
		run, err = h.nexus.ReturnMissionRun(r.Context(), runID)
	case "cancel":
		run, err = h.nexus.CancelMissionRun(r.Context(), runID, firstNonEmpty(body.Reason, "canceled by user"))
	case "step", "":
		var done bool
		run, done, err = h.nexus.Runner().ExecuteNextStep(r.Context(), runID)
		if err == nil {
			writeJSON(w, http.StatusOK, map[string]any{"run": run, "completed": done})
			return
		}
	default:
		writeError(w, http.StatusNotFound, "unknown run action")
		return
	}
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// handleRunEvidence returns only typed, bounded Flow evidence. Raw Agent or
// provider transcripts are intentionally absent from this API because they are
// not part of ContextCapsule/WorkReceipt persistence.
func (h *NexusHandler) handleRunEvidence(w http.ResponseWriter, r *http.Request, runID string) {
	if _, err := h.nexus.Runner().GetRun(r.Context(), runID); err != nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	capsuleRecords, err := st.ListFlowContextCapsules(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	receiptRecords, err := st.ListFlowWorkReceipts(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	capsules := make([]runner.ContextCapsule, 0, len(capsuleRecords))
	for _, record := range capsuleRecords {
		var capsule runner.ContextCapsule
		if err := json.Unmarshal([]byte(record.ContentJSON), &capsule); err != nil {
			writeError(w, http.StatusInternalServerError, "decode persisted context capsule")
			return
		}
		capsules = append(capsules, capsule)
	}
	receipts := make([]runner.WorkReceipt, 0, len(receiptRecords))
	for _, record := range receiptRecords {
		var receipt runner.WorkReceipt
		if err := json.Unmarshal([]byte(record.ContentJSON), &receipt); err != nil {
			writeError(w, http.StatusInternalServerError, "decode persisted work receipt")
			return
		}
		receipts = append(receipts, receipt)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"run_id":   runID,
		"capsules": capsules,
		"receipts": receipts,
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// handleIntelligence GET/PUT /api/v1/intelligence exposes secret-free Intelligence routing configuration.
func (h *NexusHandler) handleIntelligence(w http.ResponseWriter, r *http.Request) {
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	switch r.Method {
	case http.MethodGet:
		status := h.nexus.IntelligenceStatus(r.Context(), projectID)
		writeJSON(w, http.StatusOK, status)
	case http.MethodPut:
		var body coreconfig.IntelligenceConfig
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
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
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
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

// handleMissionSchedules creates/lists/cancels durable Mission schedules.
func (h *NexusHandler) handleMissionSchedules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := h.nexus.ListMissionSchedules(r.Context(), strings.TrimSpace(r.URL.Query().Get("project_id")))
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var body struct {
			PlanID       string                  `json:"plan_id"`
			Mode         string                  `json:"mode"`
			ScheduledFor string                  `json:"scheduled_for"`
			AfterRunID   string                  `json:"after_run_id"`
			AgentID      string                  `json:"agent_id"`
			CancelID     string                  `json:"cancel_id"`
			Contract     runner.AutonomyContract `json:"contract"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if body.CancelID != "" {
			if err := h.nexus.CancelMissionSchedule(r.Context(), body.CancelID); err != nil {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]bool{"canceled": true})
			return
		}
		contract := body.Contract
		if contract.MaxRetries == 0 {
			contract = runner.DefaultAutonomyContract()
		}
		var when *time.Time
		if strings.TrimSpace(body.ScheduledFor) != "" {
			parsed, err := time.Parse(time.RFC3339, body.ScheduledFor)
			if err != nil {
				writeError(w, http.StatusBadRequest, "scheduled_for must be RFC3339")
				return
			}
			when = &parsed
		}
		item, err := h.nexus.ScheduleMission(r.Context(), body.PlanID, body.Mode, when, body.AfterRunID, body.AgentID, contract)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *NexusHandler) handlePlanRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	planID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/plans/"), "/restore")
	var body struct {
		Revision int `json:"revision"`
	}
	if json.NewDecoder(r.Body).Decode(&body) != nil || body.Revision <= 0 {
		writeError(w, http.StatusBadRequest, "revision is required")
		return
	}
	plan, rev, err := h.nexus.RestoreWorkPlanRevision(r.Context(), planID, body.Revision)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plan": plan, "revision": rev})
}

func (h *NexusHandler) handlePlanDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	planID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/plans/"), "/diff")
	from, err1 := strconv.Atoi(r.URL.Query().Get("from"))
	to, err2 := strconv.Atoi(r.URL.Query().Get("to"))
	if err1 != nil || err2 != nil || from <= 0 || to <= 0 {
		writeError(w, http.StatusBadRequest, "from/to revisions are required")
		return
	}
	diff, err := h.nexus.ComparePlanRevisions(r.Context(), planID, from, to)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, diff)
}
