package nexus

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestEvaluateCostBudgetNeverTreatsPartialEvidenceAsHeadroom(t *testing.T) {
	prefs := store.DefaultCostPreferences()
	prefs.SoftLimitMicroUSD = 100
	prefs.HardLimitMicroUSD = 200
	prefs.EmergencyLimitMicroUSD = 300

	tests := []struct {
		name         string
		known        int64
		evidence     CostEvidenceState
		wantLevel    CostBudgetLevel
		confirmation bool
	}{
		{name: "partial below limits stays unknown", known: 50, evidence: CostEvidencePartial, wantLevel: CostBudgetUnknown},
		{name: "measured below limits is ok", known: 50, evidence: CostEvidenceMeasured, wantLevel: CostBudgetOK},
		{name: "known lower bound can prove soft exceeded", known: 100, evidence: CostEvidencePartial, wantLevel: CostBudgetSoft},
		{name: "known lower bound can prove hard exceeded", known: 200, evidence: CostEvidencePartial, wantLevel: CostBudgetHard, confirmation: true},
		{name: "known lower bound can prove emergency exceeded", known: 300, evidence: CostEvidencePartial, wantLevel: CostBudgetEmergency, confirmation: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := evaluateCostBudget(store.CostAggregate{CostMicroUSD: test.known}, prefs, test.evidence)
			if got.Level != test.wantLevel {
				t.Fatalf("level=%s, want %s: %+v", got.Level, test.wantLevel, got)
			}
			if got.RequiresConfirmation != test.confirmation {
				t.Fatalf("requires_confirmation=%v, want %v: %+v", got.RequiresConfirmation, test.confirmation, got)
			}
		})
	}

	disabled := store.DefaultCostPreferences()
	if got := evaluateCostBudget(store.CostAggregate{CostMicroUSD: 999}, disabled, CostEvidenceMeasured); got.Level != CostBudgetDisabled {
		t.Fatalf("unconfigured thresholds level=%s, want DISABLED", got.Level)
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
