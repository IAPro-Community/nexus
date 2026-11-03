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
