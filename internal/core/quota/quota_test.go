package quota

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestQuotaEnginePersistenceAndRendering(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	engine := NewEngine(5 * time.Minute)

	// Test 1: Empty cache returns UNKNOWN, not 100%
	snap, found := engine.GetCachedUsage("codex", "work")
	if found {
		t.Fatalf("expected no cached usage, got found=true")
	}
	if snap.Status != model.UsageUnknown {
		t.Fatalf("expected UNKNOWN status, got %s", snap.Status)
	}

	// Test 2: Progress bar for UNKNOWN
	unknownBar := RenderShortStatus(model.UsageUnknown, nil, 10)
	if !strings.Contains(unknownBar, "UNKNOWN") {
		t.Fatalf("unexpected unknown bar rendering: %s", unknownBar)
	}

	// Test 3: Save and Load Live Snapshot
	remPct := 75.5
	usedPct := 24.5
	liveSnap := model.UsageSnapshot{
		ProviderID: "codex",
		ProfileID:  "work",
		Status:     model.UsageLive,
		Source:     model.SourceOfficialAPI,
		FetchedAt:  time.Now(),
		Windows: []model.UsageWindow{
			{
				Kind:             "5h",
				RemainingPercent: &remPct,
				UsedPercent:      &usedPct,
			},
		},
	}

	if err := engine.SaveUsage(liveSnap); err != nil {
		t.Fatal(err)
	}

	cached, found := engine.GetCachedUsage("codex", "work")
	if !found {
		t.Fatalf("expected cached usage to be found")
	}
	if cached.Status != model.UsageCached {
		t.Fatalf("expected status CACHED when read from cache, got %s", cached.Status)
	}
	if len(cached.Windows) != 1 || *cached.Windows[0].RemainingPercent != 75.5 {
		t.Fatalf("unexpected cached window percent: %+v", cached.Windows)
	}

	// Test 4: Render known progress bar
	knownBar := RenderShortStatus(model.UsageCached, cached.Windows[0].RemainingPercent, 10)
	if !strings.Contains(knownBar, "76%") && !strings.Contains(knownBar, "75%") {
		t.Fatalf("unexpected known bar rendering: %s", knownBar)
	}
}

func TestFetchBatchConcurrency(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	engine := NewEngine(5 * time.Minute)

	profiles := []model.Profile{
		{Provider: "codex", Name: "acc1"},
		{Provider: "codex", Name: "acc2"},
		{Provider: "agy", Name: "acc3"},
	}

	ctx := context.Background()
	results := engine.FetchBatch(ctx, profiles, func(ctx context.Context, p model.Profile) model.UsageSnapshot {
		pct := 50.0
		return model.UsageSnapshot{
			ProviderID: p.Provider,
			ProfileID:  p.Name,
			Status:     model.UsageLive,
			Source:     model.SourceObservation,
			FetchedAt:  time.Now(),
			Windows: []model.UsageWindow{
				{
					Kind:             "5h",
					RemainingPercent: &pct,
				},
			},
		}
	}, 2)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Status != model.UsageLive {
			t.Errorf("expected LIVE status, got %s", r.Status)
		}
	}
}

func TestQuotaViewBottleneckUnknownWithoutWindowsIsZero(t *testing.T) {
	qv := QuotaView{Status: "UNKNOWN"}
	remaining, kind := qv.Bottleneck()
	if remaining != 0 || kind != "" {
		t.Fatalf("unknown quota without windows must not imply 100%% remaining, got %.1f %q", remaining, kind)
	}
}

func TestQuotaViewBottleneckFullRemainingKeepsKind(t *testing.T) {
	qv := QuotaView{
		Status: "CACHED",
		ModelGroups: []ModelGroup{
			{Name: "Claude & GPT Models", Windows: []Window{
				{Kind: "5h", Remaining: 100},
				{Kind: "weekly", Remaining: 100},
			}},
		},
	}
	remaining, kind := qv.Bottleneck()
	if remaining != 100 || kind == "" {
		t.Fatalf("100%% remaining on every window must stay 100%% with a bottleneck kind, got %.1f %q", remaining, kind)
	}
}

