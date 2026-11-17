package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// agentObservation is a redacted, read-only explanation of the current runtime.
// It is NOT a Mission completion proof, a prompt transcript, or an approval API.
type agentObservation struct {
	SchemaVersion     int    `json:"schema_version"`
	AgentID           string `json:"agent_id"`
	ProjectID         string `json:"project_id"`
	RuntimeID         string `json:"runtime_id,omitempty"`
	GenerationID      string `json:"generation_id,omitempty"`
	Provider          string `json:"provider,omitempty"`
	ProcessState      string `json:"process_state"`
	RuntimeState      string `json:"runtime_state,omitempty"`
	AgentState        string `json:"agent_state"`
	SignalSource      string `json:"signal_source"`
	ReasonCode        string `json:"reason_code"`
	AttentionRevision uint64 `json:"attention_revision"`
	ObservedAt        string `json:"observed_at,omitempty"`
}

// projectAgentObservation never infers a completed task from an idle terminal,
// an exited process, or textual "done". It only projects the existing registry.
func projectAgentObservation(agentID, projectID string, gen *store.RuntimeGeneration, sess *registry.RuntimeSession, live bool) agentObservation {
	view := agentObservation{
		SchemaVersion: 1, AgentID: agentID, ProjectID: projectID,
		ProcessState: "UNKNOWN", AgentState: "unknown", SignalSource: "UNKNOWN",
		ReasonCode: "NO_CURRENT_GENERATION",
	}
	if gen == nil {
		return view
	}
	view.RuntimeID = gen.RuntimeID
	view.GenerationID = gen.ID
	view.Provider = gen.Provider
	view.ReasonCode = "NO_MATCHING_RUNTIME"
	if sess == nil {
		return view
	}
	view.RuntimeState = string(sess.State)
	view.AttentionRevision = sess.AttentionRevision
	if !sess.UpdatedAt.IsZero() {
		view.ObservedAt = sess.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	if !live {
		view.ProcessState = "NOT_LIVE"
		view.ReasonCode = "SESSION_HOST_NOT_LIVE"
		return view
	}
	view.ProcessState = "LIVE"
	if sess.State == registry.StateDetached || sess.State == registry.StateHandoff || sess.State == registry.StateStarting {
		view.ReasonCode = "NO_CURRENT_TURN_EVIDENCE"
		return view
	}
	if sess.AttentionGeneration != "" &&
		sess.AttentionGeneration != fmt.Sprintf("%d:%d", sess.HostGeneration, sess.ProviderGeneration) {
		view.ReasonCode = "STALE_ATTENTION_GENERATION"
		return view
	}
	// A capability (ControlLevel) is not provenance. Only the event's explicit
	// recorded source can support confidence claims.
	switch sess.AttentionSource {
	case "AUTHORITATIVE", "SIGNALLED", "INFERRED":
		view.SignalSource = sess.AttentionSource
	}
	switch strings.ToLower(sess.AttentionKind) {
	case "needs_user":
		view.AgentState, view.ReasonCode = "blocked", "OBSERVED_NEEDS_USER"
	case "working":
		view.AgentState, view.ReasonCode = "working", "OBSERVED_WORKING"
	case "idle":
		view.AgentState, view.ReasonCode = "idle", "OBSERVED_IDLE"
	case "completed":
		view.AgentState, view.ReasonCode = "done", "OBSERVED_TURN_SETTLED_NOT_VERIFIED"
	case "error":
		view.AgentState, view.ReasonCode = "error", "OBSERVED_ERROR"
	default:
		view.ReasonCode = "NO_CLASSIFIABLE_AGENT_EVENT"
	}
	// Terminal state is not evidence for a fresh conversation turn.
	if sess.State == registry.StateStopped || sess.State == registry.StateFailed || sess.State == registry.StateStale {
		view.AgentState = "unknown"
		view.ReasonCode = "RUNTIME_NOT_EXECUTING"
	}
	return view
}

// observeAgent checks the existing project and generation ownership. It never
// searches other projects or selects a stale sibling from registry map order.
func (h *NexusHandler) observeAgent(agentID, projectID string) (agentObservation, int, error) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		return agentObservation{}, http.StatusServiceUnavailable, err
	}
	agent, err := st.GetAgent(agentID, projectID)
	if err != nil {
		return agentObservation{}, http.StatusNotFound, errors.New("agent not found in project")
	}
	gen, err := st.CurrentGeneration(agent.ID)
	if errors.Is(err, store.ErrNotFound) {
		return projectAgentObservation(agent.ID, agent.ProjectID, nil, nil, false), http.StatusOK, nil
	}
	if err != nil {
		return agentObservation{}, http.StatusServiceUnavailable, err
	}
	sess, ok := h.nexus.RuntimeRegistry().Get(gen.RuntimeID)
	if !ok || !directorRuntimeMatches(agent, gen, sess) {
		return projectAgentObservation(agent.ID, agent.ProjectID, &gen, nil, false), http.StatusOK, nil
	}
	return projectAgentObservation(agent.ID, agent.ProjectID, &gen, &sess, sess.HostLive()), http.StatusOK, nil
}

