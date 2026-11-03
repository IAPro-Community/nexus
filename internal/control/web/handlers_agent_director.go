package web

import (
    "errors"
    "net/http"
    "strings"

    "github.com/IAPro-Community/nexus/internal/nexus/store"
)

// Director is a read-only projection over the existing Agent/RuntimeGeneration
// and SessionHost. There is no second worker registry or orchestration engine.
type directorAction struct {
    Kind string `json:"kind"`
    Availability string `json:"availability"`
    Reason string `json:"reason"`
}
type directorProjection struct {
    SchemaVersion int `json:"schema_version"`
    ExecutionOwner string `json:"execution_owner"`
    AgentID string `json:"agent_id"`
    ProjectID string `json:"project_id"`
    AgentStatus string `json:"agent_status"`
    ProcessState string `json:"process_state"`
    ProcessEvidence string `json:"process_evidence"`
    RuntimeID string `json:"runtime_id,omitempty"`
    GenerationID string `json:"generation_id,omitempty"`
    Provider string `json:"provider,omitempty"`
    Profile string `json:"profile,omitempty"`
    Actions []directorAction `json:"actions"`
}

func projectDirectorAgent(agent store.Agent, gen *store.RuntimeGeneration, processLive bool) directorProjection {
    view := directorProjection{
        SchemaVersion: 1,
        ExecutionOwner: "NEXUS",
        AgentID: agent.ID,
        ProjectID: agent.ProjectID,
        AgentStatus: agent.Status,
        ProcessState: "UNKNOWN",
        ProcessEvidence: "NO_LIVE_RUNTIME_OBSERVED",
        Actions: []directorAction{
            {Kind: "INSPECT", Availability: "SUPPORTED", Reason: "READ_ONLY_AGENT_PROJECTION"},
            // Existing TaskPreparationDialog is the only safe path for new
            // user-directed work; it does not promise live steering.
            {Kind: "DIRECTIVE", Availability: "PREPARE_ONLY", Reason: "USE_EXISTING_TASK_PREPARATION"},
            {Kind: "QUEUE_NEXT", Availability: "UNSUPPORTED", Reason: "NO_DURABLE_OWNER_QUEUE"},
            {Kind: "STEER", Availability: "UNSUPPORTED", Reason: "NO_VERIFIED_TURN_STEERING"},
            {Kind: "WAIT", Availability: "OBSERVE_ONLY", Reason: "REFRESH_OBSERVED_STATE"},
            {Kind: "CANCEL", Availability: "TERMINAL_ONLY", Reason: "TURN_CANCEL_NOT_VERIFIED"},
        },
    }
    if gen == nil { return view }
    view.RuntimeID = gen.RuntimeID
    view.GenerationID = gen.ID
    view.Provider = gen.Provider
    view.Profile = gen.Profile
    if processLive {
        view.ProcessState = "LIVE"
        view.ProcessEvidence = "SESSION_HOST_LIVE"
    } else {
        view.ProcessState = "NOT_LIVE"
        view.ProcessEvidence = "GENERATION_NOT_IN_LIVE_REGISTRY"
    }
    // Never convert a live PTY into an inferred RUNNING/IDLE *turn*.
    return view
}

// handleAgentDirector provides a truthful read model, never a mutation API.
// routeAgent enforces existing session authentication and CSRF policy.
func (h *NexusHandler) handleAgentDirector(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        writeError(w, http.StatusMethodNotAllowed, "director mutations are unsupported")
        return
    }
    projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
    agentID := agentIDFromPath(r.URL.Path)
    if projectID == "" || agentID == "" {
        writeError(w, http.StatusBadRequest, "project_id and agent id are required")
        return
    }
    st, err := h.nexus.OpenProject()
    if err != nil {
        writeError(w, http.StatusServiceUnavailable, err.Error())
        return
    }
    agent, err := st.GetAgent(agentID, projectID)
    if err != nil {
        writeError(w, http.StatusNotFound, "agent not found in project")
        return
    }
    view, err := h.directorViewForStoredAgent(st, agent)
    if err != nil {
        writeError(w, http.StatusServiceUnavailable, err.Error())
        return
    }
    w.Header().Set("Cache-Control", "no-store")
    writeJSON(w, http.StatusOK, view)
}

func (h *NexusHandler) directorViewForStoredAgent(st *store.Store, agent store.Agent) (directorProjection, error) {
    var generation *store.RuntimeGeneration
    gen, err := st.CurrentGeneration(agent.ID)
    if err == nil {
        generation = &gen
    } else if !errors.Is(err, store.ErrNotFound) {
        return directorProjection{}, err
    }
    live := false
    if generation != nil && generation.RuntimeID != "" {
        if session, exists := h.nexus.RuntimeRegistry().Get(generation.RuntimeID); exists {
            live = session.HostLive()
        }
    }
    return projectDirectorAgent(agent, generation, live), nil
}
