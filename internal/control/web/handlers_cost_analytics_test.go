package web

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestCostAnalyticsEndpointUsesAuthenticatedLedgerAndNeverMixesEstimates(t *testing.T) {
	nexus.ResetDefaultForTest()
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	t.Cleanup(nexus.ResetDefaultForTest)
	client, srv, _ := csrfClient(t)
	st := nexus.Default().Store()
	if st == nil {
		t.Fatal("missing canonical cost ledger")
	}
	known, equivalent := int64(1000), int64(2000)
	now := time.Now().UTC()
	for _, row := range []store.CostObservation{
		{ProjectID: "alpha", Model: "model-a", Provider: "codex", CostKind: store.CostKindProviderReported, CostMicroUSD: &known, Source: "provider", ObservedAt: now},
		{ProjectID: "alpha", Model: "model-a", Provider: "codex", CostKind: store.CostKindAPIEquivalentEstimate, CostMicroUSD: &equivalent, Source: "ratecard", ObservedAt: now},
		{ProjectID: "alpha", Model: "unknown-model", Provider: "claude", CostKind: store.CostKindUnknown, Source: "runtime", ObservedAt: now},
		{ProjectID: "beta", Model: "foreign-project", Provider: "codex", CostKind: store.CostKindProviderReported, CostMicroUSD: &equivalent, Source: "provider", ObservedAt: now},
	} {
		if _, err := st.RecordCostObservation(row); err != nil {
			t.Fatal(err)
		}
	}
	resp, err := client.Get(srv.URL() + "/api/v1/cost/analytics?days=7&project_id=alpha")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("invalid analytics HTTP response: %d", resp.StatusCode)
	}
	var report nexus.CostAnalyticsReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	if report.Source != "LOCAL_COST_OBSERVATIONS" || report.ProjectID != "alpha" ||
		report.Days != 7 || report.Total.ObservationCount != 3 {
		t.Fatalf("unexpected scoped report: %+v", report)
	}
	if report.Total.ProviderReportedMicroUSD != known ||
		report.Total.APIEquivalentMicroUSD != equivalent ||
		report.Total.UnknownCostObservations != 1 ||
		report.Total.KnownCostObservations != 2 ||
		len(report.ByModel) != 2 {
		t.Fatalf("cost evidence mixed or incomplete: %+v", report)
	}
	for _, rawDays := range []string{"0", "-1", "366", "not-a-number"} {
		res, err := client.Get(srv.URL() + "/api/v1/cost/analytics?days=" + rawDays)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("days=%q status=%d, expected 400", rawDays, res.StatusCode)
		}
	}
	post, err := client.Post(srv.URL()+"/api/v1/cost/analytics", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	post.Body.Close()
	if post.StatusCode != http.StatusForbidden && post.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("analytics POST must never modify the ledger: status=%d", post.StatusCode)
	}
}

// TestCostAnalyticsEndpointExposesFailureObservability is additive (F2.5a):
// the same endpoint schema now carries invocation failure counts, error rate
// and evidence-backed averages, per period and per model, without changing the
// legacy fields.
func TestCostAnalyticsEndpointExposesFailureObservability(t *testing.T) {
	nexus.ResetDefaultForTest()
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	t.Cleanup(nexus.ResetDefaultForTest)
	client, srv, _ := csrfClient(t)
	st := nexus.Default().Store()
	if st == nil {
		t.Fatal("missing canonical cost ledger")
	}
	ttft := int64(300)
	tps := 4.5
	now := time.Now().UTC()
	for _, row := range []store.CostObservation{
		{ProjectID: "alpha", Model: "model-a", Provider: "codex", CostKind: store.CostKindProviderReported, Source: "provider", Status: "COMPLETED", DurationMS: 900, TTFTMS: &ttft, OutputTokensPerSecond: &tps, ObservedAt: now},
		{ProjectID: "alpha", Model: "model-a", Provider: "codex", CostKind: store.CostKindProviderReported, Source: "provider", Status: "FAILED_TIMEOUT", DurationMS: 300, ObservedAt: now.Add(time.Second)},
		{ProjectID: "alpha", Model: "model-b", Provider: "claude", CostKind: store.CostKindUnknown, Source: "runtime", Status: "UNKNOWN", ObservedAt: now.Add(2 * time.Second)},
	} {
		if _, err := st.RecordCostObservation(row); err != nil {
			t.Fatal(err)
		}
	}
	resp, err := client.Get(srv.URL() + "/api/v1/cost/analytics?days=7&project_id=alpha")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var report nexus.CostAnalyticsReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	if report.Total.ObservationCount != 3 || report.Total.InvocationsTotal != 3 {
		t.Fatalf("invocations_total must mirror observation_count: %+v", report.Total)
	}
	// FAILED_TIMEOUT is a failure; UNKNOWN is explicitly a non-failure.
	if report.Total.InvocationsFailed != 1 || report.Total.ErrorRate == nil ||
		*report.Total.ErrorRate != 1.0/3.0 {
		t.Fatalf("unexpected failure classification: %+v", report.Total)
	}
	if report.Total.AvgDurationMS == nil || *report.Total.AvgDurationMS != 600 {
		t.Fatalf("avg_duration_ms=%v, want 600 over measured rows only", report.Total.AvgDurationMS)
	}
	if report.Total.AvgTTFTMS == nil || *report.Total.AvgTTFTMS != 300 {
		t.Fatalf("avg_ttft_ms=%v, want 300", report.Total.AvgTTFTMS)
	}
	if report.Total.AvgOutputTokensPerSecond == nil || *report.Total.AvgOutputTokensPerSecond != 4.5 {
		t.Fatalf("avg_output_tokens_per_second=%v, want 4.5", report.Total.AvgOutputTokensPerSecond)
	}
	if len(report.ByModel) != 2 {
		t.Fatalf("unexpected model split: %+v", report.ByModel)
	}
	failed := report.ByModel[0]
	if failed.Model != "model-a" || failed.InvocationsFailed != 1 ||
		failed.ErrorRate == nil || *failed.ErrorRate != 0.5 {
		t.Fatalf("per-model failure fields missing: %+v", failed)
	}
	unknown := report.ByModel[1]
	if unknown.Model != "model-b" || unknown.InvocationsFailed != 0 ||
		unknown.ErrorRate == nil || *unknown.ErrorRate != 0 ||
		unknown.AvgTTFTMS != nil || unknown.AvgOutputTokensPerSecond != nil {
		t.Fatalf("UNKNOWN status must stay a non-failure with no synthetic averages: %+v", unknown)
	}
	if len(report.ByDay) != 1 || report.ByDay[0].InvocationsFailed != 1 {
		t.Fatalf("daily buckets must carry failure observability: %+v", report.ByDay)
	}
}
