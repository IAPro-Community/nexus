package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestAgentObservationHonorsEvidenceAndNeverPromotesTurnToMission(t *testing.T) {
	gen := &store.RuntimeGeneration{ID: "gen-1", RuntimeID: "runtime-1", Provider: "opencode"}
	sess := &registry.RuntimeSession{
		RuntimeID: "runtime-1", AgentID: "agent-1", ProviderID: "opencode",
		State: registry.StateRunning, ControlLevel: registry.ControlLevelEvents,
		AttentionKind: "completed", AttentionRevision: 8,
		AttentionContext: "PRIVATE_PROMPT", Env: []string{"SECRET=PRIVATE_KEY"},
		UpdatedAt: time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC),
	}
	view := projectAgentObservation("agent-1", "project-1", gen, sess, true)
	if view.AgentState != "done" || view.ReasonCode != "OBSERVED_TURN_SETTLED_NOT_VERIFIED" ||
		view.SignalSource != "EVENTS" || view.AttentionRevision != 8 {
		t.Fatalf("unexpected observed turn: %+v", view)
	}
	encoded, err := json.Marshal(view)
	if err != nil { t.Fatal(err) }
	for _, sensitive := range []string{"PRIVATE_PROMPT", "PRIVATE_KEY", "account_scope", "COMPLETED_VERIFIED"} {
		if strings.Contains(string(encoded), sensitive) { t.Fatalf("sensitive or unverified claim: %s", encoded) }
	}
	sess.AttentionKind = "needs_user"
	if got := projectAgentObservation("agent-1", "project-1", gen, sess, true); got.AgentState != "blocked" {
		t.Fatalf("expected exact blocked state, got %+v", got)
	}
	if got := projectAgentObservation("agent-1", "project-1", gen, sess, false); got.AgentState != "unknown" || got.ProcessState != "NOT_LIVE" {
		t.Fatalf("dead host cannot advertise blocked agent: %+v", got)
	}
	sess.State = registry.StateDetached
	if got := projectAgentObservation("agent-1", "project-1", gen, sess, true); got.AgentState != "unknown" {
		t.Fatalf("detached runtime is not proof of fresh turn: %+v", got)
	}
	sess.State = registry.StateRunning
	sess.AttentionGeneration = "old-host:old-provider"
	if got := projectAgentObservation("agent-1", "project-1", gen, sess, true); got.ReasonCode != "STALE_ATTENTION_GENERATION" || got.AgentState != "unknown" {
		t.Fatalf("stale attention must not be trusted: %+v", got)
	}
}

func TestAgentObservationMissingGenerationIsUnknown(t *testing.T) {
	view := projectAgentObservation("agent-a", "project-a", nil, nil, false)
	if view.AgentState != "unknown" || view.ProcessState != "UNKNOWN" || view.RuntimeID != "" {
		t.Fatalf("missing generation must remain unknown: %+v", view)
	}
}

func TestAgentObservationRejectsMutationAndUnscopedWait(t *testing.T) {
	h := &NexusHandler{}
	call := func(method, path string) int {
		r := httptest.NewRecorder()
		h.handleAgentObservation(r, httptest.NewRequest(method, path, nil))
		return r.Code
	}
	if got := call(http.MethodPost, "/api/v1/agents/a/explain?project_id=p"); got != http.StatusMethodNotAllowed {
		t.Fatalf("mutation status = %d", got)
	}
	if got := call(http.MethodGet, "/api/v1/agents/a/explain"); got != http.StatusBadRequest {
		t.Fatalf("unscoped status = %d", got)
	}
	if got := call(http.MethodGet, "/api/v1/agents/a/wait?project_id=p&until=verified"); got != http.StatusBadRequest {
		t.Fatalf("unsupported completion claim = %d", got)
	}
	if got := call(http.MethodGet, "/api/v1/agents/a/wait?project_id=p&until=blocked&wait_ms=30000"); got != http.StatusBadRequest {
		t.Fatalf("unbounded wait status = %d", got)
	}
}