func TestBestGroupRemainingIndependentPools(t *testing.T) {
	qv := QuotaView{
		Status: string(model.UsageCached),
		ModelGroups: []ModelGroup{
			{Key: "gemini", Name: "Gemini Models", Windows: []Window{{Kind: "5h", Remaining: 0}, {Kind: "weekly", Remaining: 66}}},
			{Key: "claude_gpt", Name: "Claude & GPT Models", Windows: []Window{{Kind: "claude_5h", Remaining: 100}, {Kind: "claude_weekly", Remaining: 100}}},
		},
	}
	best, ok := qv.BestGroupRemaining()
	if !ok || best != 100 {
		t.Fatalf("BestGroupRemaining=%v ok=%v want 100 true", best, ok)
	}
	bottleneck, kind := qv.Bottleneck()
	if bottleneck != 0 || kind != "5h" {
		t.Fatalf("Bottleneck=%v %q want 0 5h", bottleneck, kind)
	}
	summary := qv.CompactGroupSummary()
	if summary != "Gemini 0% · Claude 100%" {
		t.Fatalf("CompactGroupSummary=%q", summary)
	}
}

func TestCodexSinglePoolMinOfWindows(t *testing.T) {
	qv := QuotaView{
		Status: string(model.UsageCached),
		ModelGroups: []ModelGroup{
			{Key: "claude_gpt", Name: "Claude & GPT Models", Windows: []Window{{Kind: "5h", Remaining: 0}, {Kind: "weekly", Remaining: 90}}},
		},
	}
	best, ok := qv.BestGroupRemaining()
	if !ok || best != 0 {
		t.Fatalf("Codex 5h=0 weekly=90 must score 0, got %v ok=%v", best, ok)
	}
	full := QuotaView{
		Status: string(model.UsageCached),
		ModelGroups: []ModelGroup{
			{Key: "claude_gpt", Windows: []Window{{Kind: "5h", Remaining: 100}, {Kind: "weekly", Remaining: 100}}},
		},
	}
	bestFull, ok := full.BestGroupRemaining()
	if !ok || bestFull != 100 {
		t.Fatalf("full remaining must be 100, got %v ok=%v", bestFull, ok)
	}
}

func TestBottleneckScorePrefersMoreUsableFamilies(t *testing.T) {
	// omega: Gemini 5h dead, Claude full → 1 usable family
	omega := QuotaView{
		Status: string(model.UsageCached),
		ModelGroups: []ModelGroup{
			{Key: "gemini", Windows: []Window{{Kind: "5h", Remaining: 0}, {Kind: "weekly", Remaining: 66}}},
			{Key: "claude_gpt", Windows: []Window{{Kind: "claude_5h", Remaining: 100}, {Kind: "claude_weekly", Remaining: 100}}},
		},
	}
	// gmail: both families live
	gmail := QuotaView{
		Status: string(model.UsageCached),
		ModelGroups: []ModelGroup{
			{Key: "gemini", Windows: []Window{{Kind: "5h", Remaining: 90}, {Kind: "weekly", Remaining: 94}}},
			{Key: "claude_gpt", Windows: []Window{{Kind: "claude_5h", Remaining: 40}, {Kind: "claude_weekly", Remaining: 80}}},
		},
	}
	omegaEff, omegaKind, _ := BottleneckScore(&omega)
	gmailEff, _, _ := BottleneckScore(&gmail)
	if omegaKind != "5h" {
		t.Fatalf("omega bottleneck kind=%q want 5h", omegaKind)
	}
	// omega: usable=1, avg=(0+100)/2=50 → 150
	// gmail: usable=2, avg=(90+40)/2=65 → 265
	if omegaEff < 149 || omegaEff > 151 {
		t.Fatalf("omega effective=%v want ~150", omegaEff)
	}
	if gmailEff < 264 || gmailEff > 266 {
		t.Fatalf("gmail effective=%v want ~265", gmailEff)
	}
	if gmailEff <= omegaEff {
		t.Fatalf("gmail (%v) must beat omega (%v) — more usable model families", gmailEff, omegaEff)
	}
	omegaRatio, ok := omega.EffectiveCapacityRatio()
	gmailRatio, ok2 := gmail.EffectiveCapacityRatio()
	if !ok || !ok2 || gmailRatio <= omegaRatio {
		t.Fatalf("ratios omega=%v gmail=%v — gmail must rank higher", omegaRatio, gmailRatio)
	}
}

