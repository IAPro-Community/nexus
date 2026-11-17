package web

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/contextsnapshot"
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
			{State: "CANCELED_BY_USER"},
			{State: "FAILED_NO_PROGRESS"},
			{State: "FAILED_BUDGET_EXCEEDED"},
			{State: "FAILED_VERIFICATION"},
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

func TestNexProjectFactsAreBoundedAndAllowlisted(t *testing.T) {
	facts := []contextsnapshot.ProjectFact{
		{Category: "untrusted", Key: "ignore", Value: "do not include"},
		{Category: "commands", Key: "private.script", Value: "SECRET=not-a-key"},
	}
	for i := 0; i < 30; i++ {
		facts = append(facts, contextsnapshot.ProjectFact{
			Category: "stack", Key: fmt.Sprintf("stack.%d", i),
			Value:      strings.Repeat("a", 400),
			Basis:      contextsnapshot.FactObserved,
			Confidence: contextsnapshot.ConfidenceHigh,
		})
	}
	view := &nexus.ProjectIntelligenceView{
		CurrentSnapshot: &contextsnapshot.ProjectContextSnapshot{Facts: facts},
	}
	extracted := boundedNexProjectFacts(view)
	if len(extracted) != 12 {
		t.Fatalf("expected a bounded sample of 12 project facts, got %d", len(extracted))
	}
	for _, fact := range extracted {
		if fact.Category != "stack" || len(fact.Value) > 170 {
			t.Fatalf("project facts leaked an unexpected category or length: %+v", fact)
		}
	}
	if got := boundedNexProjectFacts(nil); len(got) != 0 {
		t.Fatal("missing snapshot must not fabricate context")
	}
}
