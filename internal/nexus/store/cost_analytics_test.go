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
	if evidence != view.Total || evidence.ReportedCostObservations != 1 ||
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
}
