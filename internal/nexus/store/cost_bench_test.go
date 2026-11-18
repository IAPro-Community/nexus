package store

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"
	"time"
)

// F2.5b performance baselines for the cost ledger (issue #228).
//
// Acceptance goal (informational, deliberately NOT a timing assert so CI hosts
// with different I/O do not flake): recording 10k synthetic observations and
// aggregating them with SummarizeCostAnalytics must each stay interactive
// (< 2s per run) on a dev host. Benchmarks only report; if a run blows the
// budget far past that, report the numbers first — no optimization without
// evidence.

const (
	costBenchObservations = 10000
	costBenchProviders    = 3
	costBenchModels       = 5
	costBenchProjects     = 4
	costBenchDays         = 365
)

var benchmarkCostProviderNames = []string{"codex", "claude", "gemini"}

func openBenchStore(b *testing.B) *Store {
	b.Helper()
	path := filepath.Join(b.TempDir(), "nexus.db")
	s, err := Open(path)
	if err != nil {
		b.Fatalf("open bench store: %v", err)
	}
	b.Cleanup(func() { _ = s.Close() })
	return s
}

// benchmarkCostObservation produces one deterministic synthetic observation
// (seeded math/rand) with varied token/cost evidence. Every 10th row has no
// invocation_id (legacy/free-form path, outside the dedupe index); the rest
// carry a per-iteration unique (source, invocation_id) key. Rows are spread
// across the last 365 days so the 90-day and 365-day analytics windows differ.
func benchmarkCostObservation(seq, iteration int, base time.Time, rng *rand.Rand) CostObservation {
	obs := CostObservation{
		ProjectID:    fmt.Sprintf("bench-project-%d", seq%costBenchProjects),
		InvocationID: fmt.Sprintf("bench-inv-%d-%06d", iteration, seq),
		Provider:     benchmarkCostProviderNames[seq%costBenchProviders],
		Profile:      "work",
		Model:        fmt.Sprintf("model-%d", seq%costBenchModels),
		InputTokens:  int64ptr(int64(100 + rng.Intn(5000))),
		OutputTokens: int64ptr(int64(10 + rng.Intn(2000))),
		DurationMS:   int64(100 + rng.Intn(5000)),
		CostMicroUSD: int64ptr(int64(1000 + rng.Intn(50000))),
		CostKind:     CostKindProviderReported,
		Source:       "provider_response",
		Status:       "COMPLETED",
		AccountScope: "bench-scope",
		ObservedAt:   base.Add(time.Duration(seq%costBenchDays) * 24 * time.Hour).Add(time.Duration(seq%1440) * time.Minute),
	}
	if seq%3 == 0 {
		obs.CachedInputTokens = int64ptr(int64(rng.Intn(500)))
	}
	if seq%7 == 0 {
		obs.ReasoningTokens = int64ptr(int64(rng.Intn(300)))
	}
	if seq%10 == 9 {
		obs.InvocationID = ""
		obs.Source = "runtime_observation"
		obs.CostKind = CostKindUnknown
		obs.CostMicroUSD = nil
		obs.Status = ""
	}
	return obs
}

func seedBenchmarkCostObservations(b *testing.B, s *Store) {
	b.Helper()
	rng := rand.New(rand.NewSource(42))
	base := time.Now().UTC().Add(-costBenchDays * 24 * time.Hour)
	for seq := 0; seq < costBenchObservations; seq++ {
		if _, err := s.RecordCostObservation(benchmarkCostObservation(seq, 0, base, rng)); err != nil {
			b.Fatalf("seed observation %d: %v", seq, err)
		}
	}
}

// BenchmarkCostRecordObservations measures raw ledger throughput: 10k
// RecordCostObservation calls (one transaction per row, INSERT OR IGNORE on
// the (source, invocation_id) partial unique index for rows that carry it).
//
// Acceptance goal (informational): < 2s per 10k-row run on a dev host.
func BenchmarkCostRecordObservations(b *testing.B) {
	s := openBenchStore(b)
	rng := rand.New(rand.NewSource(42))
	base := time.Now().UTC().Add(-costBenchDays * 24 * time.Hour)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for seq := 0; seq < costBenchObservations; seq++ {
			if _, err := s.RecordCostObservation(benchmarkCostObservation(seq, i, base, rng)); err != nil {
				b.Fatalf("record observation %d: %v", seq, err)
			}
		}
	}
}

