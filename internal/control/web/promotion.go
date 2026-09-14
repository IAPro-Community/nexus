package web

import (
	"context"
	"fmt"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

// launchPromotedMission is shared by AUTO and an explicit ASK confirmation.
// The browser terminal remains attached to the same Lead while the Mission
// executes independently and reports its result through the existing Agent
// message transport.
func launchPromotedMission(n *nexus.Nexus, session registry.RuntimeSession, prompt string) {
	if n == nil || session.ProjectID == "" || session.AgentID == "" {
		return
	}
	ctx := context.Background()
	proposal, err := n.DecomposePromptIntoFlowProposal(ctx, nexus.FlowDecompositionRequest{
		ProjectID: session.ProjectID, Goal: prompt, SourcePrompt: prompt, DelegationMode: nexus.DelegationAuto,
	})
	if err != nil {
		return
	}
	flowPlan := nexus.WorkPlanFromFlow(proposal.Flow)
	decision, err := nexus.ReadDelegationDecisionFacts(flowPlan.StructuredFacts)
	if err != nil || !decision.Delegate {
		return
	}
	decision = nexus.WithLeadAgent(decision, session.AgentID)
	facts, err := nexus.PersistDelegationDecisionFacts(flowPlan.StructuredFacts, decision)
	if err != nil {
		return
	}
	plan, err := n.CreateWorkPlan(ctx, session.ProjectID, proposal.Title, proposal.Description, flowPlan.Phases, facts)
	if err != nil {
		return
	}
	run, err := n.StartMissionRun(ctx, plan.ID, session.AgentID, runner.DefaultAutonomyContract(), true)
	if err != nil {
		return
	}
	final, err := waitPromotedMission(n, run.ID)
	if err != nil {
		return
	}
	_, _ = n.AskAgent(ctx, session.AgentID, fmt.Sprintf("A Mission %s terminou em %s para a solicitação: %s. Revise a evidência e sintetize o resultado.", final.ID, final.State, prompt), false)
}

func waitPromotedMission(n *nexus.Nexus, runID string) (*runner.MissionRun, error) {
	for {
		run, err := n.Runner().GetRun(context.Background(), runID)
		if err != nil {
			return nil, err
		}
		if runner.IsTerminalState(run.State) || run.State == runner.StateBlockedNeedsUser || run.State == runner.StatePaused {
			return run, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
}
