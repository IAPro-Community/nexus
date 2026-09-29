package nexus

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

func TestQuotaRemainingRatioIsLiteralBestPoolCapacity(t *testing.T) {
	codex := quota.QuotaView{
		Status: string(model.UsageLive),
		ModelGroups: []quota.ModelGroup{{
			Key: "claude_gpt",
			Windows: []quota.Window{
				{Kind: "5h", Remaining: 100},
				{Kind: "weekly", Remaining: 80},
			},
		}},
	}
	if got := quotaRemainingRatio(codex); got != 0.80 {
		t.Fatalf("Codex quota_remaining=%v want 0.80 (min of same-pool windows)", got)
	}

	agy := quota.QuotaView{
		Status: string(model.UsageCached),
		ModelGroups: []quota.ModelGroup{
			{Key: "gemini", Windows: []quota.Window{{Kind: "5h", Remaining: 0}, {Kind: "weekly", Remaining: 66}}},
			{Key: "claude_gpt", Windows: []quota.Window{{Kind: "claude_5h", Remaining: 100}, {Kind: "claude_weekly", Remaining: 100}}},
		},
	}
	if got := quotaRemainingRatio(agy); got != 1.0 {
		t.Fatalf("AGY quota_remaining=%v want 1.0 (best independent pool), not composite scheduler score", got)
	}
}

func TestQuotaRemainingRatioUnknownNeverInventsCapacity(t *testing.T) {
	qv := quota.QuotaView{Status: string(model.UsageUnknown)}
	if got := quotaRemainingRatio(qv); got != 0 {
		t.Fatalf("UNKNOWN quota_remaining=%v want 0", got)
	}
}


func TestSubscriptionResourceChannelsPreserveExistingSnapshot(t *testing.T) {
	remaining, used := 100.0, 0.0
	snap := model.UsageSnapshot{
		ProviderID: "codex",
		ProfileID:  "work",
		Status:     model.UsageLive,
		Source:     model.SourceOfficialAPI,
		Windows: []model.UsageWindow{{
			Kind:             "5h",
			Group:            "claude_gpt",
			RemainingPercent: &remaining,
			UsedPercent:      &used,
		}},
	}
	channels := subscriptionResourceChannels(snap)
	if len(channels) != 1 {
		t.Fatalf("expected one subscription channel, got %d", len(channels))
	}
	if channels[0].Channel != model.ResourceChannelSubscription || channels[0].Quota == nil {
		t.Fatalf("unexpected subscription projection: %+v", channels[0])
	}
	if channels[0].Quota.ProviderID != "codex" || *channels[0].Quota.Windows[0].RemainingPercent != 100 {
		t.Fatalf("subscription projection changed quota evidence: %+v", channels[0].Quota)
	}
}

func TestSubscriptionResourceChannelsEmptySnapshotStaysOmitted(t *testing.T) {
	if channels := subscriptionResourceChannels(model.UsageSnapshot{}); channels != nil {
		t.Fatalf("empty snapshot must not fabricate resource channels: %+v", channels)
	}
}