func BenchmarkCostSummarizeAnalytics90Days(b *testing.B) {
	s := openBenchStore(b)
	seedBenchmarkCostObservations(b, s)
	filter := CostObservationFilter{Since: time.Now().UTC().Add(-90 * 24 * time.Hour)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.SummarizeCostAnalytics(filter, 25); err != nil {
			b.Fatalf("summarize cost analytics 90d: %v", err)
		}
	}
}

func BenchmarkCostSummarizeAnalytics365Days(b *testing.B) {
	s := openBenchStore(b)
	seedBenchmarkCostObservations(b, s)
	filter := CostObservationFilter{Since: time.Now().UTC().Add(-costBenchDays * 24 * time.Hour)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.SummarizeCostAnalytics(filter, 25); err != nil {
			b.Fatalf("summarize cost analytics 365d: %v", err)
		}
	}
}

// BenchmarkCostListObservations measures the filtered listing path on the 10k
// ledger: provider filter + limit 100 over the (provider, profile,
// observed_at DESC) index.
func BenchmarkCostListObservations(b *testing.B) {
	s := openBenchStore(b)
	seedBenchmarkCostObservations(b, s)
	filter := CostObservationFilter{Provider: "codex", Limit: 100}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := s.ListCostObservations(filter)
		if err != nil {
			b.Fatalf("list cost observations: %v", err)
		}
		if len(rows) != 100 {
			b.Fatalf("rows=%d, want 100", len(rows))
		}
	}
}

// expectedCostTotals accumulates the exact evidence sums of the synthetic
// reconciliation dataset while it is generated, per project and globally.
type expectedCostTotals struct {
	rows               int64
	unknownCostRows    int64
	knownCostRows      int64
	knownTokenRows     int64
	costMicroUSD       int64
	inputTokens        int64
	cachedInputTokens  int64
	outputTokens       int64
	reasoningTokens    int64
	durationMS         int64
	reportedMicroUSD   int64
	equivalentMicroUSD int64
	inferredMicroUSD   int64
	ttftRows           int64
	ttftTotalMS        int64
}

func (e *expectedCostTotals) add(obs CostObservation) {
	e.rows++
	if obs.CostMicroUSD != nil {
		e.knownCostRows++
		e.costMicroUSD += *obs.CostMicroUSD
		switch obs.CostKind {
		case CostKindProviderReported:
			e.reportedMicroUSD += *obs.CostMicroUSD
		case CostKindAPIEquivalentEstimate:
			e.equivalentMicroUSD += *obs.CostMicroUSD
		case CostKindInferred:
			e.inferredMicroUSD += *obs.CostMicroUSD
		}
	} else {
		e.unknownCostRows++
	}
	if obs.InputTokens != nil {
		e.inputTokens += *obs.InputTokens
	}
	if obs.CachedInputTokens != nil {
		e.cachedInputTokens += *obs.CachedInputTokens
	}
	if obs.OutputTokens != nil {
		e.outputTokens += *obs.OutputTokens
	}
	if obs.ReasoningTokens != nil {
		e.reasoningTokens += *obs.ReasoningTokens
	}
	if obs.InputTokens != nil || obs.CachedInputTokens != nil ||
		obs.OutputTokens != nil || obs.ReasoningTokens != nil {
		e.knownTokenRows++
	}
	e.durationMS += obs.DurationMS
	if obs.TTFTMS != nil {
		e.ttftRows++
		e.ttftTotalMS += *obs.TTFTMS
	}
}

