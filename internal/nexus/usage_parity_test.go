package nexus_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
	"github.com/kivervinicius/ai-cli/internal/nexus"
	"github.com/kivervinicius/ai-cli/internal/tui"
)

// TestUsageParityAcrossSurfaces ensures QuotaView status/remaining/eligibility
// agree for the same fixture whether consumed as ProviderAccount (web), raw
// view (`nexus usage --json`) or TUI rows.
func TestUsageParityAcrossSurfaces(t *testing.T) {
	remaining := 72.0
	snap := model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "demo",
		Account:    "demo@example.com",
		Status:     model.UsageLive,
		Source:     model.SourceOfficialAPI,
		FetchedAt:  time.Now().UTC(),
		Windows: []model.UsageWindow{{
			Kind:             "5h",
			Group:            "claude",
			RemainingPercent: &remaining,
		}},
	}
	qv := quota.BuildQuotaView(snap, "demo@example.com", "pro")
	if qv.Status != string(model.UsageLive) {
		t.Fatalf("quota view status=%s", qv.Status)
	}
	if !qv.Available {
		t.Fatalf("expected available view")
	}

	acc := nexus.ProviderAccount{
		ID:            "agy:demo",
		Provider:      "agy",
		Profile:       "demo",
		Authenticated: true,
		Available:     qv.IsAvailable(),
		AvailReasons:  &qv.AvailReasons,
		QuotaView:     &qv,
		Health:        "healthy",
	}
	elig := quota.EvaluateEligibility(quota.EligibilityInput{
		View:          qv,
		Authenticated: true,
		Health:        "healthy",
	})
	acc.Eligibility = &elig

	if acc.QuotaView == nil || acc.QuotaView.Status != "LIVE" {
		t.Fatalf("web account quota_view status=%v", acc.QuotaView)
	}
	if acc.Eligibility == nil || !acc.Eligibility.Eligible || acc.Eligibility.Confidence != "LIVE" {
		t.Fatalf("eligibility=%+v", acc.Eligibility)
	}
	if acc.Available != qv.Available {
		t.Fatalf("available mismatch account=%v view=%v", acc.Available, qv.Available)
	}
	windows := qv.AllWindows()
	if len(windows) == 0 || windows[0].Remaining != remaining {
		t.Fatalf("remaining mismatch windows=%+v want %.1f", windows, remaining)
	}

	type usageJSONRow struct {
		model.UsageSnapshot
		QuotaView     quota.QuotaView      `json:"quota_view"`
		Eligibility   quota.Eligibility    `json:"eligibility"`
		Health        model.ProviderHealth `json:"health"`
		Authenticated bool                 `json:"authenticated"`
	}
	payload, err := json.Marshal(usageJSONRow{
		UsageSnapshot: snap,
		QuotaView:     qv,
		Eligibility:   elig,
		Health:        model.HealthHealthy,
		Authenticated: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["status"] != "LIVE" {
		t.Fatalf("json status=%v", decoded["status"])
	}
	qvJSON, _ := decoded["quota_view"].(map[string]any)
	if qvJSON["status"] != "LIVE" {
		t.Fatalf("json quota_view.status=%v", qvJSON["status"])
	}
	eligJSON, _ := decoded["eligibility"].(map[string]any)
	if eligJSON["eligible"] != true || eligJSON["confidence"] != "LIVE" {
		t.Fatalf("json eligibility=%v", eligJSON)
	}

	rows := tui.BuildUsageRowsForTest("agy", "demo", qv, model.AccountInfo{Email: "demo@example.com", Plan: "pro"}, nil)
	if len(rows) == 0 {
		t.Fatal("expected TUI rows")
	}
	if rows[0].SnapshotStatus != string(model.UsageLive) {
		t.Fatalf("tui status=%s", rows[0].SnapshotStatus)
	}
}
