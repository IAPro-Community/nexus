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

func TestSmartAccountSelector(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_STATE_DIR", stateDir)

	cfg := config.NewDefaultConfig()
	qEng := quota.NewEngine(5 * time.Minute)
	cdTracker := cooldown.NewTracker()
	selector := NewSelector(cfg, qEng, cdTracker)

	ctx := context.Background()

	// Scenario 1: Profile A has 20%, Profile B has 85% -> B must be selected
	pctA := 20.0
	pctB := 85.0
	_ = qEng.SaveUsage(model.UsageSnapshot{
		ProviderID: "codex",
		ProfileID:  "acc-a",
		Status:     model.UsageLive,
		Windows:    []model.UsageWindow{{Kind: "5h", RemainingPercent: &pctA}},
	})
	_ = qEng.SaveUsage(model.UsageSnapshot{
		ProviderID: "codex",
		ProfileID:  "acc-b",
		Status:     model.UsageLive,
		Windows:    []model.UsageWindow{{Kind: "5h", RemainingPercent: &pctB}},
	})

	candidates := []model.Profile{
		{Provider: "codex", Name: "acc-a"},
		{Provider: "codex", Name: "acc-b"},
	}
	accounts := map[string]model.AccountInfo{
		"acc-a": {Authenticated: true, Health: model.HealthHealthy},
		"acc-b": {Authenticated: true, Health: model.HealthHealthy},
	}

	res1, err := selector.SelectBestProfile(ctx, "codex", "/tmp", candidates, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res1.SelectedProfile.Name != "acc-b" {
		t.Fatalf("expected acc-b (85%%) to be selected, got %s", res1.SelectedProfile.Name)
	}

	// Scenario 2: Profile B becomes RATE_LIMITED -> acc-a (20%) must be selected
	cdTracker.RecordRateLimit("codex", "acc-b", 30*time.Minute, nil, "HTTP 429")
	res2, err := selector.SelectBestProfile(ctx, "codex", "/tmp", candidates, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res2.SelectedProfile.Name != "acc-a" {
		t.Fatalf("expected acc-a to be selected when acc-b is rate limited, got %s", res2.SelectedProfile.Name)
	}

	// Scenario 3: UNKNOWN quota is not healthy capacity and cannot be selected.
	cdTracker.Clear("codex", "acc-b")
	cdTracker.RecordRateLimit("codex", "acc-a", 15*time.Minute, nil, "HTTP 429")
	_ = qEng.SaveUsage(model.UsageSnapshot{
		ProviderID: "codex",
		ProfileID:  "acc-b",
		Status:     model.UsageUnknown, // UNKNOWN
		Windows:    nil,
	})

	res3, err := selector.SelectBestProfile(ctx, "codex", "/tmp", candidates, accounts, nil)
	if err == nil {
		t.Fatalf("expected UNKNOWN-only capacity to be rejected, got %#v", res3.SelectedProfile)
	}
	if res3.SelectedProfile != nil {
		t.Fatalf("UNKNOWN quota must not be selected, got %s", res3.SelectedProfile.Name)
	}

	// Scenario 4: All accounts rate limited -> returns error with reasons
	cdTracker.RecordRateLimit("codex", "acc-b", 15*time.Minute, nil, "HTTP 429")
	res4, err := selector.SelectBestProfile(ctx, "codex", "/tmp", candidates, accounts, nil)
	if err == nil {
		t.Fatalf("expected error when all accounts are rate limited, got selected=%+v", res4)
	}
	if !strings.Contains(err.Error(), "no usable codex profiles") {
		t.Fatalf("expected descriptive error message, got %v", err)
	}

	// Scenario 5: Explain Selection
	explain := selector.ExplainSelection(ctx, "codex", "/tmp", candidates, accounts)
	if !strings.Contains(explain, "Evaluation of all candidate profiles") {
		t.Fatalf("unexpected explain output: %s", explain)
	}
}

func TestMultiQuotaBottleneckSelection(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_STATE_DIR", stateDir)

	cfg := config.NewDefaultConfig()
	// omega is default — must still lose to the account with both families usable
	cfg.Defaults = map[string]string{
		"agy": "omega",
	}

	qEng := quota.NewEngine(5 * time.Minute)
	cdTracker := cooldown.NewTracker()
	selector := NewSelector(cfg, qEng, cdTracker)
	ctx := context.Background()

	// omega: Gemini 5h exhausted, Claude full (1 usable family)
	g0, g66, c100 := 0.0, 66.0, 100.0
	_ = qEng.SaveUsage(model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "omega",
		Status:     model.UsageLive,
		Source:     model.SourceCLIOutput,
		FetchedAt:  time.Now(),
		Windows: []model.UsageWindow{
			{Kind: "5h", Group: "gemini", RemainingPercent: &g0},
			{Kind: "weekly", Group: "gemini", RemainingPercent: &g66},
			{Kind: "claude_5h", Group: "claude_gpt", RemainingPercent: &c100},
			{Kind: "claude_weekly", Group: "claude_gpt", RemainingPercent: &c100},
		},
	})

	// gmail: both families available
	g90, g94, c40, c80 := 90.0, 94.0, 40.0, 80.0
	_ = qEng.SaveUsage(model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "gmail",
		Status:     model.UsageLive,
		Source:     model.SourceCLIOutput,
		FetchedAt:  time.Now(),
		Windows: []model.UsageWindow{
			{Kind: "5h", Group: "gemini", RemainingPercent: &g90},
			{Kind: "weekly", Group: "gemini", RemainingPercent: &g94},
			{Kind: "claude_5h", Group: "claude_gpt", RemainingPercent: &c40},
			{Kind: "claude_weekly", Group: "claude_gpt", RemainingPercent: &c80},
		},
	})

	candidates := []model.Profile{
		{Provider: "agy", Name: "omega"},
		{Provider: "agy", Name: "gmail"},
	}
	accounts := map[string]model.AccountInfo{
		"omega": {Authenticated: true, Health: model.HealthHealthy, Email: "omega@example.com"},
		"gmail": {Authenticated: true, Health: model.HealthHealthy, Email: "gmail@example.com"},
	}

	res, err := selector.SelectBestProfile(ctx, "agy", "/tmp", candidates, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.SelectedProfile.Name != "gmail" {
		t.Fatalf("expected gmail (2 usable families) over omega (1 usable family), got %s", res.SelectedProfile.Name)
	}
}

func TestKnownQuotaOutranksUnknownQuota(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_STATE_DIR", stateDir)

	selector := NewSelector(config.NewDefaultConfig(), quota.NewEngine(5*time.Minute), cooldown.NewTracker())
	remaining := 28.0
	if err := selector.quotaEng.SaveUsage(model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "known",
		Status:     model.UsageLive,
		Source:     model.SourceCLIOutput,
		FetchedAt:  time.Now(),
		Windows: []model.UsageWindow{
			{Kind: "weekly", Group: "gemini", RemainingPercent: &remaining},
		},
	}); err != nil {
		t.Fatal(err)
	}

	candidates := []model.Profile{
		{Provider: "agy", Name: "unknown"},
		{Provider: "agy", Name: "known"},
	}
	accounts := map[string]model.AccountInfo{
		"unknown": {Authenticated: true, Health: model.HealthHealthy},
		"known":   {Authenticated: true, Health: model.HealthHealthy},
	}

	result, err := selector.SelectBestProfile(context.Background(), "agy", t.TempDir(), candidates, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.SelectedProfile == nil || result.SelectedProfile.Name != "known" {
		t.Fatalf("expected known quota profile, got %#v", result.SelectedProfile)
	}
}

func TestDefaultProfileTieBreaker(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_STATE_DIR", stateDir)

	cfg := config.NewDefaultConfig()
	cfg.Defaults = map[string]string{
		"agy": "acc-default",
	}

	qEng := quota.NewEngine(5 * time.Minute)
	cdTracker := cooldown.NewTracker()
	selector := NewSelector(cfg, qEng, cdTracker)
	ctx := context.Background()

	// Both have 100% capacity
	p100 := 100.0
	_ = qEng.SaveUsage(model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "acc-default",
		Status:     model.UsageLive,
		Windows:    []model.UsageWindow{{Kind: "5h", RemainingPercent: &p100}},
	})
	_ = qEng.SaveUsage(model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "acc-other",
		Status:     model.UsageLive,
		Windows:    []model.UsageWindow{{Kind: "5h", RemainingPercent: &p100}},
	})

	candidates := []model.Profile{
		{Provider: "agy", Name: "acc-other"},
		{Provider: "agy", Name: "acc-default"},
	}
	accounts := map[string]model.AccountInfo{
		"acc-other":   {Authenticated: true, Health: model.HealthHealthy},
		"acc-default": {Authenticated: true, Health: model.HealthHealthy},
	}

	res, err := selector.SelectBestProfile(ctx, "agy", "/tmp", candidates, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.SelectedProfile.Name != "acc-default" {
		t.Fatalf("expected acc-default to win tie-break on equal 100%% capacity, got %s", res.SelectedProfile.Name)
	}
}

