package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
	"github.com/IAPro-Community/nexus/internal/profile"
)

// Director is a read-only projection over the existing Agent/RuntimeGeneration
// and SessionHost. There is no second worker registry or orchestration engine.
type directorAction struct {
	Kind         string `json:"kind"`
	Availability string `json:"availability"`
	Reason       string `json:"reason"`
}

// Event summaries and account scopes may contain user prompts or credentials.
// Director exposes only bounded operational metadata already persisted by Nexus.
type directorEvent struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	CorrelationID string `json:"correlation_id,omitempty"`
	Timestamp     string `json:"ts"`
}

func projectDirectorEvents(records []store.EventMetadata) []directorEvent {
	events := make([]directorEvent, 0, len(records))
	for _, item := range records {
		events = append(events, directorEvent{
			ID: item.ID, Kind: item.Kind, CorrelationID: item.CorrelationID,
			Timestamp: item.Timestamp.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return events
}

// directorLineage exposes the handoff edges of an Agent. Provider session
// identifiers stay out of the projection; only durable runtime/checkpoint
// identities are observable.
type directorLineage struct {
	ID            string    `json:"id"`
	Relation      string    `json:"relation"`
	SourceRuntime string    `json:"source_runtime"`
	TargetRuntime string    `json:"target_runtime"`
	CheckpointID  string    `json:"checkpoint_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// directorQuota keeps the observed quota provenance next to the value. An
// unmeasured quota stays UNKNOWN; it never becomes zero.
type directorQuota struct {
	Status           string   `json:"status"`
	Source           string   `json:"source"`
	RemainingPercent *float64 `json:"remaining_percent,omitempty"`
	Window           string   `json:"window,omitempty"`
}

func projectDirectorLineage(entries []store.LineageEntry) []directorLineage {
	if len(entries) == 0 {
		return nil
	}
	projected := make([]directorLineage, 0, len(entries))
	for _, entry := range entries {
		projected = append(projected, directorLineage{
			ID:            entry.ID,
			Relation:      entry.Relation,
			SourceRuntime: entry.SourceRuntime,
			TargetRuntime: entry.TargetRuntime,
			CheckpointID:  entry.CheckpointID,
			CreatedAt:     entry.CreatedAt.UTC(),
		})
	}
	return projected
}

func projectDirectorQuota(providerID, profileID string) directorQuota {
	if strings.TrimSpace(providerID) == "" || strings.TrimSpace(profileID) == "" {
		return directorQuota{Status: "UNKNOWN", Source: "UNKNOWN"}
	}
	view := profile.GetQuotaView(providerID, profileID, "", "")
	quota := directorQuota{Status: view.Status, Source: view.Source}
	if remaining, window := view.Bottleneck(); window != "" {
		quota.RemainingPercent = &remaining
		quota.Window = window
	}
	return quota
}

// agentRevisionWorkspace reports the isolation and workspace declared by the
// newest Agent revision. It never inspects the filesystem and never guesses a
// worktree path.
func agentRevisionWorkspace(revisions []store.AgentRevision) (string, string) {
	if len(revisions) == 0 {
		return "", ""
	}
	var config struct {
		Isolation string `json:"isolation"`
		Workspace string `json:"workspace"`
	}
	if json.Unmarshal([]byte(revisions[0].Config), &config) != nil {
		return "", ""
	}
	return strings.TrimSpace(config.Isolation), strings.TrimSpace(config.Workspace)
}

type directorProjection struct {
	SchemaVersion   int               `json:"schema_version"`
	ExecutionOwner  string            `json:"execution_owner"`
	AgentID         string            `json:"agent_id"`
	ProjectID       string            `json:"project_id"`
	AgentStatus     string            `json:"agent_status"`
	ProcessState    string            `json:"process_state"`
	ProcessEvidence string            `json:"process_evidence"`
	ObservedState   string            `json:"observed_state"`
	ObservedSource  string            `json:"observed_source"`
	ObservedReason  string            `json:"observed_reason"`
	RuntimeID       string            `json:"runtime_id,omitempty"`
	GenerationID    string            `json:"generation_id,omitempty"`
	Provider        string            `json:"provider,omitempty"`
	Profile         string            `json:"profile,omitempty"`
	Workspace       string            `json:"workspace,omitempty"`
	Worktree        string            `json:"worktree,omitempty"`
	Quota           directorQuota     `json:"quota"`
	Lineage         []directorLineage `json:"lineage,omitempty"`
	Actions         []directorAction  `json:"actions"`
	RecentEvents    []directorEvent   `json:"recent_events,omitempty"`
}

func projectDirectorAgent(agent store.Agent, gen *store.RuntimeGeneration, processLive bool) directorProjection {
	view := directorProjection{
		SchemaVersion:   1,
		ExecutionOwner:  "NEXUS",
		AgentID:         agent.ID,
		ProjectID:       agent.ProjectID,
		AgentStatus:     agent.Status,
		ProcessState:    "UNKNOWN",
		ProcessEvidence: "NO_LIVE_RUNTIME_OBSERVED",
		ObservedState:   "unknown",
		ObservedSource:  "UNKNOWN",
		ObservedReason:  "NO_CURRENT_GENERATION",
		Actions: []directorAction{
			{Kind: "INSPECT", Availability: "SUPPORTED", Reason: "READ_ONLY_AGENT_PROJECTION"},
			// Existing TaskPreparationDialog is the only safe path for new
			// user-directed work; it does not promise live steering.
			{Kind: "DIRECTIVE", Availability: "PREPARE_ONLY", Reason: "USE_EXISTING_TASK_PREPARATION"},
			{Kind: "QUEUE_NEXT", Availability: "UNSUPPORTED", Reason: "NO_DURABLE_OWNER_QUEUE"},
			{Kind: "STEER", Availability: "UNSUPPORTED", Reason: "NO_VERIFIED_TURN_STEERING"},
			{Kind: "WAIT", Availability: "OBSERVE_ONLY", Reason: "REFRESH_OBSERVED_STATE"},
			{Kind: "CANCEL", Availability: "UNSUPPORTED", Reason: "TURN_CANCEL_NOT_VERIFIED"},
		},
	}
	if gen == nil {
		return view
	}
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
	endpoint := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/agents/"), "/")
	if len(endpoint) != 2 || endpoint[0] == "" || endpoint[1] != "director" {
		writeError(w, http.StatusNotFound, "invalid Director agent path")
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
	recent, err := st.ListEventsMetadata(agent.ProjectID, agent.ID, 12)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "unable to query Director activity")
		return
	}
	view.RecentEvents = projectDirectorEvents(recent)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, view)
}

// Runtime IDs and stored generations are not proof that a live process still
// belongs to this Agent. Check the current registry binding before claiming
// liveness, including after a restart or concurrent runtime replacement.
func directorRuntimeMatches(agent store.Agent, gen store.RuntimeGeneration, session registry.RuntimeSession) bool {
	return session.RuntimeID == gen.RuntimeID &&
		session.AgentID == agent.ID &&
		session.ProviderID == gen.Provider
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
	var bound *registry.RuntimeSession
	if generation != nil && generation.RuntimeID != "" {
		if session, exists := h.nexus.RuntimeRegistry().Get(generation.RuntimeID); exists &&
			directorRuntimeMatches(agent, *generation, session) {
			bound = &session
			live = session.HostLive()
		}
	}
	view := projectDirectorAgent(agent, generation, live)
	observed := projectAgentObservation(agent.ID, agent.ProjectID, generation, bound, live)
	view.ObservedState = observed.AgentState
	view.ObservedSource = observed.SignalSource
	view.ObservedReason = observed.ReasonCode
	view.Quota = projectDirectorQuota(view.Provider, view.Profile)
	if bound != nil {
		view.Workspace = bound.Workspace
	}
	revisions, err := st.ListRevisions(agent.ID)
	if err != nil {
		return directorProjection{}, err
	}
	isolation, declaredWorkspace := agentRevisionWorkspace(revisions)
	if declaredWorkspace != "" {
		view.Workspace = declaredWorkspace
	}
	if isolation == "worktree" && view.Workspace != "" {
		view.Worktree = view.Workspace
	}
	lineage, err := st.ListLineage(agent.ID)
	if err != nil {
		return directorProjection{}, err
	}
	view.Lineage = projectDirectorLineage(lineage)
	return view, nil
}
