package nexus

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

func continuityTestAccount(provider, profile string, remaining float64) ProviderAccount {
	view := quota.QuotaView{Status: string(model.UsageLive), Available: true}
	return ProviderAccount{
		ID:             provider + ":" + profile,
		Provider:       provider,
		Profile:        profile,
		Authenticated:  true,
		Available:      true,
		QuotaView:      &view,
		QuotaRemaining: remaining,
		QuotaTotal:     1,
		Health:         "healthy",
	}
}

func TestSelectContinuityTargetPrefersSameProviderAccount(t *testing.T) {
	source := registry.RuntimeSession{ProviderID: "codex", ProfileID: "a"}
	accounts := []ProviderAccount{
		continuityTestAccount("claude", "x", 0.95),
		continuityTestAccount("codex", "a", 0.90),
		continuityTestAccount("codex", "b", 0.55),
	}
	got, ok := selectContinuityTarget(source, accounts)
	if !ok {
		t.Fatal("expected continuity target")
	}
	if got.Provider != "codex" || got.Profile != "b" {
		t.Fatalf("target=%s:%s want codex:b", got.Provider, got.Profile)
	}
}

func TestSelectContinuityTargetFallsBackAcrossProvider(t *testing.T) {
	source := registry.RuntimeSession{ProviderID: "codex", ProfileID: "a"}
	accounts := []ProviderAccount{
		continuityTestAccount("codex", "a", 0),
		continuityTestAccount("claude", "review", 0.80),
	}
	got, ok := selectContinuityTarget(source, accounts)
	if !ok {
		t.Fatal("expected cross-provider continuity target")
	}
	if got.Provider != "claude" || got.Profile != "review" {
		t.Fatalf("target=%s:%s want claude:review", got.Provider, got.Profile)
	}
}

func TestSelectContinuityTargetDoesNotPingPong(t *testing.T) {
	source := registry.RuntimeSession{
		ProviderID: "codex",
		ProfileID:  "b",
		Labels: map[string]string{
			"nexus.continuity_excluded": "codex:a",
		},
	}
	accounts := []ProviderAccount{
		continuityTestAccount("codex", "a", 0.99),
		continuityTestAccount("codex", "b", 0.50),
		continuityTestAccount("claude", "x", 0.60),
	}
	got, ok := selectContinuityTarget(source, accounts)
	if !ok {
		t.Fatal("expected target")
	}
	if got.Provider == "codex" && (got.Profile == "a" || got.Profile == "b") {
		t.Fatalf("continuity selected already-attempted account %s:%s", got.Provider, got.Profile)
	}
}

func TestContinuityReasonForFailureIsTyped(t *testing.T) {
	tests := []struct {
		kind model.FailureKind
		want ContinuityReason
		ok   bool
	}{
		{model.FailureQuota, ContinuityQuotaExhausted, true},
		{model.FailureRateLimit, ContinuityRateLimited, true},
		{model.FailureAuth, ContinuityAuthLost, true},
		{model.FailureNetwork, ContinuityResourceUnavailable, true},
		{model.FailureProvider, ContinuityResourceUnavailable, true},
		{model.FailureCommand, "", false},
		{model.FailureUnknown, "", false},
	}
	for _, test := range tests {
		got, ok := ContinuityReasonForFailure(model.Failure{Kind: test.kind})
		if got != test.want || ok != test.ok {
			t.Fatalf("kind=%s got=(%s,%t) want=(%s,%t)", test.kind, got, ok, test.want, test.ok)
		}
	}
}

func TestContinuityHopCountAndExcludedLabel(t *testing.T) {
	session := registry.RuntimeSession{
		ProviderID: "codex",
		ProfileID:  "b",
		Labels: map[string]string{
			"nexus.continuity_hops":     "2",
			"nexus.continuity_excluded": "codex:a",
		},
	}
	if got := continuityHopCount(session); got != 2 {
		t.Fatalf("hops=%d want 2", got)
	}
	got := continuityExcludedResources(session)
	if _, ok := got["codex:a"]; !ok {
		t.Fatal("missing previous account from exclusion set")
	}
	if _, ok := got["codex:b"]; !ok {
		t.Fatal("missing current account from exclusion set")
	}
}

func TestSelectContinuityTargetAvoidsSharedUpstreamQuota(t *testing.T) {
	scope := model.AccountScope{ProviderID: "codex", ProfileID: "a", AccountID: "shared", IdentityVersion: "v1"}
	source := registry.RuntimeSession{ProviderID: "codex", ProfileID: "a", AccountScope: scope}
	duplicate := continuityTestAccount("codex", "b", 0.90)
	scope.ProfileID = "b"
	duplicate.Scope = scope
	independent := continuityTestAccount("codex", "c", 0.50)
	independent.Scope = model.AccountScope{ProviderID: "codex", ProfileID: "c", AccountID: "independent", IdentityVersion: "v1"}
	target, ok := selectContinuityTarget(source, []ProviderAccount{duplicate, independent})
	if !ok || target.Profile != "c" {
		t.Fatalf("shared upstream quota is not a fallback target: target=%+v ok=%t", target, ok)
	}
}
