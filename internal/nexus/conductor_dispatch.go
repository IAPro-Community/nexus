package nexus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

// ─────────────────────────────────────────────────────────────────────
// § C.6  Conductor.Dispatch — single entrypoint for goal routing
// ─────────────────────────────────────────────────────────────────────

// DispatchRequest is the input for Conductor.Dispatch.
type DispatchRequest struct {
	ProjectID      string         `json:"project_id"`
	Goal           string         `json:"goal"`
	SourcePrompt   string         `json:"source_prompt"`
	LeadAgentID    string         `json:"lead_agent_id"`
	DelegationMode DelegationMode `json:"delegation_mode"`
	// GoalID is optional; when empty, a new one is generated.
	GoalID string `json:"goal_id,omitempty"`
	// AutoPromoteToMission confirms that an upstream policy boundary already
	// decided this request deserves durable Mission/Flow execution. It does not
	// imply multi-agent delegation.
	AutoPromoteToMission bool `json:"auto_promote_to_mission"`
	// PromotionSource and SourceRuntimeID are optional audit lineage. They are
	// persisted as WorkPlan facts and never participate in Agent identity.
	PromotionSource string `json:"promotion_source,omitempty"`
	SourceRuntimeID string `json:"source_runtime_id,omitempty"`
}

// DispatchResult is the output of Conductor.Dispatch.
type DispatchResult struct {
	GoalID           string             `json:"goal_id"`
	PlanID           string             `json:"plan_id,omitempty"`
	RunID            string             `json:"run_id"`
	Status           DispatchStatus     `json:"status"`
	Run              *runner.MissionRun `json:"run,omitempty"`
	AdmissionBlocked bool               `json:"admission_blocked,omitempty"`
	AdmissionStage   string             `json:"admission_stage,omitempty"`
	FallbackSafe     bool               `json:"fallback_safe,omitempty"`
	Error            string             `json:"error,omitempty"`
}

// DispatchStatus enumerates possible dispatch outcomes.
type DispatchStatus string

const (
	DispatchAccepted DispatchStatus = "ACCEPTED"
	DispatchRejected DispatchStatus = "REJECTED"
	DispatchFailed   DispatchStatus = "FAILED"
	// DispatchBlocked reports a mission waiting on human intervention.
	// It is non-terminal: the run is alive and can resume.
	DispatchBlocked DispatchStatus = "BLOCKED"
	// DispatchPaused reports a mission under manual takeover.
	// It is non-terminal: the run is alive and can resume.
	DispatchPaused DispatchStatus = "PAUSED"
	// DispatchCanceled reports a mission canceled by the user.
	DispatchCanceled  DispatchStatus = "CANCELED"
	DispatchCompleted DispatchStatus = "COMPLETED"
)

// dispatchStatusForRunState maps a mission run state to its honest dispatch
// outcome. PAUSED and BLOCKED_NEEDS_USER are non-terminal and must never be
// reported as COMPLETED.
func dispatchStatusForRunState(state runner.State) DispatchStatus {
	switch {
	case state == runner.StateBlockedNeedsUser:
		return DispatchBlocked
	case state == runner.StatePaused:
		return DispatchPaused
	case state == runner.StateCanceledByUser:
		return DispatchCanceled
	case runner.IsTerminalState(state) && (state == runner.StateCompletedVerified || state == runner.StateCompleted):
		return DispatchCompleted
	case runner.IsTerminalState(state):
		return DispatchFailed
	default:
		return DispatchFailed
	}
}