// reconciliationCostObservation is a pure-formula synthetic observation so
// every token/cost sum below is exactly predictable (no floating-point drift,
// no randomness). i%5==0 rows are UNKNOWN evidence with no cost (cost stays
// nil forever — UNKNOWN must never become a billable number); the rest cycle
// through the three cost-evidence classes.
func reconciliationCostObservation(i int, base time.Time) CostObservation {
	obs := CostObservation{
		ProjectID:    map[bool]string{true: "recon-alpha", false: "recon-beta"}[i%2 == 0],
		Provider:     benchmarkCostProviderNames[i%costBenchProviders],
		Profile:      "work",
		Model:        fmt.Sprintf("model-%d", i%costBenchModels),
		InputTokens:  int64ptr(int64(100 + i)),
		OutputTokens: int64ptr(int64(20 + i%50)),
		DurationMS:   int64(500 + i%1000),
		Source:       "provider_response",
		Status:       "COMPLETED",
		ObservedAt:   base.Add(time.Duration(i%20) * 24 * time.Hour).Add(time.Duration(i%1440) * time.Minute),
	}
	if i%3 == 0 {
		obs.CachedInputTokens = int64ptr(int64(5 + i%9))
	}
	if i%7 == 0 {
		obs.ReasoningTokens = int64ptr(int64(3 + i%11))
	}
	if i%4 == 0 {
		obs.TTFTMS = int64ptr(int64(100 + i%200))
	}
	if i%5 == 0 {
		obs.CostKind = CostKindUnknown
		obs.Source = "runtime_observation"
		obs.Status = ""
	} else {
		cost := int64(1000 + i)
		switch i % 3 {
		case 0:
			obs.CostKind = CostKindProviderReported
		case 1:
			obs.CostKind = CostKindAPIEquivalentEstimate
		default:
			obs.CostKind = CostKindInferred
		}
		obs.CostMicroUSD = &cost
	}
	if i%10 == 9 {
		obs.InvocationID = ""
	} else {
		obs.InvocationID = fmt.Sprintf("recon-inv-%06d", i)
	}
	return obs
}

