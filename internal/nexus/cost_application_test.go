package nexus

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestEvaluateCostBudgetSeparatesReportedSpendFromEstimates(t *testing.T) {
	prefs := store.DefaultCostPreferences()
	prefs.SoftLimitMicroUSD = 100
	prefs.HardLimitMicroUSD = 200
	prefs.EmergencyLimitMicroUSD = 300

	tests := []struct {
		name         string
		totals       store.CostAnalyticsTotals
		wantLevel    CostBudgetLevel
		confirmation bool
		wantReported int64
		wantUnknown  int64
	}{
		{
			name: "estimated API equivalent alone must never prove a budget exceeded",
			totals: store.CostAnalyticsTotals{
				ObservationCount: 1, EquivalentCostObservations: 1,
				APIEquivalentMicroUSD: 5000, KnownCostObservations: 1,
			},
			wantLevel: CostBudgetUnknown, confirmation: true, wantUnknown: 1,
		},
		{
			name: "reported amount below hard limit but other estimates remain unknown",
			totals: store.CostAnalyticsTotals{
				ObservationCount: 2, ReportedCostObservations: 1,
				EquivalentCostObservations: 1, ProviderReportedMicroUSD: 50,
				APIEquivalentMicroUSD: 400, KnownCostObservations: 2,
			},
			wantLevel: CostBudgetUnknown, confirmation: true, wantReported: 50, wantUnknown: 1,
		},
		{
			name: "all reported below thresholds can prove headroom",
			totals: store.CostAnalyticsTotals{
				ObservationCount: 2, ReportedCostObservations: 2,
				ProviderReportedMicroUSD: 50, KnownCostObservations: 2,
			},
			wantLevel: CostBudgetOK, wantReported: 50,
		},
		{
			name: "reported lower bound proves soft threshold exceeded even when other costs unknown",
			totals: store.CostAnalyticsTotals{
				ObservationCount: 2, ReportedCostObservations: 1,
				ProviderReportedMicroUSD: 100, UnknownCostObservations: 1,
			},
			wantLevel: CostBudgetSoft, confirmation: true, wantReported: 100, wantUnknown: 1,
		},
		{
			name: "reported lower bound proves hard threshold exceeded",
			totals: store.CostAnalyticsTotals{
				ObservationCount: 2, ReportedCostObservations: 1,
				ProviderReportedMicroUSD: 200, InferredCostObservations: 1,
				InferredMicroUSD: 600,
			},
			wantLevel: CostBudgetHard, confirmation: true, wantReported: 200, wantUnknown: 1,
		},
		{
			name: "reported lower bound proves emergency exceeded",
			totals: store.CostAnalyticsTotals{
				ObservationCount: 1, ReportedCostObservations: 1,
				ProviderReportedMicroUSD: 300,
			},
			wantLevel: CostBudgetEmergency, confirmation: true, wantReported: 300,
		},
		{
			name: "unclassified observation cannot prove remaining headroom",
			totals: store.CostAnalyticsTotals{
				ObservationCount: 1, KnownCostObservations: 1, UnclassifiedMicroUSD: 50,
			},
			wantLevel: CostBudgetUnknown, confirmation: true, wantUnknown: 1,
		},
		{
			name: "no observations is unknown rather than free",
			totals: store.CostAnalyticsTotals{},
			wantLevel: CostBudgetUnknown, confirmation: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := evaluateCostBudget(test.totals, prefs)
			if got.Level != test.wantLevel || got.RequiresConfirmation != test.confirmation {
				t.Fatalf("level=%s confirmation=%v, want %s/%v: %+v",
					got.Level, got.RequiresConfirmation, test.wantLevel, test.confirmation, got)
			}
			if got.ReportedCostMicroUSD != test.wantReported ||
				got.KnownCostMicroUSD != test.wantReported ||
				got.UnverifiedObservations != test.wantUnknown {
				t.Fatalf("reported/nonreported cost classes were mixed: %+v", got)
			}
			if got.APIEquivalentMicroUSD != test.totals.APIEquivalentMicroUSD ||
				got.InferredMicroUSD != test.totals.InferredMicroUSD {
				t.Fatalf("cost estimates disappeared or were misclassified: %+v", got)
			}
		})
	}
	disabled := store.DefaultCostPreferences()
	if got := evaluateCostBudget(store.CostAnalyticsTotals{
		ObservationCount: 1, EquivalentCostObservations: 1, APIEquivalentMicroUSD: 999,
	}, disabled); got.Level != CostBudgetDisabled || got.KnownCostMicroUSD != 0 {
		t.Fatalf("disabled budget still needs honest evidence classification: %+v", got)
	}
}