// StartDispatch runs the synchronous prefix of the Conductor pipeline:
// decompose → plan → start mission. It returns ACCEPTED with the run ID
// immediately so Web callers can answer POST with 202 Accepted and follow up
// via polling/stream instead of holding the request for the whole mission.
func (n *Nexus) StartDispatch(ctx context.Context, req DispatchRequest) (*DispatchResult, error) {
	if n == nil {
		return nil, fmt.Errorf("nexus is nil")
	}
	if req.ProjectID == "" || req.Goal == "" {
		return nil, fmt.Errorf("project_id and goal are required")
	}
	if req.DelegationMode == "" {
		req.DelegationMode = DelegationAuto
	}
	if req.GoalID == "" {
		req.GoalID = "goal_" + ids.NewULID()
	}

	// Step 1: Decompose prompt into flow proposal.
	proposal, err := n.DecomposePromptIntoFlowProposal(ctx, FlowDecompositionRequest{
		ProjectID:      req.ProjectID,
		Goal:           req.Goal,
		SourcePrompt:   req.SourcePrompt,
		DelegationMode: req.DelegationMode,
	})
	if err != nil {
		return &DispatchResult{
			GoalID:       req.GoalID,
			Status:       DispatchFailed,
			FallbackSafe: true,
			Error:        fmt.Sprintf("decompose prompt: %v", err),
		}, nil
	}

	// Step 2: Create work plan from flow.
	flowPlan := WorkPlanFromFlow(proposal.Flow)
	decision, err := ReadDelegationDecisionFacts(flowPlan.StructuredFacts)
	if err != nil {
		return &DispatchResult{
			GoalID:       req.GoalID,
			Status:       DispatchRejected,
			FallbackSafe: true,
			Error:        fmt.Sprintf("validate delegation decision: %v", err),
		}, nil
	}
	if !dispatchTopologyAllowed(req, decision) {
		return &DispatchResult{
			GoalID:       req.GoalID,
			Status:       DispatchRejected,
			FallbackSafe: true,
			Error:        "validate delegation decision: dispatch brake rejected the delegated topology",
		}, nil
	}

	// Step 3: Bind lead agent and freeze the user's goal contract into facts.
	decision = WithLeadAgent(decision, req.LeadAgentID)
	facts, err := PersistDelegationDecisionFacts(flowPlan.StructuredFacts, decision)
	if err != nil {
		return &DispatchResult{
			GoalID:       req.GoalID,
			Status:       DispatchFailed,
			FallbackSafe: true,
			Error:        fmt.Sprintf("persist delegation decision: %v", err),
		}, nil
	}
	facts, err = PersistGoalContractFact(facts, GoalContract{
		GoalDescription:      req.Goal,
		AutoPromoteToMission: req.AutoPromoteToMission,
		LeadID:               req.LeadAgentID,
		ProjectID:            req.ProjectID,
		GoalID:               req.GoalID,
		CreatedAt:            time.Now().UTC(),
	})
	if err != nil {
		return &DispatchResult{
			GoalID:       req.GoalID,
			Status:       DispatchFailed,
			FallbackSafe: true,
			Error:        fmt.Sprintf("persist goal contract: %v", err),
		}, nil
	}
	if req.AutoPromoteToMission {
		facts["nexus.dispatch.auto_promoted"] = "true"
		facts["nexus.dispatch.goal_id"] = req.GoalID
		if source := strings.TrimSpace(req.PromotionSource); source != "" {
			facts["nexus.dispatch.promotion_source"] = source
		}
		if runtimeID := strings.TrimSpace(req.SourceRuntimeID); runtimeID != "" {
			facts["nexus.dispatch.source_runtime_id"] = runtimeID
		}
	}

	// Step 4: Create work plan.
	plan, err := n.CreateWorkPlan(ctx, req.ProjectID, proposal.Title, proposal.Description, flowPlan.Phases, facts)
	if err != nil {
		return &DispatchResult{
			GoalID:       req.GoalID,
			Status:       DispatchFailed,
			FallbackSafe: true,
			Error:        fmt.Sprintf("create work plan: %v", err),
		}, nil
	}

	// Step 5: Start the freshly-created revision through the same execution
	// preview, budget and strict-admission spine used by explicit GO. The
	// upstream promotion policy is the approval boundary for this new revision.
	run, err := n.StartMissionRunApprovedWithOptions(
		ctx,
		plan.ID,
		plan.CurrentRevision,
		req.LeadAgentID,
		runner.DefaultAutonomyContract(),
		true,
		MissionAdmissionOptions{},
	)
	if err != nil {
		var admissionErr *MissionAdmissionError
		if errors.As(err, &admissionErr) {
			return &DispatchResult{
				GoalID:           req.GoalID,
				PlanID:           plan.ID,
				Status:           DispatchRejected,
				AdmissionBlocked: true,
				AdmissionStage:   admissionErr.Stage,
				FallbackSafe:     true,
				Error:            fmt.Sprintf("start mission: %v", err),
			}, nil
		}
		var preparationErr *MissionPreparationError
		if errors.As(err, &preparationErr) {
			return &DispatchResult{
				GoalID:         req.GoalID,
				PlanID:         plan.ID,
				Status:         DispatchFailed,
				AdmissionStage: preparationErr.Stage,
				FallbackSafe:   true,
				Error:          fmt.Sprintf("prepare mission: %v", err),
			}, nil
		}
		return &DispatchResult{
			GoalID: req.GoalID,
			PlanID: plan.ID,
			Status: DispatchFailed,
			Error:  fmt.Sprintf("start mission: %v", err),
		}, nil
	}

	return &DispatchResult{
		GoalID: req.GoalID,
		PlanID: plan.ID,
		RunID:  run.ID,
		Status: DispatchAccepted,
		Run:    run,
	}, nil
}

