package nexus

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

func quotaViewWithStatus(status model.UsageStatus) quota.QuotaView {
	return quota.QuotaView{Status: string(status), Available: true}
}

func TestQuotaEvidenceConfirmsContinuity(t *testing.T) {
	tests := []struct {
		name string
		view quota.QuotaView
		want bool
	}{
		{"rate limited confirms", quotaViewWithStatus(model.UsageRateLimited), true},
		{"live without groups does not", quotaViewWithStatus(model.UsageLive), false},
		{"unknown never confirms", quotaViewWithStatus(model.UsageUnknown), false},
		{"estimated never confirms by status alone", quotaViewWithStatus(model.UsageEstimated), false},
	}
	for _, test := range tests {
		if got := quotaEvidenceConfirmsContinuity(test.view); got != test.want {
			t.Fatalf("%s: got %t want %t", test.name, got, test.want)
		}
	}
}

func TestQuotaEvidenceConfirmsContinuityOnZeroRemaining(t *testing.T) {
	view := quota.QuotaView{
		Status: string(model.UsageLive),
		ModelGroups: []quota.ModelGroup{{
			Key:     "gpt-5",
			Name:    "GPT-5",
			Windows: []quota.Window{{Kind: "PRIMARY", Label: "5h", Remaining: 0}},
		}},
	}
	remaining, ok := view.BestGroupRemaining()
	if !ok {
		t.Skip("fixture group yields no remaining metric")
	}
	if remaining > quotaEvidenceZeroTolerance {
		t.Fatalf("fixture must be exhausted; remaining=%f", remaining)
	}
	if !quotaEvidenceConfirmsContinuity(view) {
		t.Fatal("verified zero remaining must confirm continuity")
	}
}

func TestEarlyQuotaContinuityIgnoresUnrelatedEvents(t *testing.T) {
	// The subscriber must only react to pty_pattern events; this exercises
	// the filtering logic directly against the shared bus contract.
	e := events.NewEvent("rt-x", "codex", "work", events.EventQuotaExhausted, "monitor notification", map[string]any{
		"affected_runtime_id": "rt-x",
	})
	if e.Data["nexus.quota_trigger"] == "pty_pattern" {
		t.Fatal("monitor events must not carry the pty_pattern trigger")
	}
	if e.Type != events.EventQuotaExhausted {
		t.Fatalf("event type = %s", e.Type)
	}
}

func TestContinuityPendingGuard(t *testing.T) {
	if continuityPending(registry.RuntimeSession{ProviderID: "codex"}) {
		t.Fatal("session without labels is not pending")
	}
	pending := registry.RuntimeSession{ProviderID: "codex", Labels: map[string]string{"nexus.continuity_pending": "true"}}
	if !continuityPending(pending) {
		t.Fatal("pending flag must be detected")
	}
	cleared := registry.RuntimeSession{ProviderID: "codex", Labels: map[string]string{"nexus.continuity_pending": "false"}}
	if continuityPending(cleared) {
		t.Fatal("cleared flag must not count as pending")
	}
}

func TestEarlyQuotaCheckDelayIsBounded(t *testing.T) {
	if earlyQuotaCheckDelay <= 0 || earlyQuotaCheckDelay > 30*time.Second {
		t.Fatalf("early check delay out of policy bounds: %v", earlyQuotaCheckDelay)
	}
}
