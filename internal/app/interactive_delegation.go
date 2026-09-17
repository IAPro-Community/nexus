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
	result, err := n.Dispatch(ctx, nexus.DispatchRequest{
		ProjectID:      session.ProjectID,
		Goal:           prompt,
		SourcePrompt:   prompt,
		LeadAgentID:    leadAgentID,
		DelegationMode: mode,
	})
	if err != nil {
		interactiveDelegationFailure(n, session, leadAgentID, "dispatch goal", err)
		return
	}
	if result.Status == nexus.DispatchFailed || result.Status == nexus.DispatchRejected {
		err = fmt.Errorf("%s: %s", result.Status, result.Error)
		interactiveDelegationFailure(n, session, leadAgentID, "dispatch goal", err)
		return
	}
	message := fmt.Sprintf("Solicitação original: %s\nA Mission %s está em %s (dispatch %s). Revise arquivos, testes, receipts e routing report; depois sintetize o resultado integrado para o usuário.", prompt, result.RunID, result.Run.State, result.Status)
	if err := submitPromptToLead(ctx, n, leadAgentID, session.RuntimeID, message); err != nil {
		fmt.Fprintf(os.Stderr, "[nexus] delegated Mission %s finished, but Lead synthesis failed: %v\n", result.RunID, err)
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
