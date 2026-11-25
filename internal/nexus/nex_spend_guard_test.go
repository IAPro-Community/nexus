package nexus

import "testing"

func TestEvaluateNexSpendGuard(t *testing.T) {
	costOK := &HUDCostSummary{BudgetLevel: "OK", KnownMicroUSD: 1_000_000}
	if d := EvaluateNexSpendGuard(costOK, NexSpendPolicy{DailyCapMicroUSD: 5_000_000, AllowPaidFallback: false}, true); d.Allowed || d.Code != "paid_fallback_denied" {
		t.Fatalf("expected paid fallback denied by default, got %+v", d)
	}

	costExceeded := &HUDCostSummary{BudgetLevel: "EXCEEDED", KnownMicroUSD: 100}
	if d := EvaluateNexSpendGuard(costExceeded, NexSpendPolicy{AllowPaidFallback: true}, false); d.Allowed || d.Code != "spend_cap_exceeded" {
		t.Fatalf("expected exceeded budget denial, got %+v", d)
	}

	if d := EvaluateNexSpendGuard(costOK, NexSpendPolicy{DailyCapMicroUSD: 1_000_000}, false); d.Allowed || d.Code != "spend_cap_exceeded" {
		t.Fatalf("expected daily cap denial, got %+v", d)
	}

	if d := EvaluateNexSpendGuard(costOK, NexSpendPolicy{DailyCapMicroUSD: 5_000_000, AllowPaidFallback: false}, false); !d.Allowed {
		t.Fatalf("expected within-cap non-fallback request to succeed, got %+v", d)
	}
}
