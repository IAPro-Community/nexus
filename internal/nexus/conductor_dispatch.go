package nexus

import (
	"context"
	"fmt"
	"time"

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
	// AutoPromoteToMission skips user confirmation.
	AutoPromoteToMission bool `json:"auto_promote_to_mission"`
}

// DispatchResult is the output of Conductor.Dispatch.
type DispatchResult struct {
	GoalID string             `json:"goal_id"`
	RunID  string             `json:"run_id"`
	Status DispatchStatus     `json:"status"`
	Run    *runner.MissionRun `json:"run,omitempty"`
	Error  string             `json:"error,omitempty"`
}

// DispatchStatus enumerates possible dispatch outcomes.
type DispatchStatus string

const (
	DispatchAccepted  DispatchStatus = "ACCEPTED"
	DispatchRejected  DispatchStatus = "REJECTED"
	DispatchFailed    DispatchStatus = "FAILED"
	DispatchCompleted DispatchStatus = "COMPLETED"
)

// Dispatch is the single Core entrypoint for routing a goal through the
// Conductor pipeline: decompose → plan → start mission → wait → assess.
// Both CLI and Web promotion paths delegate to this method.
func (n *Nexus) Dispatch(ctx context.Context, req DispatchRequest) (*DispatchResult, error) {
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
		req.GoalID = fmt.Sprintf("goal_%d", time.Now().UnixNano())
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
			GoalID: req.GoalID,
			Status: DispatchFailed,
			Error:  fmt.Sprintf("decompose prompt: %v", err),
		}, nil
	}

	// Step 2: Create work plan from flow.
	flowPlan := WorkPlanFromFlow(proposal.Flow)
	decision, err := ReadDelegationDecisionFacts(flowPlan.StructuredFacts)
	if err != nil || !decision.Delegate {
		if err == nil {
			err = fmt.Errorf("dispatch brake rejected the delegated topology")
		}
		return &DispatchResult{
			GoalID: req.GoalID,
			Status: DispatchRejected,
			Error:  fmt.Sprintf("validate delegation decision: %v", err),
		}, nil
	}

	// Step 3: Bind lead agent.
	decision = WithLeadAgent(decision, req.LeadAgentID)
	facts, err := PersistDelegationDecisionFacts(flowPlan.StructuredFacts, decision)
	if err != nil {
		return &DispatchResult{
			GoalID: req.GoalID,
			Status: DispatchFailed,
			Error:  fmt.Sprintf("persist delegation decision: %v", err),
		}, nil
	}

	// Step 4: Create work plan.
	plan, err := n.CreateWorkPlan(ctx, req.ProjectID, proposal.Title, proposal.Description, flowPlan.Phases, facts)
	if err != nil {
		return &DispatchResult{
			GoalID: req.GoalID,
			Status: DispatchFailed,
			Error:  fmt.Sprintf("create work plan: %v", err),
		}, nil
	}

	// Step 5: Start mission run.
	run, err := n.StartMissionRun(ctx, plan.ID, req.LeadAgentID, runner.DefaultAutonomyContract(), true)
	if err != nil {
		return &DispatchResult{
			GoalID: req.GoalID,
			Status: DispatchFailed,
			Error:  fmt.Sprintf("start mission: %v", err),
		}, nil
	}

	// Step 6: Wait for mission completion.
	final, waitErr := waitPromotedMission(ctx, n, run.ID)
	if waitErr != nil {
		return &DispatchResult{
			GoalID: req.GoalID,
			RunID:  run.ID,
			Status: DispatchFailed,
			Error:  fmt.Sprintf("wait mission: %v", waitErr),
		}, nil
	}

	return &DispatchResult{
		GoalID: req.GoalID,
		RunID:  final.ID,
		Status: DispatchCompleted,
		Run:    final,
	}, nil
}

// waitPromotedMission polls until the mission run reaches a terminal state.
// It honors ctx so a disconnected HTTP client releases the worker instead of
// blocking it for the full mission.
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
