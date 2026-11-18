package store

import (
	"errors"
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

// TestCostAnalyticsMultiProjectIsolation pins the multi-project isolation
// contract of the cost ledger (F2.5b).
//
// Dedupe-key collision analysis (migration 0039): the partial unique index is
// UNIQUE(source, invocation_id) — GLOBAL, not scoped by project_id. Two
// observations from different projects with the same (source, invocation_id)
// therefore collide. That is intentional for the current producers:
//   - MAESTRO_SUBMISSION uses the Maestro submissionId, which is semantically
//     global — the same submission must never be counted twice, even if a
//     replay targets another project.
//   - runtime_observation uses goal_<UnixNano>, globally unique in practice.
//
// The collision surfaces as ErrCostObservationConflict (the immutable-evidence
// comparison includes project_id), never as a silent overwrite or silent drop.
// Latent risk (documented, NOT fixed here — schema changes belong to another
// batch): a future source that mints per-project invocation ids (e.g.
// "inv-1" inside every project) would legitimately collide across projects.
// Known weaker spot: mission_execution.go ignores RecordCostObservation
// errors (`_, _ =`), so a same-nanosecond goal_ collision would be dropped
// without a conflict surfacing to the caller.
func TestCostAnalyticsMultiProjectIsolation(t *testing.T) {
	st := openTestStore(t)
	base := time.Now().UTC().Add(-24 * time.Hour).Truncate(24 * time.Hour).Add(12 * time.Hour)
	reported, input, output := int64(4000), int64(300), int64(40)

	// Alpha and beta deliberately share provider/model values; invocation ids
	// differ (shared keys collide by design — see the test comment above).
	observations := []CostObservation{
		{ProjectID: "alpha", Provider: "codex", Profile: "work", Model: "model-one", InvocationID: "alpha-inv-1", CostKind: CostKindProviderReported, CostMicroUSD: &reported, InputTokens: &input, OutputTokens: &output, Source: "provider", Status: "COMPLETED", DurationMS: 900, ObservedAt: base},
		{ProjectID: "alpha", Provider: "codex", Profile: "work", Model: "model-one", InvocationID: "alpha-inv-2", CostKind: CostKindProviderReported, CostMicroUSD: &reported, InputTokens: &input, OutputTokens: &output, Source: "provider", Status: "COMPLETED", DurationMS: 900, ObservedAt: base.Add(time.Minute)},
		{ProjectID: "alpha", Provider: "claude", Profile: "work", Model: "model-two", InvocationID: "alpha-inv-3", CostKind: CostKindUnknown, Source: "runtime_observation", Status: "COMPLETED", DurationMS: 900, ObservedAt: base.Add(2 * time.Minute)},
		{ProjectID: "beta", Provider: "codex", Profile: "work", Model: "model-one", InvocationID: "beta-inv-1", CostKind: CostKindProviderReported, CostMicroUSD: &reported, InputTokens: &input, OutputTokens: &output, Source: "provider", Status: "COMPLETED", DurationMS: 900, ObservedAt: base.Add(3 * time.Minute)},
		{ProjectID: "beta", Provider: "codex", Profile: "work", Model: "model-one", InvocationID: "beta-inv-2", CostKind: CostKindProviderReported, CostMicroUSD: &reported, InputTokens: &input, OutputTokens: &output, Source: "provider", Status: "COMPLETED", DurationMS: 900, ObservedAt: base.Add(4 * time.Minute)},
		{ProjectID: "beta", Provider: "claude", Profile: "work", Model: "model-two", InvocationID: "beta-inv-3", CostKind: CostKindUnknown, Source: "runtime_observation", Status: "COMPLETED", DurationMS: 900, ObservedAt: base.Add(5 * time.Minute)},
	}
	for _, observation := range observations {
		if _, err := st.RecordCostObservation(observation); err != nil {
			t.Fatal(err)
		}
	}

	// Same (source, invocation_id) across projects collides: the 0039 index is
	// global and project_id is immutable evidence, so the replay from beta
	// against alpha's committed row must surface as a conflict — the ledger
	// never silently overwrites or duplicates it.
	crossProject := observations[0]
	crossProject.ProjectID = "beta"
	crossProject.ID = "cost-cross-project-probe"
	if _, err := st.RecordCostObservation(crossProject); !errors.Is(err, ErrCostObservationConflict) {
		t.Fatalf("err=%v, want ErrCostObservationConflict for same (source, invocation_id) in another project", err)
	}

	// Every project-scoped query returns only that project's observations, in
	// stable observed_at DESC order (latest invocation first).
	for projectID, projectRows := range map[string][]string{
		"alpha": {"alpha-inv-3", "alpha-inv-2", "alpha-inv-1"},
		"beta":  {"beta-inv-3", "beta-inv-2", "beta-inv-1"},
	} {
		rows, err := st.ListCostObservations(CostObservationFilter{ProjectID: projectID, Limit: 100})
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != len(projectRows) {
			t.Fatalf("%s rows=%d, want %d", projectID, len(rows), len(projectRows))
		}
		for i, row := range rows {
			if row.ProjectID != projectID {
				t.Fatalf("%s filter leaked row from %q: %+v", projectID, row.ProjectID, row)
			}
			if row.InvocationID != projectRows[i] {
				t.Fatalf("%s ordering not stable by observed_at DESC: got %q at %d, want %q", projectID, row.InvocationID, i, projectRows[i])
			}
		}

		view, err := st.SummarizeCostAnalytics(CostObservationFilter{ProjectID: projectID, Since: base.Add(-time.Hour)}, 25)
		if err != nil {
			t.Fatal(err)
		}
		viewCost := view.Total.ProviderReportedMicroUSD + view.Total.APIEquivalentMicroUSD + view.Total.InferredMicroUSD
		if view.Total.ObservationCount != 3 || viewCost != 2*reported ||
			view.Total.InputTokens != 2*input || view.Total.OutputTokens != 2*output {
			t.Fatalf("%s analytics totals drifted: %+v", projectID, view.Total)
		}
		if view.Total.KnownCostObservations != 2 || view.Total.UnknownCostObservations != 1 {
			t.Fatalf("%s evidence class split drifted: %+v", projectID, view.Total)
		}
		if view.Total.UnclassifiedMicroUSD != 0 {
			t.Fatalf("%s: UNKNOWN evidence leaked into the bill: %+v", projectID, view.Total)
		}
		if len(view.ByModel) != 2 {
			t.Fatalf("%s by_model rows=%d, want 2", projectID, len(view.ByModel))
		}

		providerRows, err := st.SummarizeCostByProvider(CostObservationFilter{ProjectID: projectID})
		if err != nil {
			t.Fatal(err)
		}
		if len(providerRows) != 2 || providerRows[0].Provider != "codex" || providerRows[0].CostMicroUSD != 2*reported {
			t.Fatalf("%s provider aggregation drifted: %+v", projectID, providerRows)
		}
	}

	// Separate sums equal the global sum; the conflicting cross-project probe
	// must not have added any row.
	global, err := st.SummarizeCostObservations(CostObservationFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if global.ObservationCount != 6 || global.CostMicroUSD != 4*reported {
		t.Fatalf("global aggregate drifted (cross-project probe must be rejected): %+v", global)
	}
	perProject, err := st.SummarizeCostObservations(CostObservationFilter{ProjectID: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	betaAggregate, err := st.SummarizeCostObservations(CostObservationFilter{ProjectID: "beta"})
	if err != nil {
		t.Fatal(err)
	}
	if perProject.ObservationCount+betaAggregate.ObservationCount != global.ObservationCount ||
		perProject.CostMicroUSD+betaAggregate.CostMicroUSD != global.CostMicroUSD ||
		perProject.InputTokens+betaAggregate.InputTokens != global.InputTokens {
		t.Fatalf("per-project sums=%d/%d do not reconcile to global=%d", perProject.CostMicroUSD, betaAggregate.CostMicroUSD, global.CostMicroUSD)
	}

	// Pagination over a project filter is stable: a Since-bounded window
	// (observed_at >= since, the filter's documented inclusive semantics)
	// returns the same relative observed_at DESC order and never mixes
	// projects.
	firstPage, err := st.ListCostObservations(CostObservationFilter{ProjectID: "alpha", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(firstPage) != 2 || firstPage[0].InvocationID != "alpha-inv-3" || firstPage[1].InvocationID != "alpha-inv-2" {
		t.Fatalf("alpha first page unstable: %+v", firstPage)
	}
	windowPage, err := st.ListCostObservations(CostObservationFilter{ProjectID: "alpha", Since: firstPage[1].ObservedAt, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(windowPage) != 2 || windowPage[0].InvocationID != "alpha-inv-3" || windowPage[1].InvocationID != "alpha-inv-2" {
		t.Fatalf("alpha Since-window page unstable: %+v", windowPage)
	}
	for _, row := range windowPage {
		if row.ProjectID != "alpha" {
			t.Fatalf("window page leaked project %q: %+v", row.ProjectID, row)
		}
	}
	fullPage, err := st.ListCostObservations(CostObservationFilter{ProjectID: "alpha", Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(fullPage) != 3 || fullPage[2].InvocationID != "alpha-inv-1" {
		t.Fatalf("alpha full page unstable: %+v", fullPage)
	}
}