// TestCostLedgerTenThousandObservationReconciliation writes 10k synthetic
// observations with known per-project sums and validates that the analytics
// and plain aggregates reconcile EXACTLY: token/cost totals per project,
// sum-of-projects equals the global aggregate, by_day has the right number of
// days, invocations_total mirrors the persisted rows even after replays, and
// UNKNOWN evidence never enters any cost bucket.
func TestCostLedgerTenThousandObservationReconciliation(t *testing.T) {
	st := openTestStore(t)
	base := time.Now().UTC().Add(-30 * 24 * time.Hour).Truncate(24 * time.Hour)

	expected := map[string]*expectedCostTotals{
		"recon-alpha": {},
		"recon-beta":  {},
	}
	global := &expectedCostTotals{}
	invocationSamples := make([]CostObservation, 0, 25)

	for i := 0; i < costBenchObservations; i++ {
		obs := reconciliationCostObservation(i, base)
		if _, err := st.RecordCostObservation(obs); err != nil {
			t.Fatalf("record reconciliation observation %d: %v", i, err)
		}
		expected[obs.ProjectID].add(obs)
		global.add(obs)
		if obs.InvocationID != "" && len(invocationSamples) < 25 {
			invocationSamples = append(invocationSamples, obs)
		}
	}

	// Replay the sampled (source, invocation_id) keys: identical evidence is
	// idempotent and must not move any total; volatile-only divergence
	// (status) returns the committed row unchanged.
	for i, sample := range invocationSamples {
		replay := sample
		replay.ID = fmt.Sprintf("recon-replay-%d", i)
		replay.Status = "RUNNING"
		if _, err := st.RecordCostObservation(replay); err != nil {
			t.Fatalf("idempotent replay %d: %v", i, err)
		}
	}

	verifyTotals := func(scope string, want *expectedCostTotals, totals CostAnalyticsTotals) {
		t.Helper()
		if totals.ObservationCount != want.rows || totals.InvocationsTotal != want.rows {
			t.Fatalf("%s: observation/invocation count=%d/%d, want %d", scope, totals.ObservationCount, totals.InvocationsTotal, want.rows)
		}
		if totals.KnownCostObservations != want.knownCostRows || totals.UnknownCostObservations != want.unknownCostRows {
			t.Fatalf("%s: known/unknown cost rows=%d/%d, want %d/%d", scope, totals.KnownCostObservations, totals.UnknownCostObservations, want.knownCostRows, want.unknownCostRows)
		}
		if totals.KnownTokenObservations != want.knownTokenRows {
			t.Fatalf("%s: known token rows=%d, want %d", scope, totals.KnownTokenObservations, want.knownTokenRows)
		}
		// CostAnalyticsTotals never merges evidence classes: the only honest
		// "total cost" is the sum of the three known classes.
		knownCost := totals.ProviderReportedMicroUSD + totals.APIEquivalentMicroUSD + totals.InferredMicroUSD
		if knownCost != want.costMicroUSD {
			t.Fatalf("%s: cost evidence buckets sum=%d, want exactly %d", scope, knownCost, want.costMicroUSD)
		}
		if totals.InputTokens != want.inputTokens || totals.CachedInputTokens != want.cachedInputTokens ||
			totals.OutputTokens != want.outputTokens || totals.ReasoningTokens != want.reasoningTokens {
			t.Fatalf("%s: token totals drifted: got in=%d cached=%d out=%d reasoning=%d", scope,
				totals.InputTokens, totals.CachedInputTokens, totals.OutputTokens, totals.ReasoningTokens)
		}
		if totals.DurationMS != want.durationMS {
			t.Fatalf("%s: duration_ms=%d, want exactly %d", scope, totals.DurationMS, want.durationMS)
		}
		if totals.ProviderReportedMicroUSD != want.reportedMicroUSD ||
			totals.APIEquivalentMicroUSD != want.equivalentMicroUSD ||
			totals.InferredMicroUSD != want.inferredMicroUSD {
			t.Fatalf("%s: cost evidence buckets drifted: reported=%d equivalent=%d inferred=%d", scope,
				totals.ProviderReportedMicroUSD, totals.APIEquivalentMicroUSD, totals.InferredMicroUSD)
		}
		if totals.UnclassifiedMicroUSD != 0 {
			t.Fatalf("%s: UNKNOWN evidence leaked %d microusd into the bill", scope, totals.UnclassifiedMicroUSD)
		}
		if want.ttftRows > 0 {
			if totals.AvgTTFTMS == nil || *totals.AvgTTFTMS != float64(want.ttftTotalMS)/float64(want.ttftRows) {
				t.Fatalf("%s: avg_ttft_ms=%v, want exactly %f", scope, totals.AvgTTFTMS, float64(want.ttftTotalMS)/float64(want.ttftRows))
			}
		}
		if totals.AvgDurationMS == nil || *totals.AvgDurationMS != float64(want.durationMS)/float64(want.rows) {
			t.Fatalf("%s: avg_duration_ms=%v, want exactly %f", scope, totals.AvgDurationMS, float64(want.durationMS)/float64(want.rows))
		}
	}

	// Plain aggregate reconciliation (SummarizeCostObservations path).
	for projectID, want := range expected {
		agg, err := st.SummarizeCostObservations(CostObservationFilter{ProjectID: projectID})
		if err != nil {
			t.Fatal(err)
		}
		if agg.ObservationCount != want.rows || agg.CostMicroUSD != want.costMicroUSD ||
			agg.InputTokens != want.inputTokens || agg.OutputTokens != want.outputTokens ||
			agg.CachedInputTokens != want.cachedInputTokens || agg.ReasoningTokens != want.reasoningTokens ||
			agg.DurationMS != want.durationMS {
			t.Fatalf("%s plain aggregate drifted: %+v", projectID, agg)
		}
	}

	// Analytics view reconciliation per project and globally.
	alphaView, err := st.SummarizeCostAnalytics(CostObservationFilter{ProjectID: "recon-alpha"}, 50)
	if err != nil {
		t.Fatal(err)
	}
	betaView, err := st.SummarizeCostAnalytics(CostObservationFilter{ProjectID: "recon-beta"}, 50)
	if err != nil {
		t.Fatal(err)
	}
	globalView, err := st.SummarizeCostAnalytics(CostObservationFilter{Since: base}, 50)
	if err != nil {
		t.Fatal(err)
	}
	verifyTotals("recon-alpha", expected["recon-alpha"], alphaView.Total)
	verifyTotals("recon-beta", expected["recon-beta"], betaView.Total)
	verifyTotals("global", global, globalView.Total)

	// Sum of separated per-project views equals the global aggregate.
	alphaCost := alphaView.Total.ProviderReportedMicroUSD + alphaView.Total.APIEquivalentMicroUSD + alphaView.Total.InferredMicroUSD
	betaCost := betaView.Total.ProviderReportedMicroUSD + betaView.Total.APIEquivalentMicroUSD + betaView.Total.InferredMicroUSD
	globalCost := globalView.Total.ProviderReportedMicroUSD + globalView.Total.APIEquivalentMicroUSD + globalView.Total.InferredMicroUSD
	if alphaView.Total.ObservationCount+betaView.Total.ObservationCount != globalView.Total.ObservationCount ||
		alphaCost+betaCost != globalCost ||
		alphaView.Total.InvocationsTotal+betaView.Total.InvocationsTotal != globalView.Total.InvocationsTotal ||
		alphaView.Total.InputTokens+betaView.Total.InputTokens != globalView.Total.InputTokens {
		t.Fatalf("per-project totals do not reconcile to the global aggregate: alpha=%+v beta=%+v global=%+v",
			alphaView.Total, betaView.Total, globalView.Total)
	}

	// by_day: exactly 20 distinct days in the window, ascending, summing back
	// to the global observation count.
	if len(globalView.ByDay) != 20 {
		t.Fatalf("by_day entries=%d, want 20", len(globalView.ByDay))
	}
	var dayRows int64
	for i, day := range globalView.ByDay {
		if i > 0 && day.Date <= globalView.ByDay[i-1].Date {
			t.Fatalf("by_day not sorted ascending: %v", globalView.ByDay)
		}
		dayRows += day.ObservationCount
	}
	if dayRows != globalView.Total.ObservationCount {
		t.Fatalf("by_day rows=%d, want %d", dayRows, globalView.Total.ObservationCount)
	}

	// by_model: 3 providers x 5 models = 15 provider/model rows, no
	// truncation, summing back to the same total.
	if len(globalView.ByModel) != 15 || globalView.ModelsTruncated {
		t.Fatalf("by_model entries=%d truncated=%v, want 15 provider/model rows without truncation", len(globalView.ByModel), globalView.ModelsTruncated)
	}
	var modelRows int64
	for _, model := range globalView.ByModel {
		modelRows += model.ObservationCount
	}
	if modelRows != globalView.Total.ObservationCount {
		t.Fatalf("by_model rows=%d, want %d", modelRows, globalView.Total.ObservationCount)
	}

	// Post-replay listing: replays must not have created extra rows.
	rows, err := st.ListCostObservations(CostObservationFilter{Limit: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(rows)) != 1000 || rows[0].ObservedAt.Before(rows[len(rows)-1].ObservedAt) {
		t.Fatalf("first page must hold 1000 rows in observed_at DESC order: got %d rows", len(rows))
	}
}

// TestCostLedgerTenThousandObservationReconciliationUnderBudget is the F2.8
// regression budget (issue #228): the full 10k-observation reconciliation —
// writing every row (one transaction per row, INSERT OR IGNORE on the
// (source, invocation_id) partial unique index), replaying the idempotency
// samples, and running both the analytics and plain aggregate views — must
// finish under a single wall-clock budget.
//
// This is deliberately NOT a microbenchmark: the 60s limit is a generous,
// CI-stable guard against catastrophic regressions (an accidental O(n²)
// rewrite loop, a lost index turning aggregation into full scans per row, or
// per-row fsync churn), not a target to optimize against. On a dev host the
// reconciliation test above completes in roughly 1-2s; a 30x headroom keeps
// the guard stable on slower CI runners while still failing loudly if the
// ledger collapses to pathological behavior.
func TestCostLedgerTenThousandObservationReconciliationUnderBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("budget guard skipped in -short mode")
	}
	const budget = 60 * time.Second

	st := openTestStore(t)
	base := time.Now().UTC().Add(-30 * 24 * time.Hour).Truncate(24 * time.Hour)

	start := time.Now()

	for i := 0; i < costBenchObservations; i++ {
		if _, err := st.RecordCostObservation(reconciliationCostObservation(i, base)); err != nil {
			t.Fatalf("record reconciliation observation %d: %v", i, err)
		}
	}
	for i := 0; i < 25; i++ {
		// i*10+1 keeps every sample off the i%10==9 legacy/no-invocation rows.
		replay := reconciliationCostObservation(i*10+1, base)
		if replay.InvocationID == "" {
			t.Fatalf("sample %d must carry an invocation id", i)
		}
		replay.Status = "RUNNING"
		if _, err := st.RecordCostObservation(replay); err != nil {
			t.Fatalf("idempotent replay %d: %v", i, err)
		}
	}
	if _, err := st.SummarizeCostAnalytics(CostObservationFilter{Since: base}, 50); err != nil {
		t.Fatal(err)
	}
	if _, err := st.SummarizeCostObservations(CostObservationFilter{}); err != nil {
		t.Fatal(err)
	}

	elapsed := time.Since(start)
	if elapsed > budget {
		t.Fatalf("10k-observation reconciliation took %s, over the %s regression budget (write + replay + aggregation)", elapsed, budget)
	}
	t.Logf("10k-observation reconciliation: write + replay + aggregation in %s (budget %s)", elapsed, budget)
}
