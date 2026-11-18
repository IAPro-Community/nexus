package app

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// costAnalyticsFixture reuses the summary fixture and adds one observation in
// another project so --project-id filtering is actually exercised.
func costAnalyticsFixture(t *testing.T, st *store.Store) {
	t.Helper()
	costSummaryFixture(t, st)
	cost := int64(700)
	row := store.CostObservation{
		ProjectID: "beta", Provider: "agy", Model: "model-three",
		CostKind: store.CostKindAPIEquivalentEstimate, CostMicroUSD: &cost,
		Source: "runtime", Status: "COMPLETED", ObservedAt: time.Now().UTC(),
	}
	if _, err := st.RecordCostObservation(row); err != nil {
		t.Fatal(err)
	}
}

// TestCostAnalyticsJSONMatchesEndpointSchema proves `nexus cost analytics
// --json` emits the exact CostAnalyticsReport the web endpoint returns for the
// same ledger fixture. The CLI output is compared against a direct
// CostApplicationService.Analytics call (the endpoint's single source), with
// only the wall-clock fields (since/generated_at) normalized between the two
// invocations. Struct equality over the JSON-decoded report implies schema
// equality with the endpoint.
func TestCostAnalyticsJSONMatchesEndpointSchema(t *testing.T) {
	nexus.ResetDefaultForTest()
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	t.Cleanup(nexus.ResetDefaultForTest)

	st := nexus.Default().Store()
	if st == nil {
		t.Fatal("missing canonical cost ledger")
	}
	costAnalyticsFixture(t, st)

	out, err := captureStdout(func() error {
		return costAnalyticsCmd([]string{"--days", "7", "--project-id", "alpha", "--json"})
	})
	if err != nil {
		t.Fatal(err)
	}

	var fromCLI nexus.CostAnalyticsReport
	if err := json.Unmarshal([]byte(out), &fromCLI); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}
	if fromCLI.ProjectID != "alpha" {
		t.Fatalf("--project-id filter lost from JSON output: %q", fromCLI.ProjectID)
	}
	if fromCLI.Source != "LOCAL_COST_OBSERVATIONS" || fromCLI.Days != 7 ||
		fromCLI.Total.ObservationCount != 3 {
		t.Fatalf("unexpected report envelope: %+v", fromCLI)
	}

	// Same source the endpoint serves, so schema is identical by construction.
	svc := nexus.NewCostApplicationService(st)
	fromService, err := svc.Analytics(context.Background(), "alpha", 7)
	if err != nil {
		t.Fatal(err)
	}
	fromCLI.Since, fromCLI.GeneratedAt = time.Time{}, time.Time{}
	fromService.Since, fromService.GeneratedAt = time.Time{}, time.Time{}
	if !reflect.DeepEqual(fromCLI, fromService) {
		t.Fatalf("CLI analytics JSON diverges from endpoint source:\nCLI:     %+v\nService: %+v",
			fromCLI, fromService)
	}

	// Structural spot checks against the endpoint keys, including by_day and
	// by_model shape for the same fixture.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"since", "generated_at", "days", "source", "total", "by_day", "by_model", "models_truncated"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("JSON output missing endpoint key %q: %s", key, out)
		}
	}
	var byDay []map[string]any
	if err := json.Unmarshal(raw["by_day"], &byDay); err != nil {
		t.Fatal(err)
	}
	if len(byDay) != 1 {
		t.Fatalf("by_day=%v, want one bucket for the fixture window", byDay)
	}
	if _, ok := byDay[0]["invocations_total"]; !ok {
		t.Fatalf("by_day rows lost invocation observability: %v", byDay[0])
	}
	var byModel []map[string]any
	if err := json.Unmarshal(raw["by_model"], &byModel); err != nil {
		t.Fatal(err)
	}
	if len(byModel) != 2 {
		t.Fatalf("by_model=%v, want both alpha fixture models", byModel)
	}
}

// TestCostAnalyticsRejectsInvalidDays mirrors the HTTP handler validation:
// days outside 1..365 (or unparseable) are rejected, never silently clamped.
func TestCostAnalyticsRejectsInvalidDays(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	for _, args := range [][]string{
		{"analytics", "--days", "0"},
		{"analytics", "--days", "366"},
		{"analytics", "--days", "-3"},
		{"analytics", "--days", "not-a-number"},
		{"analytics", "--days=" + "0"},
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
		if err := costAnalyticsCmd(args); err == nil {
			t.Fatalf("cost analytics unexpectedly accepted %v", args)
		}
	}
}

// TestCostAnalyticsHumanOutputKeepsUnknownEvidenceHonest checks that the human
// path keeps invocation/error observability, keeps unknown-cost evidence
// visible as a count (UNKNOWN is never coerced into a fabricated 0) and shows
// models without cost evidence as unmeasured instead of zero-cost.
func TestCostAnalyticsHumanOutputKeepsUnknownEvidenceHonest(t *testing.T) {
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
		return costAnalyticsCmd([]string{"--days", "7"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Observações:   3 (invocações: 3, falhas: 1)") {
		t.Fatalf("human output lost invocation evidence: %s", out)
	}
	if !strings.Contains(out, "Taxa de erro:  33.3%") {
		t.Fatalf("human output lost error rate: %s", out)
	}
	if !strings.Contains(out, "Custo desconhecido: 2 de 3 observações sem custo classificável") {
		t.Fatalf("human output lost unknown-cost evidence: %s", out)
	}
	if !strings.Contains(out, "Top modelos por custo:") {
		t.Fatalf("human output lost top models section: %s", out)
	}
	// Ranked by recorded cost: the provider-reported row comes first and the
	// unknown-cost model reads as unmeasured, never as a zero-cost line.
	reported := strings.Index(out, "codex/model-one")
	unknown := strings.Index(out, "claude/model-two")
	if reported == -1 || unknown == -1 {
		t.Fatalf("human output lost model rows: %s", out)
	}
	if reported > unknown {
		t.Fatalf("top models not ranked by recorded cost: %s", out)
	}
	if !strings.Contains(out, "custo: reportado 1500 µUSD") {
		t.Fatalf("human output lost provider-reported cost evidence: %s", out)
	}
	if !strings.Contains(out, "custo não medido") {
		t.Fatalf("unknown-cost model must read as unmeasured, not fabricated zero: %s", out)
	}
}

// TestCostAnalyticsProjectIDFilter verifies --project-id narrows the report to
// the requested project without changing the schema or inventing rows.
func TestCostAnalyticsProjectIDFilter(t *testing.T) {
	nexus.ResetDefaultForTest()
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	t.Cleanup(nexus.ResetDefaultForTest)

	st := nexus.Default().Store()
	if st == nil {
		t.Fatal("missing canonical cost ledger")
	}
	costAnalyticsFixture(t, st)

	out, err := captureStdout(func() error {
		return costAnalyticsCmd([]string{"--days", "7", "--project-id", "beta", "--json"})
	})
	if err != nil {
		t.Fatal(err)
	}
	var report nexus.CostAnalyticsReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}
	if report.ProjectID != "beta" || report.Total.ObservationCount != 1 {
		t.Fatalf("--project-id filter ignored: %+v", report)
	}
	if report.Total.APIEquivalentMicroUSD != 700 {
		t.Fatalf("equivalent cost evidence lost by filter: %+v", report.Total)
	}
	if report.Total.ProviderReportedMicroUSD != 0 {
		t.Fatalf("other project's provider cost leaked into filter: %+v", report.Total)
	}
}
