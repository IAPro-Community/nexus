package quota

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestCodexHistoricalQuotaNeverBecomesOperationalCapacity(t *testing.T) {
	remaining := 12.0
	cases := []struct {
		name string
		status model.UsageStatus
		age time.Duration
		expired bool
	}{
		{"25-day estimated usage", model.UsageEstimated, 25 * 24 * time.Hour, true},
		{"one-hour cached quota", model.UsageCached, time.Hour, true},
		{"old official block", model.UsageRateLimited, time.Hour, true},
		{"recent estimated observation", model.UsageEstimated, 10 * time.Minute, false},
		{"fresh official quota", model.UsageLive, time.Minute, false},
	}
	now := time.Now()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			snap := model.UsageSnapshot{
				ProviderID: "codex",
				ProfileID: "kivergmail",
				Status: tc.status,
				Source: model.SourceObservation,
				FetchedAt: now.Add(-tc.age),
				Windows: []model.UsageWindow{{
					Kind: "weekly",
					RemainingPercent: &remaining,
				}},
			}
			if got := CodexEvidenceExpired(snap, now); got != tc.expired {
				t.Fatalf("expired = %v, want %v", got, tc.expired)
			}
			view := BuildQuotaView(snap, "account@example.com", "ChatGPT Plus")
			if tc.expired {
				if view.Status != string(model.UsageUnknown) || view.Available {
					t.Fatalf("historical quota must fail closed; view=%+v", view)
				}
				if view.FetchedAt.IsZero() || !view.FetchedAt.Equal(snap.FetchedAt) {
					t.Fatal("keep observation timestamp for troubleshooting")
				}
				for _, group := range view.ModelGroups {
					for _, w := range group.Windows {
						if w.Kind != "unknown" {
							t.Fatalf("stale %v quota leaked: %+v", tc.status, w)
						}
					}
				}
			} else if view.Status != string(tc.status) {
				t.Fatalf("fresh status changed unexpectedly: %s", view.Status)
			}
		})
	}
}

func TestCodexFreshReadWithExpiredResetWindowIsNotCurrentCapacity(t *testing.T) {
	now := time.Now()
	reset := now.Add(-time.Minute)
	remaining := 99.0
	snap := model.UsageSnapshot{
		ProviderID: "codex",
		ProfileID: "test",
		Status: model.UsageLive,
		Source: model.SourceOfficialAPI,
		FetchedAt: now.Add(-2 * time.Minute),
		Windows: []model.UsageWindow{{
			Kind: "5h",
			RemainingPercent: &remaining,
			ResetTime: &reset,
		}},
	}
	if !CodexEvidenceExpired(snap, now) {
		t.Fatal("a recently fetched reading with already-reset window must be revalidated")
	}
	view := BuildQuotaView(snap, "account", "Plus")
	if view.Available || view.Status != string(model.UsageUnknown) {
		t.Fatalf("post-reset historical percentage cannot influence routing: %+v", view)
	}
}

func TestOtherProvidersKeepIndependentFreshnessRules(t *testing.T) {
	pct := 50.0
	snap := model.UsageSnapshot{
		ProviderID: "agy",
		Status: model.UsageEstimated,
		FetchedAt: time.Now().Add(-time.Hour),
		Windows: []model.UsageWindow{{Kind: "weekly", RemainingPercent: &pct}},
	}
	if CodexEvidenceExpired(snap, time.Now()) {
		t.Fatal("do not change other provider policies in a Codex hotfix")
	}
}
