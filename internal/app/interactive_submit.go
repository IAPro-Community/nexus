package app

import (
	"context"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/promotion"
)

func decideInteractiveSubmitDecision(ctx context.Context, promoter promotion.Promoter, projectID, line string, mode nexus.DelegationMode) promotion.Decision {
	if promotion.IsImmediateSubmitLine(line) {
		return promotion.Decision{
			Action: promotion.KeepWithLead,
			Mode:   mode,
			Reason: "provider command bypasses Mission promotion",
		}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	decideCtx, cancel := context.WithTimeout(ctx, promotion.InteractiveDecisionTimeout)
	defer cancel()
	decision, err := promoter.Decide(decideCtx, projectID, line, mode)
	if err != nil {
		return promotion.Decision{
			Action: promotion.KeepWithLead,
			Mode:   mode,
			Reason: "promotion decision unavailable; fail open to interactive Lead",
		}
	}
	return decision
}

func shouldFallbackPromotionToLead(decision promotion.Decision) bool {
	return decision.Action == promotion.Promote && !decision.ExplicitMissionRequest
}
