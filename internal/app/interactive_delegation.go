package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus"
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
	mode := nexus.DelegationOff
	if session.Labels != nil {
		if configured := strings.TrimSpace(session.Labels["nexus.delegation_mode"]); configured != "" {
			mode = nexus.DelegationMode(configured)
		}
	}
	if override := strings.TrimSpace(os.Getenv("NEXUS_DELEGATION_MODE")); override != "" {
		mode = nexus.DelegationMode(override)
	}
	return nexus.NormalizeInteractiveDelegationMode(mode)
}

// confirmedInteractiveDelegationMode converts ASK to AUTO only after the
// interactive caller has received an explicit "yes". Otherwise a newly built
// WorkPlan would still carry PendingApproval=true and immediately block at
// Mission admission even though the user just approved it.
func confirmedInteractiveDelegationMode(mode nexus.DelegationMode) nexus.DelegationMode {
	if nexus.NormalizeDelegationMode(mode) == nexus.DelegationAsk {
		return nexus.DelegationAuto
	}
	return nexus.NormalizeDelegationMode(mode)
}

// startInteractiveDelegatedMission runs only the synchronous admission prefix.
// The caller keeps the original provider line untouched until this returns
// ACCEPTED. This prevents an unsuccessful heuristic promotion from consuming
// the user's prompt.
func startInteractiveDelegatedMission(n *nexus.Nexus, session registry.RuntimeSession, leadAgentID, prompt string, mode nexus.DelegationMode) (*nexus.DispatchResult, error) {
	if n == nil {
		return nil, fmt.Errorf("nexus is unavailable")
	}
	result, err := n.StartDispatch(context.Background(), nexus.DispatchRequest{
		ProjectID:            session.ProjectID,
		Goal:                 prompt,
		SourcePrompt:         prompt,
		LeadAgentID:          leadAgentID,
		DelegationMode:       confirmedInteractiveDelegationMode(mode),
		AutoPromoteToMission: true,
		PromotionSource:      "supervised_cli",
		SourceRuntimeID:      session.RuntimeID,
	})
	if err != nil {
		return nil, fmt.Errorf("prepare Mission: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("prepare Mission: empty dispatch result")
	}
	if result.Status != nexus.DispatchAccepted {
		detail := strings.TrimSpace(result.Error)
		if detail == "" {
			detail = "Mission admission was not accepted"
		}
		return result, fmt.Errorf("%s: %s", result.Status, detail)
	}
	if strings.TrimSpace(result.RunID) == "" {
		return nil, fmt.Errorf("mission admission returned no run id")
	}
	return result, nil
}

func waitInteractiveDelegatedMission(n *nexus.Nexus, session registry.RuntimeSession, leadAgentID, prompt string, accepted *nexus.DispatchResult) {
	ctx := context.Background()
	result, err := n.WaitDispatch(ctx, accepted)
	if err != nil {
		interactiveDelegationFailure(n, session, leadAgentID, accepted.RunID, "wait for Mission", err)
		return
	}
	if result == nil {
		interactiveDelegationFailure(n, session, leadAgentID, accepted.RunID, "wait for Mission", fmt.Errorf("empty dispatch result"))
		return
	}
	if result.Status == nexus.DispatchFailed || result.Status == nexus.DispatchRejected || result.Status == nexus.DispatchCanceled {
		detail := strings.TrimSpace(result.Error)
		if detail == "" && result.Run != nil {
			detail = string(result.Run.State)
		}
		interactiveDelegationFailure(n, session, leadAgentID, result.RunID, "complete Mission", fmt.Errorf("%s: %s", result.Status, detail))
		return
	}
	state := "UNKNOWN"
	if result.Run != nil {
		state = string(result.Run.State)
	}
	message := fmt.Sprintf("Solicitação original: %s\nA Mission %s está em %s (dispatch %s). Revise arquivos, testes, receipts e routing report; depois sintetize o resultado integrado para o usuário.", prompt, result.RunID, state, result.Status)
	if err := submitPromptToLead(ctx, n, leadAgentID, session.RuntimeID, message); err != nil {
		fmt.Fprintf(os.Stderr, "[nexus] delegated Mission %s settled, but Lead synthesis failed: %v\n", result.RunID, err)
	}
}


func runInteractiveSingleTurnDelegate(n *nexus.Nexus, session registry.RuntimeSession, leadAgentID, prompt, preferredProvider string) {
	if n == nil {
		return
	}
	result, err := n.ExecuteSingleTurnDelegate(context.Background(), session.ProjectID, prompt, preferredProvider)
	if err != nil {
		message := fmt.Sprintf("A delegação single-turn solicitada para %s falhou antes de criar qualquer Mission: %v. Não invente um resultado; informe a limitação ao usuário.", firstNonEmptyString(preferredProvider, "um provider registrado"), err)
		_ = submitPromptToLead(context.Background(), n, leadAgentID, session.RuntimeID, message)
		return
	}
	message := fmt.Sprintf("Solicitação original: %s\nO Nexus executou um DELEGATE single-turn sem Mission usando %s:%s. Resultado do especialista:\n\n%s\n\nSintetize este resultado para o usuário sem repetir a execução.", prompt, result.Provider, result.Profile, result.Output)
	if result.Fallback {
		message += "\n\nObservação: o provider solicitado não estava elegível e o ResourceScheduler usou fallback registrado."
	}
	_ = submitPromptToLead(context.Background(), n, leadAgentID, session.RuntimeID, message)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func interactiveDelegationFailure(n *nexus.Nexus, session registry.RuntimeSession, leadAgentID, runID, action string, err error) {
	if err == nil {
		return
	}
	runLabel := strings.TrimSpace(runID)
	if runLabel == "" {
		runLabel = "unknown"
	}
	fallback := fmt.Sprintf("A Mission %s já foi admitida, mas o Nexus não conseguiu %s. Não repita automaticamente a solicitação no Lead porque podem existir side effects parciais. Informe o estado e a limitação ao usuário: %v", runLabel, action, err)
	fmt.Fprintf(os.Stderr, "[nexus] Mission %s could not %s: %v; automatic Direct fallback is disabled after admission\n", runLabel, action, err)
	_ = submitPromptToLead(context.Background(), n, leadAgentID, session.RuntimeID, fallback)
}
