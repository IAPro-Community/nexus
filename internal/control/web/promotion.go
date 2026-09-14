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
		ProjectID:      session.ProjectID,
		Goal:           prompt,
		SourcePrompt:   prompt,
		LeadAgentID:    session.AgentID,
		DelegationMode: nexus.DelegationAuto,
	})
	if err != nil || result.Status == nexus.DispatchFailed || result.Status == nexus.DispatchRejected {
		return
	}
	_, _ = n.AskAgent(ctx, session.AgentID, fmt.Sprintf("A Mission %s terminou em %s para a solicitação: %s. Revise a evidência e sintetize o resultado.", result.RunID, result.Run.State, prompt), false)
}
