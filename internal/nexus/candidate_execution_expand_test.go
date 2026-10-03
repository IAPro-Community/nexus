package nexus

import (
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

func TestExpandCandidatePlanSpecPreservesOriginalDependencyThroughCoordinator(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_CANDIDATE_RUNS", "1")
	t.Setenv("NEXUS_ENABLE_EXECUTION_WORKSPACES_V2", "1")

	spec := runner.PlanSpec{
		ID: "plan", ProjectID: "project", Revision: 7, Autonomous: true,
		Packages: []runner.PackageSpec{
			{ID: "build", Title: "Build", Goal: "Implement", AssignmentStrategy: string(FlowAssignmentAuto)},
			{ID: "verify", Title: "Verify", Goal: "Verify", Dependencies: []string{"build"}, AssignmentStrategy: string(FlowAssignmentAuto)},
		},
	}
	preview := &PlanExecutionPreview{
		PlanID: "plan", ProjectID: "project", Revision: 7,
		Steps: []StepExecutionPreview{
			{
				PackageID: "build",
				CandidateExecution: CandidateExecutionDecision{Enabled: true, CandidateCount: 2, RequiresIndependentEvaluation: true},
			},
			{
				PackageID: "verify",
				CandidateExecution: CandidateExecutionDecision{CandidateCount: 1},
			},
		},
	}

	expanded, err := expandCandidatePlanSpec(spec, preview)
	if err != nil {
		t.Fatal(err)
	}
	if len(expanded.Packages) != 4 {
		t.Fatalf("expanded package count=%d, want 4", len(expanded.Packages))
	}
	first, second, coordinator, downstream := expanded.Packages[0], expanded.Packages[1], expanded.Packages[2], expanded.Packages[3]
	if first.ID != "build__candidate_1" || second.ID != "build__candidate_2" {
		t.Fatalf("candidate ids=%q,%q", first.ID, second.ID)
	}
	if first.ParallelGroup != "candidate:build" || second.ParallelGroup != "candidate:build" {
		t.Fatalf("candidate parallel group was not preserved")
	}
	if !first.ReviewRequired || !second.ReviewRequired {
		t.Fatal("candidate implementations must require independent review")
	}
	if !coordinator.CandidateCoordinator || coordinator.ID != "build" {
		t.Fatalf("coordinator did not preserve original package identity: %+v", coordinator)
	}
	if len(coordinator.Dependencies) != 2 || coordinator.Dependencies[0] != first.ID || coordinator.Dependencies[1] != second.ID {
		t.Fatalf("coordinator dependencies=%v", coordinator.Dependencies)
	}
	if len(downstream.Dependencies) != 1 || downstream.Dependencies[0] != "build" {
		t.Fatalf("downstream package must continue depending on original id, got %v", downstream.Dependencies)
	}
}

func TestExpandCandidatePlanSpecRequiresTaskScopedWorkspaces(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_CANDIDATE_RUNS", "1")
	t.Setenv("NEXUS_DISABLE_EXECUTION_WORKSPACES_V2", "1")
	spec := runner.PlanSpec{ID: "plan", ProjectID: "project", Revision: 1, Packages: []runner.PackageSpec{{ID: "build"}}}
	preview := &PlanExecutionPreview{PlanID: "plan", Revision: 1, Steps: []StepExecutionPreview{{
		PackageID: "build", CandidateExecution: CandidateExecutionDecision{Enabled: true, CandidateCount: 2},
	}}}
	_, err := expandCandidatePlanSpec(spec, preview)
	if err == nil || !strings.Contains(err.Error(), "execution_workspaces_v2") {
		t.Fatalf("candidate expansion without task workspace isolation must fail closed, got %v", err)
	}
}

func TestConstrainCandidateExecutionRespectsPinnedAgent(t *testing.T) {
	decision := constrainCandidateExecution(
		CandidateExecutionDecision{Enabled: true, CandidateCount: 3, RequiresIndependentEvaluation: true},
		string(FlowAssignmentExisting),
		"agent-fixed",
		true,
	)
	if decision.Enabled || decision.CandidateCount != 1 {
		t.Fatalf("explicit Agent assignment must remain single execution: %+v", decision)
	}
}
