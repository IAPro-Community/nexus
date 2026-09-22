package web

import (
	"context"
	"fmt"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus"
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
	result, err := n.Dispatch(ctx, nexus.DispatchRequest{
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
		_, _ = n.AskAgent(ctx, session.AgentID, fmt.Sprintf("A promoção para Mission falhou antes do dispatch: %v. Não execute a solicitação fora da Mission; informe o usuário para tentar novamente.", err), false)
		return
	}
	if result == nil {
		_, _ = n.AskAgent(ctx, session.AgentID, "A promoção para Mission não retornou resultado. Não execute a solicitação fora da Mission; informe o usuário para tentar novamente.", false)
		return
	}
	if result.Status == nexus.DispatchFailed || result.Status == nexus.DispatchRejected {
		_, _ = n.AskAgent(ctx, session.AgentID, fmt.Sprintf("A promoção para Mission foi recusada (%s): %s. Não execute a solicitação fora da Mission; informe o usuário.", result.Status, result.Error), false)
		return
	}
	state := "UNKNOWN"
	if result.Run != nil {
		state = string(result.Run.State)
	}
	_, _ = n.AskAgent(ctx, session.AgentID, fmt.Sprintf("A Mission %s está em %s (dispatch %s) para a solicitação: %s. Revise a evidência e sintetize o resultado.", result.RunID, state, result.Status, prompt), false)
}
