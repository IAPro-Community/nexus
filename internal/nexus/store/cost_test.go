package store

import (
	"testing"
	"time"
)

func int64ptr(v int64) *int64 { return &v }

func TestCostObservationRoundTripPreservesUnknownAndKnownEvidence(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	known, err := s.RecordCostObservation(CostObservation{
		ProjectID:           "project-1",
		MissionRunID:        "run-1",
		PackageID:           "pkg-1",
		InvocationID:        "inv-1",
		AgentID:             "agent-1",
		RuntimeGenerationID: "gen-1",
		RuntimeID:           "runtime-1",
		Provider:            "codex",
		Profile:             "work",
		Model:               "gpt-5.6",
		Branch:              "feat/test",
		WorkspaceKind:       "worktree",
		InputTokens:         int64ptr(100),
		CachedInputTokens:   int64ptr(20),
		OutputTokens:        int64ptr(30),
		ReasoningTokens:     int64ptr(10),
		DurationMS:          1250,
		CostMicroUSD:        int64ptr(4200),
		CostKind:            CostKindProviderReported,
		Source:              "provider_response",
		Status:              "COMPLETED",
		ObservedAt:          now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if known.ID == "" {
		t.Fatal("cost observation must get a durable id")
	}
	if _, err := s.RecordCostObservation(CostObservation{
		ProjectID:  "project-1",
		Provider:   "claude",
		Profile:    "default",
		DurationMS: 250,
		CostKind:   CostKindUnknown,
		Source:     "runtime_observation",
		Status:     "COMPLETED",
		ObservedAt: now.Add(time.Second),
	}); err != nil {
		t.Fatal(err)
	}

	rows, err := s.ListCostObservations(CostObservationFilter{ProjectID: "project-1", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("observations=%d, want 2", len(rows))
	}
	if rows[0].CostKind != CostKindUnknown || rows[0].CostMicroUSD != nil || rows[0].InputTokens != nil {
		t.Fatalf("unknown evidence was fabricated: %+v", rows[0])
	}
	if rows[1].CostMicroUSD == nil || *rows[1].CostMicroUSD != 4200 || rows[1].InputTokens == nil || *rows[1].InputTokens != 100 {
		t.Fatalf("known evidence lost: %+v", rows[1])
	}
}

func TestCostObservationRejectsInvalidNegativeEvidence(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.RecordCostObservation(CostObservation{
		CostKind:    CostKindInferred,
		Source:      "test",
		InputTokens: int64ptr(-1),
		ObservedAt:  time.Now().UTC(),
	}); err == nil {
		t.Fatal("negative token evidence must be rejected")
	}
	if _, err := s.RecordCostObservation(CostObservation{
		CostKind:   "FAKE",
		Source:     "test",
		ObservedAt: time.Now().UTC(),
	}); err == nil {
		t.Fatal("unknown cost kind must be rejected")
	}
}

func TestCostObservationFiltersByMissionProviderAndTime(t *testing.T) {
	s := openTestStore(t)
	base := time.Now().UTC().Add(-time.Hour)
	for _, obs := range []CostObservation{
		{ProjectID: "p", MissionRunID: "m1", Provider: "codex", Profile: "a", CostKind: CostKindUnknown, Source: "test", ObservedAt: base},
		{ProjectID: "p", MissionRunID: "m2", Provider: "claude", Profile: "b", CostKind: CostKindUnknown, Source: "test", ObservedAt: base.Add(30 * time.Minute)},
		{ProjectID: "p", MissionRunID: "m1", Provider: "codex", Profile: "a", CostKind: CostKindUnknown, Source: "test", ObservedAt: base.Add(45 * time.Minute)},
	} {
		if _, err := s.RecordCostObservation(obs); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := s.ListCostObservations(CostObservationFilter{
		MissionRunID: "m1",
		Provider:     "codex",
		Since:        base.Add(15 * time.Minute),
		Limit:        10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].MissionRunID != "m1" || rows[0].Provider != "codex" {
		t.Fatalf("filtered rows=%+v, want one m1/codex observation", rows)
	}
}

func TestCostSummarySeparatesMeasuredFromUnknown(t *testing.T) {
	s := openTestStore(t)
	cost := int64(5000)
	input := int64(100)
	for _, obs := range []CostObservation{
		{ProjectID: "p", Provider: "codex", Profile: "a", InputTokens: &input, CostMicroUSD: &cost, CostKind: CostKindProviderReported, Source: "provider", DurationMS: 100, ObservedAt: time.Now().UTC()},
		{ProjectID: "p", Provider: "codex", Profile: "a", CostKind: CostKindUnknown, Source: "runtime", DurationMS: 200, ObservedAt: time.Now().UTC()},
		{ProjectID: "p", Provider: "claude", Profile: "b", CostKind: CostKindUnknown, Source: "runtime", DurationMS: 300, ObservedAt: time.Now().UTC()},
	} {
		if _, err := s.RecordCostObservation(obs); err != nil {
			t.Fatal(err)
		}
	}
	agg, err := s.SummarizeCostObservations(CostObservationFilter{ProjectID: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if agg.ObservationCount != 3 || agg.KnownCostObservations != 1 || agg.UnknownCostObservations != 2 {
		t.Fatalf("cost counts=%+v", agg)
	}
	if agg.KnownTokenObservations != 1 || agg.InputTokens != 100 || agg.CostMicroUSD != 5000 || agg.DurationMS != 600 {
		t.Fatalf("aggregate evidence=%+v", agg)
	}
	providers, err := s.SummarizeCostByProvider(CostObservationFilter{ProjectID: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) != 2 {
		t.Fatalf("provider aggregates=%+v, want 2", providers)
	}
	if providers[0].Provider != "codex" || providers[0].CostMicroUSD != 5000 {
		t.Fatalf("provider ordering/evidence=%+v", providers)
	}
}

func TestCostPreferencesDefaultsRoundTripAndValidation(t *testing.T) {
	s := openTestStore(t)

	defaults, err := s.GetCostPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if !defaults.Enabled || defaults.WindowDays != 7 || defaults.AnomalyMultiplier != 3 || defaults.AnomalyMinimumSamples != 5 {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}
	if defaults.SoftLimitMicroUSD != 0 || defaults.HardLimitMicroUSD != 0 || defaults.EmergencyLimitMicroUSD != 0 {
		t.Fatalf("budgets must be opt-in, got %+v", defaults)
	}

	prefs := defaults
	prefs.WindowDays = 30
	prefs.SoftLimitMicroUSD = 1_000_000
	prefs.HardLimitMicroUSD = 2_000_000
	prefs.EmergencyLimitMicroUSD = 3_000_000
	prefs.AnomalyMultiplier = 4
	prefs.AnomalyMinimumSamples = 8
	prefs.AnomalyMinCostMicroUSD = 250_000
	prefs.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)
	if err := s.PutCostPreferences(prefs); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetCostPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if got.WindowDays != prefs.WindowDays ||
		got.SoftLimitMicroUSD != prefs.SoftLimitMicroUSD ||
		got.HardLimitMicroUSD != prefs.HardLimitMicroUSD ||
		got.EmergencyLimitMicroUSD != prefs.EmergencyLimitMicroUSD ||
		got.AnomalyMultiplier != prefs.AnomalyMultiplier ||
		got.AnomalyMinimumSamples != prefs.AnomalyMinimumSamples ||
		got.AnomalyMinCostMicroUSD != prefs.AnomalyMinCostMicroUSD {
		t.Fatalf("cost preferences round trip lost data: got=%+v want=%+v", got, prefs)
	}

	invalid := prefs
	invalid.SoftLimitMicroUSD = 2_000_000
	invalid.HardLimitMicroUSD = 1_000_000
	if err := s.PutCostPreferences(invalid); err == nil {
		t.Fatal("inverted budget thresholds must be rejected")
	}
}
