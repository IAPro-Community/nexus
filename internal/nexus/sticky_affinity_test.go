package nexus

import (
	"testing"
)

// Sticky affinity reuses the existing seams:
// RuntimeAffinityPolicy (desired vs actual), RuntimeRoutingDecision (durable
// explanation) and selectCurrentResource (mission allocate keeps the current
// Provider/Profile while it stays eligible). No persistent RouteLease is
// introduced: stickiness is a per-allocation decision, not durable state.
//
// These tests prove the contract:
// - healthy+authenticated+eligible session is retained even when another
// resource outranks it on score;
// - exhaustion/rate-limit/unauthenticated/unhealthy breaks stickiness;
// - a new execution without current state re-selects the best candidate.

func stickyTestAccounts() []ProviderAccount {
	// Same provider so ProviderPriorities tie-break does not mask quota:
	// current has low quota, alternative has high quota.
	return []ProviderAccount{
		{Provider: "codex", Profile: "main", Authenticated: true, Available: true, Health: "healthy", QuotaRemaining: 0.2, QuotaTotal: 1},
		{Provider: "codex", Profile: "backup", Authenticated: true, Available: true, Health: "healthy", QuotaRemaining: 0.9, QuotaTotal: 1},
	}
}

func TestStickyHealthySessionRetainsDespiteHigherScore(t *testing.T) {
	accounts := stickyTestAccounts()
	cfg := AgentConfig{Provider: "codex", Profile: "main"}
	req := TaskRequirements{TaskKind: "coding"}
	selected, keep := selectCurrentResource(accounts, cfg, req, PolicyBalanced)
	if !keep {
		t.Fatalf("healthy session must be retained, got keep=false selected=%+v", selected)
	}
	if selected.Provider != "codex" || selected.Profile != "main" {
		t.Fatalf("sticky must keep current codex:main, got %s:%s", selected.Provider, selected.Profile)
	}
	// Prove the score-only path would have rebalanced: RecommendResources
	// prefers the higher-quota alternative, so stickiness is what prevents it.
	rec := RecommendResources(accounts, req, PolicyBalanced)
	if rec.Recommended == nil {
		t.Fatalf("expected recommendation")
	}
	if rec.Recommended.Account.Profile != "backup" {
		t.Fatalf("score-only recommendation should prefer codex:backup, got %s:%s",
			rec.Recommended.Account.Provider, rec.Recommended.Account.Profile)
	}
}

func TestStickyBreaksOnRateLimit(t *testing.T) {
	accounts := stickyTestAccounts()
	accounts[0].RateLimited = true
	cfg := AgentConfig{Provider: "codex", Profile: "main"}
	_, keep := selectCurrentResource(accounts, cfg, TaskRequirements{TaskKind: "coding"}, PolicyBalanced)
	if keep {
		t.Fatalf("rate-limited session must not stick")
	}
}

func TestStickyBreaksOnExhaustion(t *testing.T) {
	accounts := stickyTestAccounts()
	accounts[0].Available = false
	accounts[0].QuotaRemaining = 0
	cfg := AgentConfig{Provider: "codex", Profile: "main"}
	_, keep := selectCurrentResource(accounts, cfg, TaskRequirements{TaskKind: "coding"}, PolicyBalanced)
	if keep {
		t.Fatalf("exhausted session must not stick")
	}
}

func TestStickyBreaksOnUnauthenticated(t *testing.T) {
	accounts := stickyTestAccounts()
	accounts[0].Authenticated = false
	cfg := AgentConfig{Provider: "codex", Profile: "main"}
	_, keep := selectCurrentResource(accounts, cfg, TaskRequirements{TaskKind: "coding"}, PolicyBalanced)
	if keep {
		t.Fatalf("unauthenticated session must not stick")
	}
}

func TestStickyBreaksOnUnhealthy(t *testing.T) {
	accounts := stickyTestAccounts()
	accounts[0].Health = "unhealthy"
	cfg := AgentConfig{Provider: "codex", Profile: "main"}
	_, keep := selectCurrentResource(accounts, cfg, TaskRequirements{TaskKind: "coding"}, PolicyBalanced)
	if keep {
		t.Fatalf("unhealthy session must not stick")
	}
}

func TestNewExecutionReselectsBest(t *testing.T) {
	accounts := stickyTestAccounts()
	// No current resource: fresh execution must pick the best score.
	_, keep := selectCurrentResource(accounts, AgentConfig{}, TaskRequirements{TaskKind: "coding"}, PolicyBalanced)
	if keep {
		t.Fatalf("empty current must not stick")
	}
	rec := RecommendResources(accounts, TaskRequirements{TaskKind: "coding"}, PolicyBalanced)
	if rec.Recommended == nil || rec.Recommended.Account.Profile != "backup" {
		t.Fatalf("new execution must re-select best (codex:backup), got %+v", rec.Recommended)
	}
}

// ShouldRetainCurrentResource is defined in sticky_affinity.go and documents
// the sticky predicate over the existing eligibility seam.

func TestShouldRetainCurrentResourcePredicate(t *testing.T) {
	accounts := stickyTestAccounts()
	if !ShouldRetainCurrentResource(accounts, AgentConfig{Provider: "codex", Profile: "main"}, TaskRequirements{TaskKind: "coding"}, PolicyBalanced) {
		t.Fatalf("predicate must retain healthy session")
	}
	bad := stickyTestAccounts()
	bad[0].RateLimited = true
	if ShouldRetainCurrentResource(bad, AgentConfig{Provider: "codex", Profile: "main"}, TaskRequirements{TaskKind: "coding"}, PolicyBalanced) {
		t.Fatalf("predicate must release rate-limited session")
	}
}
