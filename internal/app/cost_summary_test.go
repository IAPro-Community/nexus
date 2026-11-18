package app

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// costSummaryFixture records observations with heterogeneous failure status
// and partial latency/throughput evidence in the process-local store.
func costSummaryFixture(t *testing.T, st *store.Store) {
	t.Helper()
	now := time.Now().UTC()
	input, output := int64(300), int64(120)
	ttft := int64(250)
	tps := 6.5
	cost := int64(1500)
	rows := []store.CostObservation{
		{ProjectID: "alpha", Provider: "codex", Model: "model-one", CostKind: store.CostKindProviderReported, CostMicroUSD: &cost, InputTokens: &input, OutputTokens: &output, Source: "provider", Status: "COMPLETED", DurationMS: 1200, TTFTMS: &ttft, OutputTokensPerSecond: &tps, ObservedAt: now},
		{ProjectID: "alpha", Provider: "codex", Model: "model-one", CostKind: store.CostKindProviderReported, Source: "provider", Status: "FAILED", DurationMS: 400, ObservedAt: now.Add(time.Second)},
		{ProjectID: "alpha", Provider: "claude", Model: "model-two", CostKind: store.CostKindUnknown, Source: "runtime", Status: "UNKNOWN", ObservedAt: now.Add(2 * time.Second)},
	}
	for _, row := range rows {
		if _, err := st.RecordCostObservation(row); err != nil {
			t.Fatal(err)
		}
	}
}

// TestCostSummaryJSONMatchesEndpointSchema proves `nexus cost summary --json`
// emits the same schema as GET /api/v1/cost/analytics (CostAnalyticsReport)
// without duplicating any aggregation formula in the CLI layer.
func TestCostSummaryJSONMatchesEndpointSchema(t *testing.T) {
	nexus.ResetDefaultForTest()
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	t.Cleanup(nexus.ResetDefaultForTest)

	st := nexus.Default().Store()
	if st == nil {
		t.Fatal("missing canonical cost ledger")
	}
	costSummaryFixture(t, st)

	out, err := captureStdout(func() error {
		return costSummaryCmd([]string{"--days", "7", "--json"})
	})
	if err != nil {
		t.Fatal(err)
	}
	var report nexus.CostAnalyticsReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}
	if report.Source != "LOCAL_COST_OBSERVATIONS" || report.Days != 7 ||
		report.Total.ObservationCount != 3 {
		t.Fatalf("unexpected report envelope: %+v", report)
	}
	if report.Total.InvocationsTotal != 3 || report.Total.InvocationsFailed != 1 {
		t.Fatalf("failure observability missing from JSON output: %+v", report.Total)
	}
	if report.Total.ErrorRate == nil || *report.Total.ErrorRate != 1.0/3.0 {
		t.Fatalf("error_rate=%v, want 1/3; never fabricated 0", report.Total.ErrorRate)
	}
	if report.Total.AvgTTFTMS == nil || *report.Total.AvgTTFTMS != 250 {
		t.Fatalf("avg_ttft_ms=%v, want 250 over measured evidence only", report.Total.AvgTTFTMS)
	}
	if report.Total.AvgOutputTokensPerSecond == nil || *report.Total.AvgOutputTokensPerSecond != 6.5 {
		t.Fatalf("avg_output_tokens_per_second=%v, want 6.5", report.Total.AvgOutputTokensPerSecond)
	}
	if len(report.ByModel) != 2 {
		t.Fatalf("by_model must stay scoped: %+v", report.ByModel)
	}
}

// TestCostSummaryRejectsInvalidDays mirrors the HTTP handler validation: days
// outside 1..365 (or unparseable) are rejected with an error, never silently
// clamped to a different window.
func TestCostSummaryRejectsInvalidDays(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	for _, args := range [][]string{
		{"summary", "--days", "0"},
		{"summary", "--days", "366"},
		{"summary", "--days", "-3"},
		{"summary", "--days", "not-a-number"},
		{"summary", "--days=" + "0"},
		{"unknown"},
	} {
		if err := costCmd(args); err == nil {
			t.Fatalf("cost command unexpectedly accepted %v", args)
		}
	}
	for _, args := range [][]string{
		{"--days", "0"},
		{"--days", "366"},
		{"--days", "not-a-number"},
		{"--days"},
		{"--unknown-flag"},
	} {
		if err := costSummaryCmd(args); err == nil {
			t.Fatalf("cost summary unexpectedly accepted %v", args)
		}
	}
}

// TestCostSummaryHumanOutputKeepsUnknownEvidenceHonest checks that the human
// path renders the failure observability and omits unmeasured averages instead
// of printing synthetic zeros.
func TestCostSummaryHumanOutputKeepsUnknownEvidenceHonest(t *testing.T) {
	nexus.ResetDefaultForTest()
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	t.Cleanup(nexus.ResetDefaultForTest)

	st := nexus.Default().Store()
	if st == nil {
		t.Fatal("missing canonical cost ledger")
	}
	costSummaryFixture(t, st)

	out, err := captureStdout(func() error {
		return costSummaryCmd([]string{"--days", "7"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Observações:   3 (falhas: 1)") {
		t.Fatalf("human output lost invocation evidence: %s", out)
	}
	if !strings.Contains(out, "Taxa de erro:  33.3%") {
		t.Fatalf("human output lost error rate: %s", out)
	}
	if !strings.Contains(out, "TTFT médio") {
		t.Fatalf("human output lost measured TTFT evidence: %s", out)
	}
}
