package scheduler

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/cooldown"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

func TestCodexHistoricalQuotaDoesNotMisrouteAndHonorsExplicitDefault(t *testing.T) {
	t.Setenv("AI_CLI_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	oldHigh, recentLow := 100.0, 40.0
	candidates := []model.Profile{
		{Provider: "codex", Name: "old-high"},
		{Provider: "codex", Name: "recent"},
	}
	accounts := map[string]model.AccountInfo{
		"old-high": {
			Authenticated: true, Health: model.HealthHealthy,
			Usage: model.UsageSnapshot{
				ProviderID: "codex", ProfileID: "old-high",
				Source: model.SourceObservation, Status: model.UsageEstimated,
				FetchedAt: time.Now().Add(-25 * 24 * time.Hour),
				Windows:   []model.UsageWindow{{Kind: "weekly", RemainingPercent: &oldHigh}},
			},
		},
		"recent": {
			Authenticated: true, Health: model.HealthHealthy,
			Usage: model.UsageSnapshot{
				ProviderID: "codex", ProfileID: "recent",
				Source: model.SourceOfficialAPI, Status: model.UsageLive,
				FetchedAt: time.Now(),
				Windows:   []model.UsageWindow{{Kind: "weekly", RemainingPercent: &recentLow}},
			},
		},
	}
	cfg := config.NewDefaultConfig()
	cfg.Defaults["codex"] = "old-high"
	selector := NewSelector(cfg, quota.NewEngine(5*time.Minute), cooldown.NewTracker())

	result, err := selector.SelectBestProfile(context.Background(), "codex", t.TempDir(), candidates, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.SelectedProfile == nil || result.SelectedProfile.Name != "recent" {
		t.Fatalf("fresh verified capacity must beat a stale default; got %+v", result.SelectedProfile)
	}

	accounts["recent"] = model.AccountInfo{
		Authenticated: true, Health: model.HealthHealthy,
		Usage: model.UsageSnapshot{ProviderID: "codex", ProfileID: "recent", Status: model.UsageUnknown},
	}
	result, err = selector.SelectBestProfile(context.Background(), "codex", t.TempDir(), candidates, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.SelectedProfile == nil || result.SelectedProfile.Name != "old-high" ||
		!strings.Contains(result.Reason, "unverified") {
		t.Fatalf("unknown capacity must honor explicit default without pretending it is best: %+v", result)
	}

	result, err = selector.SelectBestProfile(context.Background(), "codex", t.TempDir(), candidates, accounts, []string{"old-high"})
	if err == nil || result.SelectedProfile != nil {
		t.Fatalf("excluded default must not be selected when alternatives have UNKNOWN quota: result=%+v err=%v", result, err)
	}
}

func TestCodexUnknownCapacityWithoutPreferenceDoesNotGuessAccount(t *testing.T) {
	t.Setenv("AI_CLI_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	cfg := config.NewDefaultConfig()
	delete(cfg.Defaults, "codex")
	selector := NewSelector(cfg, quota.NewEngine(time.Minute), cooldown.NewTracker())
	candidates := []model.Profile{{Provider: "codex", Name: "a"}}
	accounts := map[string]model.AccountInfo{
		"a": {Authenticated: true, Health: model.HealthHealthy,
			Usage: model.UsageSnapshot{ProviderID: "codex", ProfileID: "a", Status: model.UsageUnknown}},
	}
	res, err := selector.SelectBestProfile(context.Background(), "codex", t.TempDir(), candidates, accounts, nil)
	if err == nil || res.SelectedProfile != nil {
		t.Fatalf("unknown with no pinned preference must be fail-closed; res=%+v err=%v", res, err)
	}
}
