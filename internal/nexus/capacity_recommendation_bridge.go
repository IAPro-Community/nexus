package nexus

import (
	"context"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/capacitymonitor"
)

// CapacityRecommender is the T4 thin adapter from the capacity monitor to the
// EXISTING RecommendResources ranking (resource_recommendation.go).
//
// No import cycle: the Recommender interface is declared in the child package
// capacitymonitor (dependency-neutral) and implemented here in the parent;
// capacitymonitor never imports nexus. Injection happens via
// capacitymonitor.Dependencies.Recommender.
//
// Advisory only: it returns a candidate + reason for publication. It never
// migrates workloads, switches Agents, or mutates scheduler state — any
// switch stays behind the explicit MANUAL allocation flow.
type CapacityRecommender struct {
	// Accounts supplies the pool to rank (nil-safe: empty pool means
	// unavailable, never an error).
	Accounts func() []ProviderAccount
}

// RecommendForLowResource ranks healthy alternatives to the at-risk resource.
func (r *CapacityRecommender) RecommendForLowResource(ctx context.Context, atRisk capacitymonitor.ResourceKey, reason string) (capacitymonitor.Recommendation, error) {
	if err := ctx.Err(); err != nil {
		return capacitymonitor.Recommendation{}, err
	}
	var pool []ProviderAccount
	if r != nil && r.Accounts != nil {
		pool = r.Accounts()
	}
	pool = BalanceCandidateAccounts(pool)
	candidates := make([]ProviderAccount, 0, len(pool))
	for _, acc := range pool {
		if strings.EqualFold(acc.Provider, atRisk.Provider) && strings.EqualFold(acc.Profile, atRisk.Profile) {
			continue
		}
		if acc.RateLimited {
			continue
		}
		if acc.QuotaTotal > 0 && acc.QuotaRemaining <= 0 {
			continue
		}
		candidates = append(candidates, acc)
	}
	if len(candidates) == 0 {
		return capacitymonitor.Recommendation{
			Reason: fmt.Sprintf("no eligible replacement for %s/%s (%s)", atRisk.Provider, atRisk.Profile, strings.TrimSpace(reason)),
		}, nil
	}
	result := RecommendResources(candidates,
		TaskRequirements{TaskKind: "coding", CurrentProvider: atRisk.Provider, CurrentProfile: atRisk.Profile},
		PolicyBalanced)
	if result.Recommended == nil {
		return capacitymonitor.Recommendation{
			Reason: fmt.Sprintf("no eligible replacement for %s/%s (%s)", atRisk.Provider, atRisk.Profile, strings.TrimSpace(reason)),
		}, nil
	}
	top := result.Recommended
	why := strings.TrimSpace(result.Explanation)
	if why == "" {
		why = fmt.Sprintf("fallback for %s/%s: %s", atRisk.Provider, atRisk.Profile, strings.TrimSpace(reason))
	}
	return capacitymonitor.Recommendation{
		Provider:    top.Account.Provider,
		Profile:     top.Account.Profile,
		DisplayName: top.Account.DisplayName,
		Score:       top.TotalScore,
		Reason:      why,
		Available:   true,
	}, nil
}
