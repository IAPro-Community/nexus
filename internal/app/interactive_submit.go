package app

import (
	"context"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/promotion"
)


func stickyLeadExplicitDelegateProvider(n *nexus.Nexus, line string) (string, bool, error) {
	if n == nil || !nexus.HasExplicitProviderRoutingRequest(line) {
		return "", false, nil
	}
	provider, ok, err := n.ResolveRegisteredProviderPreference(line)
	if err != nil || !ok {
		return "", false, err
	}
	requirements := nexus.ClassifyTaskRequirements(line)
	mode, ok := nexus.AgentModeForRole(requirements.Role)
	if !ok || !mode.ReadOnlyByDefault || requirements.RequiresDecomposition {
		return "", false, nil
	}
	return provider, true, nil
}

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
	return decision.Action == promotion.Promote && !decision.ExplicitMissionRequest
}
