package nexus

import (
	"context"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/capacitymonitor"
)

// T4 bridge: o adapter fino delega ao EXISTENTE RecommendResources sem ciclo
// de import (capacitymonitor define a interface; este pacote a implementa).
func TestCapacityRecommenderRecommendsAlternative(t *testing.T) {
	pool := []ProviderAccount{
		{Provider: "codex", Profile: "kiver", DisplayName: "kiver",
			Authenticated: true, Available: true, Health: "healthy",
			QuotaRemaining: 0.05, QuotaTotal: 1, QuotaView: measuredBalanceQuotaView("LIVE", 0.05)},
		{Provider: "claude", Profile: "backup", DisplayName: "backup",
			Authenticated: true, Available: true, Health: "healthy",
			QuotaRemaining: 0.8, QuotaTotal: 1, QuotaView: measuredBalanceQuotaView("LIVE", 0.80)},
	}
	r := &CapacityRecommender{Accounts: func() []ProviderAccount { return pool }}
	rec, err := r.RecommendForLowResource(context.Background(),
		capacitymonitor.ResourceKey{Provider: "codex", Profile: "kiver", Group: "claude_gpt", Window: "5h"},
		"quota critical")
	if err != nil {
		t.Fatalf("RecommendForLowResource() = %v", err)
	}
	if !rec.Available {
		t.Fatalf("recommendation unavailable, want available (reason: %q)", rec.Reason)
	}
	if rec.Reason == "" {
		t.Fatal("recommendation reason must never be empty")
	}
	if rec.Provider == "codex" && rec.Profile == "kiver" {
		t.Fatalf("recommended the at-risk resource itself: %+v", rec)
	}
}

func TestCapacityRecommenderNoEligibleStaysUnavailable(t *testing.T) {
	pool := []ProviderAccount{
		{Provider: "codex", Profile: "kiver", DisplayName: "kiver",
			Authenticated: true, Available: true, Health: "healthy",
			QuotaRemaining: 0.05, QuotaTotal: 1, QuotaView: measuredBalanceQuotaView("LIVE", 0.05)},
	}
	r := &CapacityRecommender{Accounts: func() []ProviderAccount { return pool }}
	rec, err := r.RecommendForLowResource(context.Background(),
		capacitymonitor.ResourceKey{Provider: "codex", Profile: "kiver"},
		"quota critical")
	if err != nil {
		t.Fatalf("RecommendForLowResource() = %v", err)
	}
	if rec.Available {
		t.Fatalf("recommendation = %+v, want unavailable (pool só tem o recurso em risco)", rec)
	}
}