func TestStaleCachedSnapshotMarkedEstimatedByTrustworthy(t *testing.T) {
	engine := NewEngine(5 * time.Minute)
	rem := 70.0
	snap := model.UsageSnapshot{
		ProviderID: "codex",
		ProfileID:  "old",
		Status:     model.UsageCached,
		Source:     model.SourceLocalFiles,
		FetchedAt:  time.Now().Add(-8 * 24 * time.Hour),
		Windows:    []model.UsageWindow{{Kind: "5h", Group: "claude_gpt", RemainingPercent: &rem}},
	}
	if engine.Trustworthy(snap) {
		t.Fatal("8-day-old cache must not be Trustworthy")
	}
}

func TestUsageSnapshotRepresentsAbsoluteEvidenceWithoutInventingUnknowns(t *testing.T) {
	limit := 1000.0
	used := 125.0
	remaining := 875.0
	snapshot := model.UsageSnapshot{
		ProviderID: "codex",
		ProfileID:  "work",
		Status:     model.UsageLive,
		Source:     model.SourceOfficialAPI,
		Confidence: model.UsageConfidenceHigh,
		Windows: []model.UsageWindow{{
			Kind:      "5h",
			Limit:     &limit,
			Used:      &used,
			Remaining: &remaining,
			Unit:      "tokens",
		}},
	}
	if snapshot.Windows[0].Limit == nil || *snapshot.Windows[0].Remaining != remaining {
		t.Fatalf("absolute usage evidence was not retained: %+v", snapshot.Windows[0])
	}

	unknown := model.UsageSnapshot{Status: model.UsageUnknown, Source: model.SourceNone}
	if unknown.Windows != nil {
		t.Fatalf("unknown snapshot must not gain synthetic windows: %+v", unknown.Windows)
	}
}

func TestFormatFreshnessDays(t *testing.T) {
	got := FormatFreshness(time.Now().Add(-8 * 24 * time.Hour))
	if got != "8 days ago" {
		t.Fatalf("FormatFreshness=%q want 8 days ago", got)
	}
}

func TestSortModelGroupsUsesKeyNotDisplayName(t *testing.T) {
	groups := []ModelGroup{
		{Key: "claude_gpt", Name: "Claude & GPT Models"},
		{Key: "gemini", Name: "Gemini Models"},
	}
	SortModelGroups(groups)
	if groups[0].Key != "gemini" || groups[1].Key != "claude_gpt" {
		t.Fatalf("order=%v %v", groups[0].Key, groups[1].Key)
	}
}

func TestAGYAvailabilityKeepsProfileUsableWhenOneModelGroupHasQuota(t *testing.T) {
	qv := QuotaView{
		Provider: "agy",
		Status:   string(model.UsageCached),
		ModelGroups: []ModelGroup{
			{Name: "Gemini Models", Windows: []Window{{Kind: "5h", Remaining: 8}, {Kind: "weekly", Remaining: 17}}},
			{Name: "Claude & GPT Models", Windows: []Window{{Kind: "claude_5h", Remaining: 0}, {Kind: "claude_weekly", Remaining: 0}}},
		},
	}

	qv.ComputeAvailability()

	if !qv.IsAvailable() {
		t.Fatalf("AGY must remain available for a usable model group")
	}
	if len(qv.AvailReasons.ExhaustedWindows) != 2 {
		t.Fatalf("expected both exhausted Claude/GPT windows, got %+v", qv.AvailReasons.ExhaustedWindows)
	}
}

