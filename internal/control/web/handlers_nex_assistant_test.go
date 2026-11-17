package web

import (
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

func TestNexAssistantOverviewProjectsExistingHUDFacts(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	expired := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	view := buildNexAssistantOverview(nexus.HUDState{
		GeneratedAt: now,
		Agents: []nexus.HUDAgent{
			{Status: "WORKING"},
			{Status: "IDLE"},
		},
		Missions: []nexus.HUDMission{
			{State: "BLOCKED_NEEDS_USER", NeedsUser: true},
			{State: "COMPLETED_VERIFIED"},
		},
		Alerts: []nexus.HUDAlert{
			{Severity: "WARNING", SnoozedUntil: &future},
			{Severity: "WARNING", SnoozedUntil: &expired},
			{Severity: "CRITICAL", Acked: true},
		},
	}, now)
	if view.AgentCount != 1 || view.RunCount != 1 || view.NeedsUser != 1 || view.Alerts != 1 {
		t.Fatalf("unexpected projection: %+v", view)
	}
	if len(view.Suggestions) != 3 {
		t.Fatalf("expected three factual navigation options, got %+v", view.Suggestions)
	}
}

func TestNexAssistantDoesNotInventSuggestions(t *testing.T) {
	view := buildNexAssistantOverview(nexus.HUDState{}, time.Now())
	if len(view.Suggestions) != 0 || view.AgentCount != 0 || view.NeedsUser != 0 {
		t.Fatalf("empty snapshot must not create recommendations: %+v", view)
	}
	if _, ok := nexDeterministicReply("analise meu código", view); ok {
		t.Fatal("do not pretend code analysis without model/tool access")
	}
	answer, ok := nexDeterministicReply("status", view)
	if !ok || !strings.Contains(answer, "0 agentes ativos") {
		t.Fatalf("status must reflect actual snapshot: %q", answer)
	}
}
