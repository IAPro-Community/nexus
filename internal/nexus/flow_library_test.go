package nexus

import (
	"context"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

func TestCloneFlowToProjectClearsAgentBindingsAndPreservesLeaderSuggestion(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	from, _ := st.CreateProject(store.Project{Name: "From", CanonicalPath: t.TempDir()})
	to, _ := st.CreateProject(store.Project{Name: "To", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: from.ID, Name: "Lead", Role: "architect"})
	plan, err := n.CreateWorkPlan(context.Background(), from.ID, "Reusable", "", []store.PlanPhase{{ID: "phase", Title: "Build", Order: 1, Packages: []store.WorkPackage{{ID: "step", Title: "Build", Goal: "Build it", Priority: "HIGH", Status: "READY", Role: "implementer", AssignmentStrategy: "EXISTING", AgentAllocation: agent.ID}}}}, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := n.SetFlowLeaderPolicy(context.Background(), plan.ID, FlowLeaderPolicy{Role: "architect", PreferredAgentID: agent.ID, Strategy: "EXISTING", Why: "coordinates architecture"}); err != nil {
		t.Fatal(err)
	}
	clone, err := n.CloneFlowToProject(context.Background(), plan.ID, to.ID)
	if err != nil {
		t.Fatal(err)
	}
	if clone.ProjectID != to.ID || clone.ID == plan.ID {
		t.Fatalf("invalid clone: %+v", clone)
	}
	if clone.Phases[0].Packages[0].AgentAllocation != "" || clone.Phases[0].Packages[0].AssignmentStrategy != "AUTO" {
		t.Fatalf("project binding survived clone: %+v", clone.Phases[0].Packages[0])
	}
	policy, err := n.GetFlowLeaderPolicy(context.Background(), clone.ID)
	if err != nil {
		t.Fatal(err)
	}
	if policy.Role != "architect" || policy.PreferredAgentID != "" || policy.Strategy != "AUTO" {
		t.Fatalf("leader policy not made portable: %+v", policy)
	}
}

func TestGetFlowLeaderPolicyReturnsValidDefaultForUnconfiguredPlan(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	project, _ := st.CreateProject(store.Project{Name: "Leader default", CanonicalPath: t.TempDir()})
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Unconfigured", "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	policy, err := n.GetFlowLeaderPolicy(context.Background(), plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if policy.Role != "orchestrator" || policy.Strategy != "AUTO" {
		t.Fatalf("invalid default leader policy: %+v", policy)
	}
}

func TestSetFlowLeaderPolicyAllowsExplicitlyNoLeader(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	project, _ := st.CreateProject(store.Project{Name: "No leader", CanonicalPath: t.TempDir()})
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "No leader flow", "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := n.SetFlowLeaderPolicy(context.Background(), plan.ID, FlowLeaderPolicy{Strategy: "NONE"}); err != nil {
		t.Fatal(err)
	}
	policy, err := n.GetFlowLeaderPolicy(context.Background(), plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if policy.Strategy != "NONE" || policy.PreferredAgentID != "" {
		t.Fatalf("invalid no-leader policy: %+v", policy)
	}
}