func TestAGYAvailabilityWhenEveryModelGroupHasQuota(t *testing.T) {
	qv := QuotaView{
		Provider: "agy",
		Status:   string(model.UsageLive),
		ModelGroups: []ModelGroup{
			{Name: "Gemini Models", Windows: []Window{{Kind: "5h", Remaining: 8}, {Kind: "weekly", Remaining: 17}}},
			{Name: "Claude & GPT Models", Windows: []Window{{Kind: "claude_5h", Remaining: 20}, {Kind: "claude_weekly", Remaining: 40}}},
		},
	}

	qv.ComputeAvailability()

	if !qv.IsAvailable() || !qv.AvailReasons.AllOK {
		t.Fatalf("AGY must be available when both model groups have quota: %+v", qv.AvailReasons)
	}
}

func TestAGYAvailabilityRejectsProfileWhenEveryModelGroupIsExhausted(t *testing.T) {
	qv := QuotaView{
		Provider: "agy",
		Status:   string(model.UsageCached),
		ModelGroups: []ModelGroup{
			{Name: "Gemini Models", Windows: []Window{{Kind: "5h", Remaining: 0}, {Kind: "weekly", Remaining: 0}}},
			{Name: "Claude & GPT Models", Windows: []Window{{Kind: "claude_5h", Remaining: 0}, {Kind: "claude_weekly", Remaining: 0}}},
		},
	}

	qv.ComputeAvailability()

	if qv.IsAvailable() {
		t.Fatalf("AGY must be unavailable when every model group is exhausted")
	}
	if len(qv.AvailReasons.ExhaustedWindows) != 4 {
		t.Fatalf("expected all exhausted windows, got %+v", qv.AvailReasons.ExhaustedWindows)
	}
}

func TestLegacyAGYQuotaPreservesRemainingPercent(t *testing.T) {
	if got := clampPercent(100); got != 100 {
		t.Fatalf("100%% remaining must stay 100%%, got %v", got)
	}
	if got := clampPercent(90.15); got < 90.14 || got > 90.16 {
		t.Fatalf("90.15%% remaining must stay 90.15%%, got %v", got)
	}
	if got := clampPercent(0); got != 0 {
		t.Fatalf("0%% remaining must stay 0%%, got %v", got)
	}
}