// evaluatedAgentWait is tied to the current persistent Agent generation. A
// changed runtime generation invalidates a prior observation cursor even when
// two sessions happen to reuse the same attention revision counter.
func evaluatedAgentWait(view agentObservation, until string, afterRevision uint64, afterGeneration string) (matched, generationChanged bool) {
	generationChanged = afterGeneration != "" && view.GenerationID != afterGeneration
	matched = !generationChanged && view.ProcessState == "LIVE" &&
		view.AgentState == until && view.AttentionRevision > afterRevision
	return matched, generationChanged
}

// GET /api/v1/agents/{id}/explain?project_id={id}
// GET /api/v1/agents/{id}/wait?project_id={id}&until=blocked&wait_ms=5000
// Wait is bounded observation, not provider prompt/turn completion or Mission wait.
func (h *NexusHandler) handleAgentObservation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "agent observation is read-only")
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/agents/"), "/"), "/")
	if len(parts) != 2 || parts[0] == "" || (parts[1] != "explain" && parts[1] != "wait") {
		writeError(w, http.StatusNotFound, "invalid agent observation path")
		return
	}
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "project_id is required")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	until := ""
	var afterRevision uint64
	afterGeneration := ""
	waitFor := time.Duration(0)
	if parts[1] == "wait" {
		until = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("until")))
		switch until {
		case "blocked", "working", "idle", "done", "error":
		default:
			writeError(w, http.StatusBadRequest, "until must be blocked, working, idle, done or error")
			return
		}
		if raw := r.URL.Query().Get("after_revision"); raw != "" {
			var parseErr error
			afterRevision, parseErr = strconv.ParseUint(raw, 10, 64)
			if parseErr != nil {
				writeError(w, http.StatusBadRequest, "invalid after_revision")
				return
			}
		}
		afterGeneration = strings.TrimSpace(r.URL.Query().Get("after_generation"))
		if len(afterGeneration) > 160 {
			writeError(w, http.StatusBadRequest, "after_generation is too long")
			return
		}
		if afterRevision > 0 && afterGeneration == "" {
			writeError(w, http.StatusBadRequest, "after_generation required with after_revision")
			return
		}
		if raw := r.URL.Query().Get("wait_ms"); raw != "" {
			millis, parseErr := strconv.Atoi(raw)
			if parseErr != nil || millis < 0 || millis > 15000 {
				writeError(w, http.StatusBadRequest, "wait_ms must be between 0 and 15000")
				return
			}
			waitFor = time.Duration(millis) * time.Millisecond
		}
	}
	deadline := time.Now().Add(waitFor)
	for {
		view, status, err := h.observeAgent(parts[0], projectID)
		if err != nil {
			writeError(w, status, err.Error())
			return
		}
		if parts[1] == "explain" {
			writeJSON(w, http.StatusOK, view)
			return
		}
		matched, generationChanged := evaluatedAgentWait(view, until, afterRevision, afterGeneration)
		if matched || generationChanged || view.ProcessState != "LIVE" || !time.Now().Before(deadline) {
			writeJSON(w, http.StatusOK, map[string]any{
				"matched": matched,
				"generation_changed": generationChanged,
				"timed_out": !matched && !generationChanged && view.ProcessState == "LIVE",
				"observation": view,
			})
			return
		}
		timer := time.NewTimer(200 * time.Millisecond)
		select {
		case <-r.Context().Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
