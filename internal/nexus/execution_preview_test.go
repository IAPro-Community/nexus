package nexus

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestPreviewPlanExecutionUsesExistingMatcherAndResourceSchedulerWithoutMutation(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Composer preview", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Frontend specialist", Role: "frontend-engineer"})
	if err != nil {
		t.Fatal(err)
	}

	requirements := TaskRequirements{
		TaskKind:             "coding",
		Role:                 "frontend-engineer",
		PreferredRoles:       []string{"frontend-engineer"},
		AcceptableRoles:      []string{"frontend-engineer", "fullstack-engineer"},
		RequiredCapabilities: []string{"headless", "submit_prompt"},
		Confidence:           "CLASSIFIED",
		Source:               "composer",
	}
	rawRequirements, err := json.Marshal(requirements)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Improve login", "", []store.PlanPhase{{
		ID: "phase-build", Title: "Build", Order: 1,
		Packages: []store.WorkPackage{{
			ID: "frontend", Title: "Frontend", Goal: "Improve the React login flow", Status: "READY",
			Role: "implementer", TaskRequirements: string(rawRequirements), AssignmentStrategy: "AUTO",
			ResourcePolicy: "BALANCED", AcceptanceCriteria: []string{"login flow works"},
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}

	accounts := []ProviderAccount{
		{
			ID: "codex-main", Provider: "codex", Profile: "main", DisplayName: "Codex main",
			Authenticated: true, Available: true, Health: "healthy",
			QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.80,
			Capabilities: map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED"},
		},
		{
			ID: "claude-low", Provider: "claude", Profile: "default", DisplayName: "Claude low",
			Authenticated: true, Available: true, Health: "healthy",
			QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.25,
			Capabilities: map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED"},
		},
	}

	beforeAgents, err := st.ListAgents(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := n.previewPlanExecution(context.Background(), st, plan, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	afterAgents, err := st.ListAgents(project.ID)
	if err != nil {
		t.Fatal(err)
	}

	if !preview.Ready || len(preview.Steps) != 1 {
		t.Fatalf("preview=%+v", preview)
	}
	step := preview.Steps[0]
	if step.Agent.AgentID != agent.ID || step.Agent.WillCreate {
		t.Fatalf("agent preview=%+v", step.Agent)
	}
	if step.Resource.Provider != "codex" || step.Resource.Profile != "main" {
		t.Fatalf("resource preview=%+v", step.Resource)
	}
	if step.Resource.Freshness != "LIVE" {
		t.Fatalf("freshness=%q, want LIVE", step.Resource.Freshness)
	}
	if len(beforeAgents) != len(afterAgents) {
		t.Fatalf("execution preview mutated Agent count: before=%d after=%d", len(beforeAgents), len(afterAgents))
	}
}

func TestPreviewPlanExecutionExplainsSpecialistCreationWithoutStartingAgent(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Composer specialist preview", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	requirements := TaskRequirements{
		TaskKind:             "review",
		Role:                 "reviewer",
		PreferredRoles:       []string{"reviewer"},
		AcceptableRoles:      []string{"reviewer"},
		RequiredCapabilities: []string{"headless", "submit_prompt"},
	}
	rawRequirements, _ := json.Marshal(requirements)
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Review", "", []store.PlanPhase{{
		ID: "phase-review", Title: "Review", Order: 1,
		Packages: []store.WorkPackage{{
			ID: "review", Title: "Independent review", Goal: "Review the change", Status: "READY",
			Role: "reviewer", TaskRequirements: string(rawRequirements), AssignmentStrategy: "AUTO",
			ResourcePolicy: "BALANCED", AcceptanceCriteria: []string{"review completed"},
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	accounts := []ProviderAccount{{
		ID: "codex-main", Provider: "codex", Profile: "main", DisplayName: "Codex main",
		Authenticated: true, Available: true, Health: "healthy",
		QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.70,
		Capabilities: map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED"},
	}}

	preview, err := n.previewPlanExecution(context.Background(), st, plan, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Ready || len(preview.Steps) != 1 {
		t.Fatalf("preview=%+v", preview)
	}
	if !preview.Steps[0].Agent.WillCreate {
		t.Fatalf("expected preview to explain specialist creation, got %+v", preview.Steps[0].Agent)
	}
	agents, err := st.ListAgents(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 0 {
		t.Fatalf("preview must not create specialist Agent, got %d", len(agents))
	}
}

func TestPreviewPlanExecutionBlocksWhenNoEligibleResourceExists(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Composer blocked preview", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Blocked", "", []store.PlanPhase{{
		ID: "phase", Title: "Build", Order: 1,
		Packages: []store.WorkPackage{{
			ID: "step", Title: "Build", Goal: "Build safely", Status: "READY", Role: "implementer",
			AssignmentStrategy: "AUTO", ResourcePolicy: "BALANCED",
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	accounts := []ProviderAccount{{
		ID: "unknown", Provider: "codex", Profile: "main", DisplayName: "Codex unknown",
		Authenticated: true, Available: true, Health: "healthy",
		QuotaView: &quota.QuotaView{Status: "UNKNOWN"},
		Capabilities: map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED"},
	}}

	preview, err := n.previewPlanExecution(context.Background(), st, plan, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Ready || len(preview.Blockers) == 0 {
		t.Fatalf("UNKNOWN-only resource pool must block automatic preview: %+v", preview)
	}
}