func dispatchTopologyAllowed(req DispatchRequest, decision DelegationDecision) bool {
	return decision.Delegate || req.AutoPromoteToMission
}

// Dispatch is the synchronous Core entrypoint for routing a goal through the
// Conductor pipeline: decompose → plan → start mission → wait → assess.
// It is a thin wrapper over StartDispatch for CLI callers that want to block.
// The returned status is honest: COMPLETED is reported only when the mission
// run is really completed; blocked/paused/canceled runs keep their own status
// so callers can resume instead of assuming success.
func (n *Nexus) Dispatch(ctx context.Context, req DispatchRequest) (*DispatchResult, error) {
	accepted, err := n.StartDispatch(ctx, req)
	if err != nil {
		return nil, err
	}
	return n.WaitDispatch(ctx, accepted)
}

// WaitDispatch waits for an already accepted Mission without creating another
// WorkPlan or MissionRun. Interactive clients use StartDispatch first so they
// can keep the provider prompt intact until strict admission succeeds, then
// wait in the background after the Mission has actually started.
func (n *Nexus) WaitDispatch(ctx context.Context, accepted *DispatchResult) (*DispatchResult, error) {
	if accepted == nil {
		return nil, fmt.Errorf("accepted dispatch result is required")
	}
	if accepted.Status != DispatchAccepted {
		return accepted, nil
	}
	if strings.TrimSpace(accepted.RunID) == "" {
		return &DispatchResult{
			GoalID: accepted.GoalID,
			PlanID: accepted.PlanID,
			Status: DispatchFailed,
			Error:  "accepted dispatch did not return a mission run id",
		}, nil
	}

	// Step 6: Wait for a settled run state (terminal, blocked or paused).
	final, waitErr := waitPromotedMission(ctx, n, accepted.RunID)
	if waitErr != nil {
		return &DispatchResult{
			GoalID: accepted.GoalID,
			PlanID: accepted.PlanID,
			RunID:  accepted.RunID,
			Status: DispatchFailed,
			Error:  fmt.Sprintf("wait mission: %v", waitErr),
		}, nil
	}

	return &DispatchResult{
		GoalID: accepted.GoalID,
		PlanID: accepted.PlanID,
		RunID:  final.ID,
		Status: dispatchStatusForRunState(final.State),
		Run:    final,
	}, nil
}

// waitPromotedMission polls until the mission run settles: a terminal state,
// or a non-terminal state that needs user action (blocked/paused). It honors
// ctx so a disconnected client releases the worker instead of blocking it
// for the full mission.
func waitPromotedMission(ctx context.Context, n *Nexus, runID string) (*runner.MissionRun, error) {
	for {
		run, err := n.Runner().GetRun(ctx, runID)
		if err != nil {
			return nil, err
		}
		if runner.IsTerminalState(run.State) || run.State == runner.StateBlockedNeedsUser || run.State == runner.StatePaused {
			return run, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}