// TestSelectBestProfileEmptyCandidatesNeverNilResult is a regression test for a
// nil-pointer crash: controlStartCmd dereferenced res.SelectedProfile after
// ignoring the error, and SelectBestProfile returned a nil result for providers
// with no configured profiles.
func TestSelectBestProfileEmptyCandidatesNeverNilResult(t *testing.T) {
	cfg := config.NewDefaultConfig()
	qEng := quota.NewEngine(5 * time.Minute)
	cdTracker := cooldown.NewTracker()
	selector := NewSelector(cfg, qEng, cdTracker)

	ctx := context.Background()
	res, err := selector.SelectBestProfile(ctx, "provider-with-no-profiles", "/tmp", nil, nil, nil)
	if res == nil {
		t.Fatal("SelectBestProfile must never return a nil result for empty candidates")
	}
	if err == nil {
		t.Fatal("expected an error describing the missing profiles")
	}
	if res.SelectedProfile != nil {
		t.Error("expected no selected profile for empty candidates")
	}
}

func TestCodexExhaustedFiveHourLosesToHealthyAccount(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_STATE_DIR", stateDir)

	selector := NewSelector(config.NewDefaultConfig(), quota.NewEngine(5*time.Minute), cooldown.NewTracker())
	zero, healthy5h, healthyWk := 0.0, 99.0, 63.0
	denied5h, deniedWk := 0.0, 56.0

	accounts := map[string]model.AccountInfo{
		"denied": {
			Authenticated: true,
			Health:        model.HealthHealthy,
			Usage: model.UsageSnapshot{
				ProviderID: "codex", ProfileID: "denied", Status: model.UsageLive,
				Source: model.SourceObservation, FetchedAt: time.Now(),
				Windows: []model.UsageWindow{
					{Kind: "5h", Group: "claude_gpt", RemainingPercent: &denied5h},
					{Kind: "weekly", Group: "claude_gpt", RemainingPercent: &deniedWk},
				},
			},
		},
		"healthy": {
			Authenticated: true,
			Health:        model.HealthHealthy,
			Usage: model.UsageSnapshot{
				ProviderID: "codex", ProfileID: "healthy", Status: model.UsageLive,
				Source: model.SourceObservation, FetchedAt: time.Now(),
				Windows: []model.UsageWindow{
					{Kind: "5h", Group: "claude_gpt", RemainingPercent: &healthy5h},
					{Kind: "weekly", Group: "claude_gpt", RemainingPercent: &healthyWk},
				},
			},
		},
	}
	_ = zero
	candidates := []model.Profile{
		{Provider: "codex", Name: "denied"},
		{Provider: "codex", Name: "healthy"},
	}
	res, err := selector.SelectBestProfile(context.Background(), "codex", "/tmp", candidates, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.SelectedProfile == nil || res.SelectedProfile.Name != "healthy" {
		t.Fatalf("yolo/auto-select must prefer healthy capacity over exhausted 5h, got %#v", res.SelectedProfile)
	}
}
