package nexus

import (
	"testing"

	"github.com/kivervinicius/ai-cli/internal/nexus/runner"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

func TestFlowWorkPlanRoundTripPreservesIdentityAndExecutionFields(t *testing.T) {
	plan := store.WorkPlan{
		ID: "flow-1", ProjectID: "project-1", Title: "Release", CurrentRevision: 7,
		Phases: []store.PlanPhase{{ID: "phase-1", Packages: []store.WorkPackage{{
			ID: "a", Dependencies: []string{}, ParallelGroup: "g", AgentAllocation: "agent-1", AcceptanceCriteria: []string{"tests pass"},
		}}}},
	}
	got := FlowFromWorkPlan(plan).ToWorkPlan()
	if got.ID != plan.ID || got.ProjectID != plan.ProjectID || got.CurrentRevision != plan.CurrentRevision {
		t.Fatalf("identity/revision lost: %#v", got)
	}
	step := got.Phases[0].Packages[0]
	if step.ID != "a" || step.AgentAllocation != "agent-1" || step.ParallelGroup != "g" || step.AcceptanceCriteria[0] != "tests pass" {
		t.Fatalf("step fields lost: %#v", step)
	}
}

func TestFlowRunStatusFailsClosedForUnknownState(t *testing.T) {
	if got, ok := FlowRunStatus(runner.StateCompletedVerified); !ok || got != "COMPLETED" {
		t.Fatalf("completed=%q ok=%v", got, ok)
	}
	if got, ok := FlowRunStatus(runner.State("future-state")); ok || got != "" {
		t.Fatalf("unknown mapped to %q", got)
	}
}
