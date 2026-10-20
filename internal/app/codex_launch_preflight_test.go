package app

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestCodexLaunchPreflightUsesOnlyRecentOfficialQuota(t *testing.T) {
	now := time.Now()
	remaining := 84.0
	reset := now.Add(2 * time.Hour)
	fresh := model.UsageSnapshot{
		ProviderID: "codex", ProfileID: "kivergmail",
		Source: model.SourceOfficialAPI, Status: model.UsageLive,
		FetchedAt: now.Add(-time.Minute),
		Windows: []model.UsageWindow{{Kind: "weekly", RemainingPercent: &remaining, ResetTime: &reset}},
	}
	if codexLaunchProbeNeeded(fresh, now) {
		t.Fatal("recent official evidence must avoid expensive launch preflight")
	}
	cases := []struct {
		name string
		snap model.UsageSnapshot
	}{
		{"rollout not official", func() model.UsageSnapshot {
			s := fresh
			s.Source = model.SourceObservation
			return s
		}()},
		{"25-day-old official data", func() model.UsageSnapshot {
			s := fresh
			s.FetchedAt = now.Add(-25 * 24 * time.Hour)
			return s
		}()},
		{"6-minute official cache", func() model.UsageSnapshot {
			s := fresh
			s.FetchedAt = now.Add(-6 * time.Minute)
			return s
		}()},
		{"reset already elapsed", func() model.UsageSnapshot {
			s := fresh
			expired := now.Add(-time.Minute)
			s.Windows = []model.UsageWindow{{Kind: "weekly", RemainingPercent: &remaining, ResetTime: &expired}}
			return s
		}()},
		{"unknown capacity", model.UsageSnapshot{
			ProviderID: "codex", ProfileID: "kivergmail", Status: model.UsageUnknown,
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !codexLaunchProbeNeeded(tc.snap, now) {
				t.Fatal("automatic Codex selection must try to refresh unverified capacity")
			}
		})
	}
}
