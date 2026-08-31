package nexus

import (
	"fmt"

	"github.com/kivervinicius/ai-cli/internal/nexus/plangraph"
	"github.com/kivervinicius/ai-cli/internal/nexus/runner"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

// FlowDefinition is the V0 user-facing name for the durable WorkPlan. The
// embedded compatibility value keeps persistence and IDs unchanged.
type FlowDefinition struct{ store.WorkPlan }

// FlowStep and FlowRevision are compatibility aliases, not duplicate domain
// models. Existing callers can continue using WorkPackage/PlanRevision.
type FlowStep = store.WorkPackage
type FlowRevision = store.PlanRevision

func FlowFromWorkPlan(plan store.WorkPlan) FlowDefinition { return FlowDefinition{WorkPlan: plan} }

func (f FlowDefinition) ToWorkPlan() store.WorkPlan { return f.WorkPlan }

// ValidateFlow validates the static graph before a draft can be executed.
func ValidateFlow(plan store.WorkPlan) error {
	if plan.ID == "" || plan.ProjectID == "" {
		return fmt.Errorf("flow id and project id are required")
	}
	var nodes []plangraph.Node
	for _, phase := range plan.Phases {
		for _, step := range phase.Packages {
			nodes = append(nodes, plangraph.Node{ID: step.ID, Title: step.Title, Dependencies: step.Dependencies})
		}
	}
	_, err := plangraph.NormalizeAndValidate(nodes)
	return err
}

// FlowRunStatus maps durable runner states to the deliberately smaller V0
// vocabulary. Unknown states fail closed rather than appearing completed.
func FlowRunStatus(state runner.State) (string, bool) {
	switch state {
	case runner.StatePending:
		return "QUEUED", true
	case runner.StateReady, runner.StateAllocating, runner.StateCompiling:
		return "READY", true
	case runner.StateExecuting, runner.StateTesting, runner.StateReviewing, runner.StateRemediating:
		return "RUNNING", true
	case runner.StateVerifying:
		return "VERIFYING", true
	case runner.StateCompletedVerified:
		return "COMPLETED", true
	case runner.StatePaused, runner.StateBlockedNeedsUser, runner.StateEscalated:
		return "BLOCKED", true
	case runner.StateFailed, runner.StateFailedNoProgress, runner.StateFailedBudgetExceeded, runner.StateFailedVerification:
		return "FAILED", true
	case runner.StateCanceledByUser:
		return "CANCELED", true
	default:
		return "", false
	}
}
