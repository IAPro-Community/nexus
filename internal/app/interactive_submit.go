package app

import (
	"context"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/promotion"
)

func decideInteractiveSubmit(ctx context.Context, promoter promotion.Promoter, projectID, line string, mode nexus.DelegationMode) promotion.Action {
	if promotion.IsImmediateSubmitLine(line) {
		return promotion.KeepWithLead
	}
	if ctx == nil {
		ctx = context.Background()
	}
	decideCtx, cancel := context.WithTimeout(ctx, promotion.InteractiveDecisionTimeout)
	defer cancel()
	decision, err := promoter.Decide(decideCtx, projectID, line, mode)
	if err != nil || decision.Action == promotion.KeepWithLead {
		return promotion.KeepWithLead
	}
	return decision.Action
}
