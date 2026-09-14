package app

import (
	"context"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/promotion"
)

// interactivePromotionDecideTimeout bounds how long Enter may be held while
// Nexus classifies a Lead prompt for Mission promotion. Provider CLIs must
// remain responsive; on timeout the line stays with the Lead (KeepWithLead).
const interactivePromotionDecideTimeout = 250 * time.Millisecond

func decideInteractiveSubmit(ctx context.Context, promoter promotion.Promoter, projectID, line string, mode nexus.DelegationMode) promotion.Action {
	if promotion.IsImmediateSubmitLine(line) {
		return promotion.KeepWithLead
	}
	if ctx == nil {
		ctx = context.Background()
	}
	decideCtx, cancel := context.WithTimeout(ctx, interactivePromotionDecideTimeout)
	defer cancel()
	decision, err := promoter.Decide(decideCtx, projectID, line, mode)
	if err != nil || decision.Action == promotion.KeepWithLead {
		return promotion.KeepWithLead
	}
	return decision.Action
}
