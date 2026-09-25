package web

import (
	"context"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus"
)

// startPromotedMission runs only the admission prefix. The browser terminal
// keeps the original prompt in the provider TUI until this returns ACCEPTED.
func startPromotedMission(n *nexus.Nexus, session registry.RuntimeSession, prompt string) (*nexus.DispatchResult, error) {
	if n == nil || session.ProjectID == "" || session.AgentID == "" {
		return nil, fmt.Errorf("mission promotion is unavailable for this runtime")
	}
	result, err := n.StartDispatch(context.Background(), nexus.DispatchRequest{
		ProjectID:            session.ProjectID,
		Goal:                 prompt,
		SourcePrompt:         prompt,
		LeadAgentID:          session.AgentID,
		DelegationMode:       nexus.DelegationAuto,
		AutoPromoteToMission: true,
		PromotionSource:      "supervised_conversation",
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

// waitPromotedMission waits after admission and reports the settled Mission to
// the same Lead. Once admission succeeds, automatic Direct fallback is unsafe
// because the Mission may already have produced side effects.
func waitPromotedMission(n *nexus.Nexus, session registry.RuntimeSession, prompt string, accepted *nexus.DispatchResult) {
	ctx := context.Background()
	result, err := n.WaitDispatch(ctx, accepted)
	if err != nil {
		_, _ = n.AskAgent(ctx, session.AgentID, fmt.Sprintf("A Mission %s já foi admitida, mas falhou ao aguardar conclusão: %v. Não repita automaticamente a solicitação no Lead porque podem existir side effects parciais.", accepted.RunID, err), false)
		return
	}
	if result == nil {
		_, _ = n.AskAgent(ctx, session.AgentID, fmt.Sprintf("A Mission %s já foi admitida, mas não retornou resultado final. Não repita automaticamente a solicitação no Lead.", accepted.RunID), false)
		return
	}
	if result.Status == nexus.DispatchFailed || result.Status == nexus.DispatchRejected || result.Status == nexus.DispatchCanceled {
		detail := strings.TrimSpace(result.Error)
		if detail == "" && result.Run != nil {
			detail = string(result.Run.State)
		}
		_, _ = n.AskAgent(ctx, session.AgentID, fmt.Sprintf("A Mission %s terminou em %s: %s. Não repita automaticamente a solicitação no Lead porque podem existir side effects parciais.", result.RunID, result.Status, detail), false)
		return
	}
	state := "UNKNOWN"
	if result.Run != nil {
		state = string(result.Run.State)
	}
	_, _ = n.AskAgent(ctx, session.AgentID, fmt.Sprintf("A Mission %s está em %s (dispatch %s) para a solicitação: %s. Revise a evidência e sintetize o resultado.", result.RunID, state, result.Status, prompt), false)
}
