package nexus

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestDecomposePromptIntoFlowProposal_SimpleTask(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "DecompSimple", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	proposal, err := n.DecomposePromptIntoFlowProposal(context.Background(), FlowDecompositionRequest{
		ProjectID: project.ID,
		Goal:      "corrija a UI React",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(proposal.Flow.Steps) != 1 {
		t.Fatalf("expected 1 atomic step for simple task, got %d", len(proposal.Flow.Steps))
	}
	if proposal.Flow.Steps[0].Role != "implementer" {
		t.Fatalf("expected generic execution role, got %s", proposal.Flow.Steps[0].Role)
	}
	if proposal.Flow.Steps[0].TaskRequirements == "" {
		t.Fatal("atomic task must persist classified TaskRequirements for routing")
	}
	var requirements TaskRequirements
	if err := json.Unmarshal([]byte(proposal.Flow.Steps[0].TaskRequirements), &requirements); err != nil {
		t.Fatal(err)
	}
	if requirements.Role != "frontend-engineer" {
		t.Fatalf("classified role=%q, want frontend-engineer", requirements.Role)
	}
}

func TestDecomposePromptIntoFlowProposal_UsesCompleteSourcePromptForAtomicity(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "DecompComposer", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	proposal, err := n.DecomposePromptIntoFlowProposal(context.Background(), FlowDecompositionRequest{
		ProjectID:      project.ID,
		Goal:           "Implement Google OAuth login safely",
		DelegationMode: DelegationAuto,
		SourcePrompt: `Implement Google OAuth login safely.
Backend: implement the Go callback and session contract.
Frontend: implement the React sign-in and session handling.
Testing: add backend, frontend and browser regression tests.
Independent review must validate security and regressions.`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(proposal.Flow.Steps); got < 4 {
		t.Fatalf("short display goal must not collapse a complete multi-domain prompt: got %d steps", got)
	}
	if err := ValidateFlowDAG(proposal.Flow); err != nil {
		t.Fatalf("proposal DAG is invalid: %v", err)
	}
}

func TestDecomposePromptIntoFlowProposal_ComplexFeature(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "DecompComplex", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	proposal, err := n.DecomposePromptIntoFlowProposal(context.Background(), FlowDecompositionRequest{
		ProjectID:      project.ID,
		Goal:           "Build a complete user authentication system with session management and JWT tokens",
		DelegationMode: DelegationAuto,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(proposal.Flow.Steps) < 2 {
		t.Fatalf("expected at least 2 steps for complex feature, got %d", len(proposal.Flow.Steps))
	}
	if proposal.Flow.Steps[0].TaskRequirements == "" {
		t.Fatal("complex task must persist classified TaskRequirements for routing")
	}
	// Verify DAG order
	if err := ValidateFlowDAG(proposal.Flow); err != nil {
		t.Fatalf("proposal DAG is invalid: %v", err)
	}
}

func TestPreflightFlow_PassesOnValidFlow(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "PreflightProj", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Valid Plan", "Testing preflight", []store.PlanPhase{
		{
			ID: "p1", Title: "Phase 1", Order: 1,
			Packages: []store.WorkPackage{
				{ID: "pkg1", Title: "Task 1", Status: "READY", Role: "implementer"},
			},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	report, err := n.PreflightFlow(context.Background(), plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Ready {
		t.Fatalf("expected ready report, got ready=false: %+v", report)
	}
	if len(report.Checks) < 3 {
		t.Fatalf("expected at least 3 checks, got %d", len(report.Checks))
	}
	for _, ch := range report.Checks {
		if ch.Key == "dag_validity" && ch.Status != "PASS" {
			t.Fatalf("dag_validity should PASS, got %s", ch.Status)
		}
	}
}

func TestPreflightFlow_RejectsCyclicDAG(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "CyclicProj", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	// Create a plan with a circular dependency between pkg1 and pkg2
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Cyclic Plan", "Testing circular DAG preflight", []store.PlanPhase{
		{
			ID: "p1", Title: "Phase 1", Order: 1,
			Packages: []store.WorkPackage{
				{ID: "pkg1", Title: "Task 1", Status: "READY", Role: "implementer", Dependencies: []string{"pkg2"}},
				{ID: "pkg2", Title: "Task 2", Status: "READY", Role: "implementer", Dependencies: []string{"pkg1"}},
			},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	report, err := n.PreflightFlow(context.Background(), plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready {
		t.Fatalf("expected ready=false for cyclic plan, got ready=true")
	}

	// Attempting to start the mission run must fail closed
	_, err = n.StartMissionRun(context.Background(), plan.ID, "", runner.AutonomyContract{}, false)
	if err == nil {
		t.Fatalf("expected StartMissionRun to fail closed on cyclic DAG, but it succeeded")
	}
}
