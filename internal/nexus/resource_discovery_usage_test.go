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
