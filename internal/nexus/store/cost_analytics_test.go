package store

import (
	"testing"
	"time"
)

func TestCostAnalyticsSeparatesReportedEstimatedUnknownAndProjectScope(t *testing.T) {
	st := openTestStore(t)
	day := time.Now().UTC().Add(-24 * time.Hour).Truncate(24 * time.Hour).Add(12 * time.Hour)
	otherDay := day.Add(-24 * time.Hour)
	reported, estimate, outside := int64(1200), int64(3600), int64(100000)
	input, cache, output := int64(200), int64(50), int64(20)
	observations := []CostObservation{
		{ProjectID: "alpha", Provider: "codex", Profile: "work", Model: "model-one", CostKind: CostKindProviderReported, CostMicroUSD: &reported, InputTokens: &input, CachedInputTokens: &cache, OutputTokens: &output, Source: "provider", ObservedAt: day},
		{ProjectID: "alpha", Provider: "codex", Profile: "work", Model: "model-one", CostKind: CostKindAPIEquivalentEstimate, CostMicroUSD: &estimate, Source: "ratecard", ObservedAt: day.Add(time.Minute)},
		{ProjectID: "alpha", Provider: "claude", Profile: "default", Model: "model-two", CostKind: CostKindUnknown, Source: "runtime", ObservedAt: otherDay},
		{ProjectID: "beta", Provider: "codex", Model: "other-project", CostKind: CostKindProviderReported, CostMicroUSD: &outside, Source: "provider", ObservedAt: day},
		{ProjectID: "alpha", Provider: "codex", Model: "expired", CostKind: CostKindProviderReported, CostMicroUSD: &outside, Source: "provider", ObservedAt: day.Add(-30 * 24 * time.Hour)},
	}
	for _, observation := range observations {
		if _, err := st.RecordCostObservation(observation); err != nil {
			t.Fatal(err)
		}
	}
	view, err := st.SummarizeCostAnalytics(CostObservationFilter{
		ProjectID: "alpha", Since: day.Add(-7 * 24 * time.Hour),
	}, 25)
	if err != nil {
		t.Fatal(err)
	}
	if view.Total.ObservationCount != 3 || view.Total.KnownCostObservations != 2 ||
		view.Total.UnknownCostObservations != 1 || view.Total.KnownTokenObservations != 1 {
		t.Fatalf("wrong evidence coverage: %+v", view.Total)
	}
	if view.Total.ProviderReportedMicroUSD != reported ||
		view.Total.APIEquivalentMicroUSD != estimate ||
		view.Total.InferredMicroUSD != 0 || view.Total.UnclassifiedMicroUSD != 0 {
		t.Fatalf("reported vs estimated values mixed: %+v", view.Total)
	}
	evidence, err := st.SummarizeCostEvidence(CostObservationFilter{
		ProjectID: "alpha", Since: day.Add(-7 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if stripDerivedAnalytics(evidence) != stripDerivedAnalytics(view.Total) || evidence.ReportedCostObservations != 1 ||
		evidence.EquivalentCostObservations != 1 || evidence.UnknownCostObservations != 1 {
		t.Fatalf("single aggregate and full analytics disagree about evidence classes: %+v vs %+v", evidence, view.Total)
	}

	if view.Total.InputTokens != input || view.Total.CachedInputTokens != cache ||
		view.Total.OutputTokens != output {
		t.Fatalf("token evidence was lost: %+v", view.Total)
	}
	if len(view.ByDay) != 2 || view.ByDay[0].Date >= view.ByDay[1].Date {
		t.Fatalf("daily series must be sorted and scoped: %+v", view.ByDay)
	}
	if len(view.ByModel) != 2 || view.ByModel[0].Model != "model-one" ||
		view.ByModel[0].ObservationCount != 2 || view.ByModel[1].Model != "model-two" {
		t.Fatalf("model attribution incorrect: %+v", view.ByModel)
	}
	if view.ModelsTruncated {
		t.Fatal("unexpected truncated model list")
	}
	top, err := st.SummarizeCostAnalytics(CostObservationFilter{ProjectID: "alpha", Since: day.Add(-7 * 24 * time.Hour)}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(top.ByModel) != 1 || !top.ModelsTruncated || top.Total.ObservationCount != 3 {
		t.Fatalf("top-model limit must never truncate aggregate totals: %+v", top)
	}
}

func TestCostAnalyticsIsBoundedAndEmptyEvidenceIsNotBillable(t *testing.T) {
	st := openTestStore(t)
	for _, max := range []int{0, 51} {
		if _, err := st.SummarizeCostAnalytics(CostObservationFilter{}, max); err == nil {
			t.Fatalf("maxModels=%d must fail", max)
		}
	}
	empty, err := st.SummarizeCostAnalytics(CostObservationFilter{ProjectID: "not-observed"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if empty.Total.ObservationCount != 0 || len(empty.ByDay) != 0 ||
		len(empty.ByModel) != 0 || empty.ModelsTruncated {
		t.Fatalf("empty evidence must not invent model/daily totals: %+v", empty)
	}
	if empty.Total.ErrorRate != nil || empty.Total.InvocationsFailed != 0 ||
		empty.Total.InvocationsTotal != 0 || empty.Total.AvgDurationMS != nil ||
		empty.Total.AvgTTFTMS != nil || empty.Total.AvgOutputTokensPerSecond != nil {
		t.Fatalf("empty scope must never fabricate error or latency evidence: %+v", empty.Total)
	}
}

// stripDerivedAnalytics returns a copy without the derived pointer fields so
// the legacy plain-aggregate struct equality stays meaningful after F2.5a.
func stripDerivedAnalytics(t CostAnalyticsTotals) CostAnalyticsTotals {
	t.ErrorRate = nil
	t.AvgDurationMS = nil
	t.AvgTTFTMS = nil
	t.AvgOutputTokensPerSecond = nil
	return t
}

// TestCostAnalyticsClassifiesFailuresWithEvidenceAverages covers the F2.5a
// failure observability contract: UNKNOWN is a NON-failure, explicit failure
// statuses (including substring variants like FAILED_TIMEOUT and BLOCKED) are
// failures, error_rate is a derived fraction that only exists with
// observations, and averages only use actually measured evidence.
func TestCostAnalyticsClassifiesFailuresWithEvidenceAverages(t *testing.T) {
	st := openTestStore(t)
	day := time.Now().UTC().Add(-24 * time.Hour).Truncate(24 * time.Hour).Add(12 * time.Hour)
	ttft := func(ms int64) *int64 { return &ms }
	tps := func(v float64) *float64 { return &v }
	observations := []CostObservation{
		{ProjectID: "alpha", Provider: "codex", Model: "model-one", CostKind: CostKindProviderReported, Source: "provider", Status: "COMPLETED", DurationMS: 1000, TTFTMS: ttft(200), OutputTokensPerSecond: tps(10), ObservedAt: day},
		{ProjectID: "alpha", Provider: "codex", Model: "model-one", CostKind: CostKindProviderReported, Source: "provider", Status: "FAILED", DurationMS: 2000, ObservedAt: day.Add(time.Minute)},
		{ProjectID: "alpha", Provider: "codex", Model: "model-one", CostKind: CostKindProviderReported, Source: "provider", Status: "FAILED_TIMEOUT", TTFTMS: ttft(400), OutputTokensPerSecond: tps(5), ObservedAt: day.Add(2 * time.Minute)},
		{ProjectID: "alpha", Provider: "codex", Model: "model-one", CostKind: CostKindProviderReported, Source: "provider", Status: "", ObservedAt: day.Add(3 * time.Minute)},
		{ProjectID: "alpha", Provider: "codex", Model: "model-one", CostKind: CostKindProviderReported, Source: "provider", Status: "CANCELED", ObservedAt: day.Add(4 * time.Minute)},
		{ProjectID: "alpha", Provider: "codex", Model: "model-one", CostKind: CostKindProviderReported, Source: "provider", Status: "UNKNOWN", ObservedAt: day.Add(5 * time.Minute)},
		{ProjectID: "alpha", Provider: "claude", Model: "model-two", CostKind: CostKindProviderReported, Source: "provider", Status: "BLOCKED", ObservedAt: day.Add(6 * time.Minute)},
		// Outside the project scope: must not leak into the alpha aggregates.
		{ProjectID: "beta", Provider: "codex", Model: "model-one", CostKind: CostKindProviderReported, Source: "provider", Status: "FAILED", ObservedAt: day},
	}
	for _, observation := range observations {
		if _, err := st.RecordCostObservation(observation); err != nil {
			t.Fatal(err)
		}
	}
	view, err := st.SummarizeCostAnalytics(CostObservationFilter{
		ProjectID: "alpha", Since: day.Add(-7 * 24 * time.Hour),
	}, 25)
	if err != nil {
		t.Fatal(err)
	}
	total := view.Total
	if total.ObservationCount != 7 || total.InvocationsTotal != 7 {
		t.Fatalf("invocations_total must mirror the observation count: %+v", total)
	}
	// FAILED, FAILED_TIMEOUT and BLOCKED are failures; COMPLETED, CANCELED,
	// empty and UNKNOWN statuses are not.
	if total.InvocationsFailed != 3 {
		t.Fatalf("unexpected failure classification: %+v", total)
	}
	if total.ErrorRate == nil {
		t.Fatal("error_rate must exist when observations exist")
	}
	if want := 3.0 / 7.0; *total.ErrorRate != want {
		t.Fatalf("error_rate=%v, want %v", *total.ErrorRate, want)
	}
	// avg_duration_ms only counts rows with duration_ms > 0: (1000+2000)/2.
	if total.AvgDurationMS == nil || *total.AvgDurationMS != 1500 {
		t.Fatalf("avg_duration_ms=%v, want 1500 over measured rows only", total.AvgDurationMS)
	}
	// avg_ttft_ms only over NOT NULL evidence: (200+400)/2.
	if total.AvgTTFTMS == nil || *total.AvgTTFTMS != 300 {
		t.Fatalf("avg_ttft_ms=%v, want 300 over measured rows only", total.AvgTTFTMS)
	}
	if total.AvgOutputTokensPerSecond == nil || *total.AvgOutputTokensPerSecond != 7.5 {
		t.Fatalf("avg_output_tokens_per_second=%v, want 7.5 over measured rows only", total.AvgOutputTokensPerSecond)
	}
	if len(view.ByModel) != 2 {
		t.Fatalf("unexpected model split: %+v", view.ByModel)
	}
	modelOne := view.ByModel[0]
	if modelOne.Model != "model-one" || modelOne.InvocationsFailed != 2 ||
		modelOne.ErrorRate == nil || *modelOne.ErrorRate != 2.0/6.0 {
		t.Fatalf("per-model failure aggregation incorrect: %+v", modelOne)
	}
	modelTwo := view.ByModel[1]
	if modelTwo.Model != "model-two" || modelTwo.InvocationsFailed != 1 ||
		modelTwo.ErrorRate == nil || *modelTwo.ErrorRate != 1 {
		t.Fatalf("fully failed model must report rate 1: %+v", modelTwo)
	}
	// model-two measured no duration/ttft/throughput: omitted, never zero.
	if modelTwo.AvgDurationMS != nil || modelTwo.AvgTTFTMS != nil ||
		modelTwo.AvgOutputTokensPerSecond != nil {
		t.Fatalf("unmeasured evidence must stay omitted: %+v", modelTwo)
	}
	if len(view.ByDay) != 1 || view.ByDay[0].InvocationsFailed != 3 || view.ByDay[0].ErrorRate == nil {
		t.Fatalf("daily buckets must carry the same failure observability: %+v", view.ByDay)
	}
}
