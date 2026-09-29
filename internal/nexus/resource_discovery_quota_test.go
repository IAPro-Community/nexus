package nexus

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

func TestQuotaRemainingIsLiteralAndRoutingScoreIsSeparate(t *testing.T) {
	qv := quota.QuotaView{
		Status: string(model.UsageCached),
		ModelGroups: []quota.ModelGroup{
			{Key: "gemini", Windows: []quota.Window{{Kind: "5h", Remaining: 0}, {Kind: "weekly", Remaining: 66}}},
			{Key: "claude_gpt", Windows: []quota.Window{{Kind: "5h", Remaining: 100}, {Kind: "weekly", Remaining: 100}}},
		},
	}

	ratio, ok := literalQuotaRemainingRatio(qv)
	if !ok || ratio != 1 {
		t.Fatalf("quota_remaining=%v ok=%v want literal best-pool 1.0", ratio, ok)
	}
	routing, ok := qv.EffectiveCapacityScore()
	if !ok || routing != 150 {
		t.Fatalf("routing capacity=%v ok=%v want 150; it must remain distinct from quota_remaining", routing, ok)
	}
}

func TestSubscriptionResourceChannelDoesNotInventUnknownQuota(t *testing.T) {
	qv := quota.QuotaView{Status: string(model.UsageUnknown)}
	state := subscriptionResourceChannel(qv, 0, false)
	if state.Channel != model.ResourceChannelSubscription {
		t.Fatalf("channel=%q", state.Channel)
	}
	if state.QuotaRemaining != nil {
		t.Fatalf("UNKNOWN quota must not become synthetic zero: %+v", state)
	}
}
