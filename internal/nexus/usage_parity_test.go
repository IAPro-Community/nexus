package nexus

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/profile"
)

func TestUsageCatalogAndProjectionParityExcludesGhosts(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	hostHome := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("NEXUS_REAL_HOME", hostHome)
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)

	// Create 2 legitimate AGY profiles + ghost/unverified directories.
	if _, err := profile.Create("agy", "acc-primary"); err != nil {
		t.Fatal(err)
	}
	if _, err := profile.Create("agy", "acc-aux-only"); err != nil {
		t.Fatal(err)
	}
	for _, ghost := range []string{"--with", "--json", "(sem perfil)", "stray-without-metadata"} {
		if err := os.MkdirAll(filepath.Join(dataDir, "profiles", "agy", ghost, "home"), 0700); err != nil {
			t.Fatal(err)
		}
	}

	profiles, err := profile.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 2 {
		t.Fatalf("profile.List() = %+v, want strictly 2 registered profiles", profiles)
	}

	n := &Nexus{}
	accounts, err := n.ListResourcesContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 2 {
		t.Fatalf("ListResourcesContext returned %d accounts, want 2: %+v", len(accounts), accounts)
	}
	for _, acc := range accounts {
		if acc.QuotaProjection == nil {
			t.Fatalf("account %s missing QuotaProjection", acc.ID)
		}
	}

	// Verify tier-first projection parity across CLI/TUI/Web for AGY snapshots.
	now := time.Now()
	pHigh, pLow, pZero, pFull := 85.0, 5.0, 0.0, 100.0
	snapPrimary := model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "acc-primary",
		Status:     model.UsageLive,
		Source:     model.SourceLocalFiles,
		FetchedAt:  now,
		Windows: []model.UsageWindow{
			{Kind: "5h", Group: "gemini", RemainingPercent: &pHigh},
			{Kind: "claude_5h", Group: "claude_gpt", RemainingPercent: &pZero},
		},
	}
	snapAuxOnly := model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "acc-aux-only",
		Status:     model.UsageLive,
		Source:     model.SourceLocalFiles,
		FetchedAt:  now,
		Windows: []model.UsageWindow{
			{Kind: "5h", Group: "gemini", RemainingPercent: &pZero},
			{Kind: "claude_5h", Group: "claude_gpt", RemainingPercent: &pFull},
		},
	}
	snapLowPrimaryHighAux := model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "acc-low",
		Status:     model.UsageLive,
		Source:     model.SourceLocalFiles,
		FetchedAt:  now,
		Windows: []model.UsageWindow{
			{Kind: "5h", Group: "gemini", RemainingPercent: &pLow},
			{Kind: "claude_5h", Group: "claude_gpt", RemainingPercent: &pFull},
		},
	}

	qvPrimary := quota.BuildQuotaView(snapPrimary, "a@x.com", "Pro")
	qvAuxOnly := quota.BuildQuotaView(snapAuxOnly, "b@x.com", "Pro")
	qvLow := quota.BuildQuotaView(snapLowPrimaryHighAux, "c@x.com", "Pro")
	projPrimary := qvPrimary.ProjectionAt(now, quota.DefaultTTL, true, "healthy")
	projAuxOnly := qvAuxOnly.ProjectionAt(now, quota.DefaultTTL, true, "healthy")
	projLow := qvLow.ProjectionAt(now, quota.DefaultTTL, true, "healthy")

	if projPrimary.AuxiliaryOnly || projPrimary.ActiveTier != quota.GroupTierPrimary {
		t.Fatalf("expected primary tier for acc-primary, got %+v", projPrimary)
	}
	if !projAuxOnly.AuxiliaryOnly || projAuxOnly.ActiveTier != quota.GroupTierAuxiliary {
		t.Fatalf("expected auxiliary fallback tier for acc-aux-only, got %+v", projAuxOnly)
	}
	projs := []quota.QuotaProjection{projAuxOnly, projLow, projPrimary}
	sort.Slice(projs, func(i, j int) bool { return projs[i].RankScore > projs[j].RankScore })
	if projs[0].RankScore != projPrimary.RankScore || projs[1].RankScore != projLow.RankScore || projs[2].RankScore != projAuxOnly.RankScore {
		t.Fatalf("unexpected rank ordering: primary=%.2f low=%.2f auxOnly=%.2f", projPrimary.RankScore, projLow.RankScore, projAuxOnly.RankScore)
	}
}
