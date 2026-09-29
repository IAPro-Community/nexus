package app

import (
	"context"
	"strings"

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

func decideInteractiveSubmit(ctx context.Context, promoter promotion.Promoter, projectID, line string, mode nexus.DelegationMode) promotion.Action {
	return decideInteractiveSubmitDecision(ctx, promoter, projectID, line, mode).Action
}

func shouldFallbackPromotionToLead(decision promotion.Decision) bool {
	// A heuristic topology promotion may safely fall back to the current Lead
	// before Mission admission. Explicit user ownership may not: this includes
	// both an explicit Mission request and a registered provider handoff such as
	// "chame o codex". Falling back that request to AGY/another Lead would violate
	// the requested resource and recreate provider discovery outside Nexus.
	return decision.Action == promotion.Promote &&
		!decision.ExplicitMissionRequest &&
		strings.TrimSpace(decision.Intent.TaskRequirements.PreferProvider) == ""
}