func TestBudgetConfirmationTokenBindsToUnreportedEvidenceChanges(t *testing.T) {
	prefs := store.DefaultCostPreferences()
	prefs.HardLimitMicroUSD = 100
	baseline := evaluateCostBudget(store.CostAnalyticsTotals{
		ObservationCount: 2, ReportedCostObservations: 1,
		EquivalentCostObservations: 1, ProviderReportedMicroUSD: 20,
		APIEquivalentMicroUSD: 200,
	}, prefs)
	summary := CostSummary{Budget: baseline, Preferences: prefs}
	original := costBudgetConfirmationToken("plan-a", 1, summary)
	if len(original) != 64 {
		t.Fatalf("expected a revision/evidence-bound confirmation token: %q", original)
	}
	summary.Budget.APIEquivalentMicroUSD += 1
	if next := costBudgetConfirmationToken("plan-a", 1, summary); next == original {
		t.Fatal("updated API-equivalent evidence must invalidate approval token")
	}
}

func TestEvaluateCostAnomalyUsesOnlyComparableMeasuredHistory(t *testing.T) {
	now := time.Now().UTC()
	prefs := store.DefaultCostPreferences()
	prefs.AnomalyMinimumSamples = 3
	prefs.AnomalyMultiplier = 2
	prefs.AnomalyMinCostMicroUSD = 1

	cost := func(v int64) *int64 { return &v }
	observations := []store.CostObservation{
		{ID: "latest", Provider: "codex", Profile: "work", CostKind: store.CostKindProviderReported, CostMicroUSD: cost(1000), ObservedAt: now},
		{ID: "same-110", Provider: "codex", Profile: "work", CostKind: store.CostKindProviderReported, CostMicroUSD: cost(110), ObservedAt: now.Add(-time.Minute)},
		{ID: "same-100", Provider: "codex", Profile: "work", CostKind: store.CostKindProviderReported, CostMicroUSD: cost(100), ObservedAt: now.Add(-2 * time.Minute)},
		{ID: "same-90", Provider: "codex", Profile: "work", CostKind: store.CostKindProviderReported, CostMicroUSD: cost(90), ObservedAt: now.Add(-3 * time.Minute)},
		{ID: "other-profile", Provider: "codex", Profile: "personal", CostKind: store.CostKindProviderReported, CostMicroUSD: cost(10000), ObservedAt: now.Add(-4 * time.Minute)},
		{ID: "other-kind", Provider: "codex", Profile: "work", CostKind: store.CostKindInferred, CostMicroUSD: cost(20000), ObservedAt: now.Add(-5 * time.Minute)},
		{ID: "unknown", Provider: "codex", Profile: "work", CostKind: store.CostKindUnknown, ObservedAt: now.Add(-6 * time.Minute)},
	}

	got := evaluateCostAnomaly(observations, prefs)
	if got.Status != CostAnomalyTriggered || !got.Triggered {
		t.Fatalf("anomaly=%+v, want triggered anomaly", got)
	}
	if got.HistoricalSamples != 3 || got.BaselineMedianMicroUSD != 100 || got.ThresholdMicroUSD != 200 {
		t.Fatalf("comparable baseline was polluted: %+v", got)
	}
	if got.Provider != "codex" || got.Profile != "work" || got.CostKind != store.CostKindProviderReported {
		t.Fatalf("latest comparable identity lost: %+v", got)
	}
}

func TestEvaluateCostAnomalyRequiresEnoughComparableEvidence(t *testing.T) {
	prefs := store.DefaultCostPreferences()
	prefs.AnomalyMinimumSamples = 3
	v := int64(100)
	got := evaluateCostAnomaly([]store.CostObservation{
		{ID: "latest", Provider: "codex", Profile: "work", CostKind: store.CostKindProviderReported, CostMicroUSD: &v, ObservedAt: time.Now().UTC()},
		{ID: "older", Provider: "codex", Profile: "work", CostKind: store.CostKindProviderReported, CostMicroUSD: &v, ObservedAt: time.Now().UTC().Add(-time.Minute)},
	}, prefs)
	if got.Status != CostAnomalyInsufficient || got.Triggered {
		t.Fatalf("anomaly=%+v, want insufficient non-triggered evidence", got)
	}
}