func TestLegacyAGYQuotaAvailableIsNotExhausted(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	profDir := dataDir + "/profiles/agy/work"
	if err := os.MkdirAll(profDir, 0700); err != nil {
		t.Fatal(err)
	}
	content := `{
		"provider": "agy",
		"profile_name": "work",
		"account": "user@example.com",
		"five_hour": { "percent_left": 0, "resets_in": "Refreshes in 1h 15m" },
		"weekly": { "percent_left": 65.47, "resets_in": "Refreshes in 142h 40m" },
		"claude_five_hour": { "percent_left": 100, "resets_in": "Quota available" },
		"claude_weekly": { "percent_left": 100, "resets_in": "Quota available" }
	}`
	if err := os.WriteFile(profDir+"/quota.json", []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	engine := NewEngine(5 * time.Minute)
	snap, found := engine.GetCachedUsage("agy", "work")
	if !found {
		t.Fatal("expected cached AGY usage")
	}
	if len(snap.Windows) != 4 {
		t.Fatalf("expected 4 windows, got %d", len(snap.Windows))
	}
	if snap.Windows[0].RemainingPercent == nil || *snap.Windows[0].RemainingPercent != 0 {
		t.Fatalf("gemini 5h remaining=%v want 0", snap.Windows[0].RemainingPercent)
	}
	if snap.Windows[2].RemainingPercent == nil || *snap.Windows[2].RemainingPercent != 100 {
		t.Fatalf("claude 5h remaining=%v want 100 (Quota available)", snap.Windows[2].RemainingPercent)
	}

	qv := BuildQuotaView(snap, "user@example.com", "Google AI Pro")
	if !qv.IsAvailable() {
		t.Fatalf("AGY must stay available while Claude/GPT still has quota: %+v", qv.AvailReasons)
	}
}

func TestCodexLegacyQuotaPreservesRemainingAndGroup(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	engine := NewEngine(5 * time.Minute)

	// Create a profile root for codex:testprof
	profDir := dataDir + "/profiles/codex/testprof"
	if err := os.MkdirAll(profDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Legacy quota JSON with 70% left — and AGY-shaped empty Claude stubs that
	// must NOT be imported into the Codex single pool (they would zero capacity).
	content := `{
		"provider": "codex",
		"profile_name": "testprof",
		"five_hour": { "percent_left": 70, "reset_time": "resets 17:34" },
		"weekly": { "percent_left": 95, "reset_time": "resets 12:34 on 3 Sep" },
		"claude_five_hour": { "percent_left": 0 },
		"claude_weekly": { "percent_left": 0 }
	}`
	if err := os.WriteFile(profDir+"/quota.json", []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	snap, found := engine.GetCachedUsage("codex", "testprof")
	if !found {
		t.Fatalf("expected cached usage found")
	}
	if len(snap.Windows) != 2 {
		t.Fatalf("expected 2 Codex windows (no AGY Claude phantoms), got %d", len(snap.Windows))
	}
	// For Codex, 70% left should remain 70%, NOT 30%
	if snap.Windows[0].RemainingPercent == nil || *snap.Windows[0].RemainingPercent != 70 {
		t.Fatalf("expected remaining 70%%, got %v", snap.Windows[0].RemainingPercent)
	}
	if snap.Windows[0].Group != "claude_gpt" {
		t.Fatalf("expected group claude_gpt, got %s", snap.Windows[0].Group)
	}
	if snap.Windows[1].RemainingPercent == nil || *snap.Windows[1].RemainingPercent != 95 {
		t.Fatalf("expected remaining 95%%, got %v", snap.Windows[1].RemainingPercent)
	}

	qv := BuildQuotaView(snap, "user@example.com", "Plus")
	if len(qv.ModelGroups) != 1 || qv.ModelGroups[0].Name != "Codex" {
		t.Fatalf("Codex quota group label=%q, want Codex", qv.ModelGroups[0].Name)
	}
	best, ok := qv.BestGroupRemaining()
	if !ok || best != 70 {
		t.Fatalf("BestGroupRemaining=%v ok=%v want 70 (min of 70/95, no phantom 0)", best, ok)
	}
}

func TestAvailabilityLabelUnknownIsSemDadosNotDisponivel(t *testing.T) {
	qv := &QuotaView{Status: string(model.UsageUnknown)}
	qv.ComputeAvailability()
	if qv.IsAvailable() {
		t.Fatal("UNKNOWN quota must not be available for automatic allocation")
	}
	if got := qv.AvailabilityLabel(); got != "SEM DADOS" {
		t.Fatalf("AvailabilityLabel=%q want SEM DADOS (never DISPONIVEL for unknown)", got)
	}
}

func TestMissingRemainingPercentIsNotSyntheticZeroExhausted(t *testing.T) {
	snap := model.UsageSnapshot{
		ProviderID: "codex",
		ProfileID:  "default",
		Status:     model.UsageLive,
		Windows: []model.UsageWindow{{
			Kind:  "5h",
			Group: "codex",
			// RemainingPercent intentionally nil
		}},
	}
	qv := BuildQuotaView(snap, "acct", "pro")
	if len(qv.ModelGroups) == 0 || len(qv.ModelGroups[0].Windows) == 0 {
		t.Fatal("expected synthetic unknown window")
	}
	w := qv.ModelGroups[0].Windows[0]
	if w.Kind != "unknown" {
		t.Fatalf("missing percent kind=%q want unknown (not measured 0%%)", w.Kind)
	}
	if qv.IsAvailable() {
		t.Fatal("missing percent must not invent available capacity")
	}
}

func TestAvailabilityLabelExhaustedFiveHour(t *testing.T) {
	zero := 0.0
	fifty := 56.0
	qv := &QuotaView{
		Status: string(model.UsageLive),
		ModelGroups: []ModelGroup{{
			Name: "Codex",
			Windows: []Window{
				{Kind: "5h", Remaining: zero, Status: string(model.UsageLive)},
				{Kind: "weekly", Remaining: fifty, Status: string(model.UsageLive)},
			},
		}},
	}
	qv.ComputeAvailability()
	if qv.IsAvailable() {
		t.Fatal("5h=0 must not be available")
	}
	if got := qv.AvailabilityLabel(); got != "QUOTA ESGOTADA" {
		t.Fatalf("AvailabilityLabel=%q want QUOTA ESGOTADA", got)
	}
}
