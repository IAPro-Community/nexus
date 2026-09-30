package nexus

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

func liveQuotaViewForResourceTruth() quota.QuotaView {
	remaining := 75.0
	return quota.BuildQuotaView(model.UsageSnapshot{
		ProviderID: "codex",
		ProfileID:  "work",
		Status:     model.UsageLive,
		Windows: []model.UsageWindow{{
			Kind:             "5h",
			Group:            "codex",
			RemainingPercent: &remaining,
		}},
	}, "work@example.com", "pro")
}

func TestReconcileResourceAvailabilityRequiresInstalledAuthenticatedProvider(t *testing.T) {
	tests := []struct {
		name           string
		detectionKnown bool
		installed      bool
		authenticated  bool
		wantAvailable  bool
		wantAuthReason bool
	}{
		{name: "usable", detectionKnown: true, installed: true, authenticated: true, wantAvailable: true},
		{name: "provider missing", detectionKnown: true, installed: false, authenticated: true, wantAvailable: false},
		{name: "driver unknown", detectionKnown: false, installed: false, authenticated: true, wantAvailable: false},
		{name: "auth missing", detectionKnown: true, installed: true, authenticated: false, wantAvailable: false, wantAuthReason: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qv := liveQuotaViewForResourceTruth()
			got := reconcileResourceAvailability(&qv, tt.detectionKnown, tt.installed, tt.authenticated)
			if got != tt.wantAvailable {
				t.Fatalf("available=%v want %v", got, tt.wantAvailable)
			}
			if qv.Available != tt.wantAvailable {
				t.Fatalf("quota view available=%v want %v", qv.Available, tt.wantAvailable)
			}
			if qv.AvailReasons.AuthRequired != tt.wantAuthReason {
				t.Fatalf("auth_required=%v want %v", qv.AvailReasons.AuthRequired, tt.wantAuthReason)
			}
		})
	}
}

func TestQuotaTotalRatioDoesNotTurnUnknownIntoZeroPercent(t *testing.T) {
	qv := quota.QuotaView{Status: string(model.UsageUnknown)}
	if got := quotaTotalRatio(qv); got != 0 {
		t.Fatalf("UNKNOWN quota total=%v want 0 (unmeasured)", got)
	}

	live := liveQuotaViewForResourceTruth()
	if got := quotaTotalRatio(live); got != 1 {
		t.Fatalf("LIVE measurable quota total=%v want 1", got)
	}
}
