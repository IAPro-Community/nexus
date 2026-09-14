package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

func submitPromptToRuntime(runtimeID, prompt string) error {
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.SubmitPrompt(prompt)
}

func submitPromptToLead(ctx context.Context, n *nexus.Nexus, agentID, runtimeID, prompt string) error {
	if n != nil && strings.TrimSpace(agentID) != "" {
		if _, err := n.AskAgent(ctx, agentID, prompt, false); err == nil {
			return nil
		}
	}
	return submitPromptToRuntime(runtimeID, prompt)
}

func interactiveDelegationMode(session registry.RuntimeSession) nexus.DelegationMode {
	mode := nexus.DelegationMode("")
	if session.Labels != nil {
		mode = nexus.DelegationMode(session.Labels["nexus.delegation_mode"])
	}
	if override := strings.TrimSpace(os.Getenv("NEXUS_DELEGATION_MODE")); override != "" {
		mode = nexus.DelegationMode(override)
	}
	return nexus.NormalizeDelegationMode(mode)
}

func runInteractiveDelegatedMission(n *nexus.Nexus, session registry.RuntimeSession, leadAgentID, prompt string, mode nexus.DelegationMode) {
	ctx := context.Background()
	proposal, err := n.DecomposePromptIntoFlowProposal(ctx, nexus.FlowDecompositionRequest{
		ProjectID: session.ProjectID,
		Goal:      prompt, SourcePrompt: prompt, DelegationMode: mode,
	})
	if err != nil {
		interactiveDelegationFailure(n, session, leadAgentID, "decompor prompt", err)
		return
	}
	flowPlan := nexus.WorkPlanFromFlow(proposal.Flow)
	decision, err := nexus.ReadDelegationDecisionFacts(flowPlan.StructuredFacts)
	if err != nil || !decision.Delegate {
		if err == nil {
			err = fmt.Errorf("dispatch brake rejected the delegated topology")
		}
		interactiveDelegationFailure(n, session, leadAgentID, "validar decisão de delegação", err)
		return
	}
	decision = nexus.WithLeadAgent(decision, leadAgentID)
	facts, err := nexus.PersistDelegationDecisionFacts(flowPlan.StructuredFacts, decision)
	if err != nil {
		interactiveDelegationFailure(n, session, leadAgentID, "persistir decisão de delegação", err)
		return
	}
	plan, err := n.CreateWorkPlan(ctx, session.ProjectID, proposal.Title, proposal.Description, flowPlan.Phases, facts)
	if err != nil {
		interactiveDelegationFailure(n, session, leadAgentID, "persistir WorkPlan", err)
		return
	}
	run, err := n.StartMissionRun(ctx, plan.ID, leadAgentID, runner.DefaultAutonomyContract(), true)
	if err != nil {
		interactiveDelegationFailure(n, session, leadAgentID, "iniciar Mission", err)
		return
	}
	final, waitErr := waitInteractiveMission(n, run.ID)
	if waitErr != nil {
		interactiveDelegationFailure(n, session, leadAgentID, "aguardar Mission", waitErr)
		return
	}
	message := fmt.Sprintf("Solicitação original: %s\nA Mission %s terminou em %s. Revise arquivos, testes, receipts e routing report; depois sintetize o resultado integrado para o usuário.", prompt, final.ID, final.State)
	if err := submitPromptToLead(ctx, n, leadAgentID, session.RuntimeID, message); err != nil {
		fmt.Fprintf(os.Stderr, "[nexus] delegated Mission %s finished, but Lead synthesis failed: %v\n", final.ID, err)
	}
}

func waitInteractiveMission(n *nexus.Nexus, runID string) (*runner.MissionRun, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		run, err := n.Runner().GetRun(context.Background(), runID)
		if err != nil {
			return nil, err
		}
		if runner.IsTerminalState(run.State) || run.State == runner.StateBlockedNeedsUser || run.State == runner.StatePaused {
			return run, nil
		}
		<-ticker.C
	}
}

func interactiveDelegationFailure(n *nexus.Nexus, session registry.RuntimeSession, leadAgentID, action string, err error) {
	if err == nil {
		return
	}
	fallback := fmt.Sprintf("Nexus não conseguiu %s para a solicitação; trate-a diretamente e informe a limitação: %v", action, err)
	fmt.Fprintf(os.Stderr, "[nexus] delegated Mission could not %s: %v; Lead remains available\n", action, err)
	_ = submitPromptToLead(context.Background(), n, leadAgentID, session.RuntimeID, fallback)
}
