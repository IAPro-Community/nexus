package nexus

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestMatchAgentsUsesHardGatesAndDeterministicRanking(t *testing.T) {
	candidates := []AgentMatchCandidate{
		{Agent: store.Agent{ID: "disabled", Name: "Disabled", Role: "frontend-engineer", Status: store.AgentWorking}, Spec: intelligence.AgentSpec{Role: "frontend-engineer", Capabilities: []string{"react"}}},
		{Agent: store.Agent{ID: "backend", Name: "Backend", Role: "backend-engineer", Status: store.AgentStopped}, Spec: intelligence.AgentSpec{Role: "backend-engineer", Capabilities: []string{"go"}}},
		{Agent: store.Agent{ID: "frontend", Name: "Frontend", Role: "frontend", Status: store.AgentStopped}, Spec: intelligence.AgentSpec{Role: "frontend-engineer", Domains: []string{"web"}, Capabilities: []string{"react"}, Strengths: []string{"accessibility"}}},
	}
	result := MatchAgents(candidates, AgentMatchRequirements{PreferredRoles: []string{"frontend-engineer"}, AcceptableRoles: []string{"frontend-engineer", "fullstack-engineer"}, RequiredCapabilities: []string{"react"}, Domains: []string{"web"}, DesiredStrengths: []string{"accessibility"}})
	if result.Recommended == nil || result.Recommended.Agent.ID != "frontend" {
		t.Fatalf("recommended=%+v", result.Recommended)
	}
	for _, candidate := range result.Candidates {
		if candidate.Agent.ID == "disabled" && candidate.Eligible {
			t.Fatal("working agent must be rejected")
		}
		if candidate.Agent.ID == "backend" && candidate.Eligible {
			t.Fatal("missing capability must be rejected")
		}
	}
}

func TestNormalizeAgentTaxonomy(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"frontend", "frontend-engineer"},
		{"front-end", "frontend-engineer"},
		{"frontend_dev", "frontend-engineer"},
		{"backend", "backend-engineer"},
		{"reviewer", "reviewer"},
		{"security", "security-reviewer"},
		{"security_auditor", "security-reviewer"},
		{"software architect", "architect"},
		{"root_cause_investigator", "investigator"},
		{"performance engineer", "performance-investigator"},
		{"senior engineer", "senior-developer"},
		{"diagnostician", "investigator"},
		{"verifier", "qa"},
	} {
		if got := normalizeAgentTaxonomy(test.input); got != test.want {
			t.Errorf("normalizeAgentTaxonomy(%q)=%q, want %q", test.input, got, test.want)
		}
	}
}

func TestAgentRequiredCapabilitiesExcludeResourceGates(t *testing.T) {
	got := agentRequiredCapabilities([]string{"headless", "submit_prompt", "react"})
	if len(got) != 1 || got[0] != "react" {
		t.Fatalf("agent capabilities=%v, want [react]", got)
	}
}
