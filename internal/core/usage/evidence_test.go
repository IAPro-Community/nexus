package usage

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestRemainingPercentPreservesProviderSemantics(t *testing.T) {
	usedZero := model.UsageObservation{Semantics: model.SemanticsUsedPercent, Value: 0}
	if got, ok := RemainingPercent(usedZero); !ok || got != 100 {
		t.Fatalf("used_percent=0 must mean 100%% remaining, got %v ok=%v", got, ok)
	}

	remaining := model.UsageObservation{Semantics: model.SemanticsRemainingPercent, Value: 73}
	if got, ok := RemainingPercent(remaining); !ok || got != 73 {
		t.Fatalf("remaining_percent=73 must stay 73%%, got %v ok=%v", got, ok)
	}

	absolute := model.UsageObservation{Semantics: model.SemanticsRemainingAbsolute, Value: 1200}
	if _, ok := RemainingPercent(absolute); ok {
		t.Fatal("absolute evidence must not fabricate a percentage")
	}
}

func TestReconcileQuotaObservationsSeparatesChannelsAndAccounts(t *testing.T) {
	now := time.Now().UTC()
	scopeA := model.AccountScope{ProviderID: "codex", ProfileID: "a", AccountID: "acct-a", IdentityVersion: "1"}
	scopeB := model.AccountScope{ProviderID: "codex", ProfileID: "a", AccountID: "acct-b", IdentityVersion: "1"}

	observations := []model.UsageObservation{
		{
			ProviderID: "codex", ProfileID: "a", AccountScope: scopeA,
			Channel: model.ResourceChannelSubscription, Metric: model.UsageMetricQuota,
			Semantics: model.SemanticsUsedPercent, Value: 0, ModelGroup: "claude_gpt",
			WindowKind: "5h", Status: model.UsageLive, Source: model.SourceOfficialAPI,
			Confidence: model.UsageConfidenceHigh, ObservedAt: now,
		},
		{
			ProviderID: "codex", ProfileID: "a", AccountScope: scopeA,
			Channel: model.ResourceChannelAPI, Metric: model.UsageMetricQuota,
			Semantics: model.SemanticsRemainingPercent, Value: 12, WindowKind: "monthly",
			Status: model.UsageLive, Source: model.SourceOfficialAPI, ObservedAt: now.Add(time.Second),
		},
		{
			ProviderID: "codex", ProfileID: "a", AccountScope: scopeB,
			Channel: model.ResourceChannelSubscription, Metric: model.UsageMetricQuota,
			Semantics: model.SemanticsRemainingPercent, Value: 1, ModelGroup: "claude_gpt",
			WindowKind: "weekly", Status: model.UsageLive, Source: model.SourceOfficialAPI,
			ObservedAt: now.Add(2 * time.Second),
		},
	}

	snapshot := ReconcileQuotaObservations("codex", "a", model.ResourceChannelSubscription, observations)
	if len(snapshot.Windows) != 1 {
		t.Fatalf("subscription reconciliation must exclude API and foreign-account evidence, windows=%d", len(snapshot.Windows))
	}
	if snapshot.Windows[0].RemainingPercent == nil || *snapshot.Windows[0].RemainingPercent != 100 {
		t.Fatalf("remaining=%v want 100", snapshot.Windows[0].RemainingPercent)
	}
	if snapshot.AccountScope.Key() != scopeA.Key() {
		t.Fatalf("scope=%+v want %+v", snapshot.AccountScope, scopeA)
	}
}
