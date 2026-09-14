package agy

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

func TestAgyClampPercent(t *testing.T) {
	for _, tc := range []struct{ in, want float64 }{{0, 0}, {90.15, 90.15}, {100, 100}, {-5, 0}, {120, 100}} {
		if got := agyClampPercent(tc.in); math.Abs(got-tc.want) > 0.001 {
			t.Errorf("agyClampPercent(%v)=%v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseAgyQuotaOutputTreatsPercentAsRemaining(t *testing.T) {
	out := "Gemini Models\tWeekly Limit Remaining\t65.47%\t2026-09-11T04:31:30Z\n" +
		"Gemini Models\tFive Hour Limit Remaining\t0.00%\t2026-09-04T22:50:00Z\n" +
		"Claude and GPT models\tWeekly Limit Remaining\t100.00%\t\n" +
		"Claude and GPT models\tFive Hour Limit Remaining\t100.00%\tQuota available\n"
	windows, ok := parseAgyQuotaOutput(out)
	if !ok {
		t.Fatal("expected parsed windows")
	}
	if len(windows) != 4 {
		t.Fatalf("windows=%d want 4", len(windows))
	}

	byKind := map[string]float64{}
	for _, w := range windows {
		if w.RemainingPercent == nil {
			t.Fatalf("missing remaining for %s", w.Kind)
		}
		byKind[w.Kind] = *w.RemainingPercent
	}
	if byKind["5h"] != 0 {
		t.Fatalf("gemini 5h remaining=%v want 0", byKind["5h"])
	}
	if math.Abs(byKind["weekly"]-65.47) > 0.001 {
		t.Fatalf("gemini weekly remaining=%v want 65.47", byKind["weekly"])
	}
	if byKind["claude_5h"] != 100 {
		t.Fatalf("claude 5h remaining=%v want 100", byKind["claude_5h"])
	}
	if byKind["claude_weekly"] != 100 {
		t.Fatalf("claude weekly remaining=%v want 100", byKind["claude_weekly"])
	}
}

func TestParseAgyQuotaOutputAcceptsSingleFamilyTSV(t *testing.T) {
	// Two Gemini lines only — valid since agyQuotaComplete accepts single-family.
	out := "Gemini Models\tWeekly Limit Remaining\t65.47%\t2026-09-11T04:31:30Z\n" +
		"Gemini Models\tFive Hour Limit Remaining\t0.00%\t2026-09-04T22:50:00Z\n"
	windows, ok := parseAgyQuotaOutput(out)
	if !ok || windows == nil {
		t.Fatalf("single-family TSV must be accepted, ok=%v windows=%d", ok, len(windows))
	}
	if len(windows) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(windows))
	}
}

func TestLegacyAGYQuotaKeepsClaudeAtZero(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	root := dataDir + "/profiles/agy/work"
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	content := `{
		"account": "user@example.com",
		"five_hour": { "percent_left": 50, "resets_in": "Refreshes in 1h" },
		"weekly": { "percent_left": 80, "resets_in": "Refreshes in 2d" },
		"claude_five_hour": { "percent_left": 0 },
		"claude_weekly": { "percent_left": 0 }
	}`
	if err := os.WriteFile(root+"/quota.json", []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	snap, ok := New().readCachedQuotaFiles(model.Profile{Provider: "agy", Name: "work"})
	if !ok {
		t.Fatal("expected legacy cache")
	}
	if len(snap.Windows) != 4 {
		t.Fatalf("windows=%d want 4 (Claude 0%% must remain)", len(snap.Windows))
	}
}

func TestFormatAgyResetRelative(t *testing.T) {
	future := time.Now().Add(90 * time.Minute).UTC().Format(time.RFC3339)
	got := formatAgyReset(future)
	if got != "Refreshes in 1h 30m" && got != "Refreshes in 1h 29m" && got != "Refreshes in 1h 31m" {
		t.Fatalf("relative reset=%q", got)
	}
	if formatAgyReset("Quota available") != "Quota available" {
		t.Fatal("passthrough reset text")
	}
}

// --- agyQuotaComplete tests ---

func TestAgyQuotaCompleteAcceptsFullOutput(t *testing.T) {
	windows := []model.UsageWindow{
		{Kind: "5h", Group: "gemini"},
		{Kind: "weekly", Group: "gemini"},
		{Kind: "claude_5h", Group: "claude_gpt"},
		{Kind: "claude_weekly", Group: "claude_gpt"},
	}
	if !agyQuotaComplete(windows) {
		t.Fatal("full 4-window output should be accepted")
	}
}

func TestAgyQuotaCompleteAcceptsOnlyWeeklyWindows(t *testing.T) {
	// AGY CLI may omit 5h windows when the account is exhausted.
	// Only weekly windows (one per group) must still be accepted.
	windows := []model.UsageWindow{
		{Kind: "weekly", Group: "gemini"},
		{Kind: "claude_weekly", Group: "claude_gpt"},
	}
	if !agyQuotaComplete(windows) {
		t.Fatal("two weekly-only windows should be accepted")
	}
}

func TestAgyQuotaCompleteAcceptsMixedPartialWindows(t *testing.T) {
	// One group has both 5h+weekly, the other has only weekly.
	windows := []model.UsageWindow{
		{Kind: "5h", Group: "gemini"},
		{Kind: "weekly", Group: "gemini"},
		{Kind: "claude_weekly", Group: "claude_gpt"},
	}
	if !agyQuotaComplete(windows) {
		t.Fatal("mixed partial windows (3 total) should be accepted")
	}
}

func TestAgyQuotaCompleteAcceptsSingleGroup(t *testing.T) {
	// Only gemini windows — valid since agyQuotaComplete accepts single-family.
	windows := []model.UsageWindow{
		{Kind: "5h", Group: "gemini"},
		{Kind: "weekly", Group: "gemini"},
	}
	if !agyQuotaComplete(windows) {
		t.Fatal("single known group should be accepted")
	}
}

func TestAgyQuotaCompleteRejectsEmptyWindows(t *testing.T) {
	if agyQuotaComplete(nil) {
		t.Fatal("should reject nil windows")
	}
	if agyQuotaComplete([]model.UsageWindow{}) {
		t.Fatal("should reject empty windows")
	}
}

func TestAgyQuotaCompleteRejectsUnknownGroup(t *testing.T) {
	windows := []model.UsageWindow{
		{Kind: "weekly", Group: "unknown_group"},
	}
	if agyQuotaComplete(windows) {
		t.Fatal("should reject when no known groups are present")
	}
}

// --- parseAgyQuotaOutput with partial data ---

func TestParseAgyQuotaOutputAcceptsOnlyWeeklyLines(t *testing.T) {
	// Real AGY CLI output when 5h windows are exhausted/omitted.
	out := "Gemini Models\tWeekly Limit Remaining\t0%\t2026-09-13T23:06:34Z\n" +
		"Claude and GPT models\tWeekly Limit Remaining\t0%\t2026-09-13T23:46:17Z\n"
	windows, ok := parseAgyQuotaOutput(out)
	if !ok {
		t.Fatal("two weekly lines (one per group) should be accepted")
	}
	if len(windows) != 2 {
		t.Fatalf("windows=%d want 2", len(windows))
	}

	byKind := map[string]float64{}
	for _, w := range windows {
		if w.RemainingPercent == nil {
			t.Fatalf("missing remaining for %s/%s", w.Group, w.Kind)
		}
		byKind[w.Group+"/"+w.Kind] = *w.RemainingPercent
	}
	if byKind["gemini/weekly"] != 0 {
		t.Fatalf("gemini weekly remaining=%v want 0", byKind["gemini/weekly"])
	}
	if byKind["claude_gpt/claude_weekly"] != 0 {
		t.Fatalf("claude weekly remaining=%v want 0", byKind["claude_gpt/claude_weekly"])
	}
}

func TestParseAgyQuotaOutputAcceptsSingleGroupOnly(t *testing.T) {
	// Only gemini windows — valid since agyQuotaComplete accepts single-family.
	out := "Gemini Models\tWeekly Limit Remaining\t50%\t2026-09-13T23:06:34Z\n" +
		"Gemini Models\tFive Hour Limit Remaining\t100%\t2026-09-07T21:00:00Z\n"
	windows, ok := parseAgyQuotaOutput(out)
	if !ok || windows == nil {
		t.Fatalf("single-group output must be accepted, ok=%v windows=%d", ok, len(windows))
	}
}

func TestParseAgyQuotaOutputRejectsEmptyOutput(t *testing.T) {
	windows, ok := parseAgyQuotaOutput("")
	if ok || windows != nil {
		t.Fatal("empty output must be rejected")
	}
}

func TestParseAgyQuotaOutputRejectsGarbage(t *testing.T) {
	windows, ok := parseAgyQuotaOutput("not a TSV line\nanother garbage\n")
	if ok || windows != nil {
		t.Fatal("garbage output must be rejected")
	}
}

// --- readCachedQuotaFiles LIVE→CACHED conversion ---

func TestReadCachedQuotaFilesConvertsLiveToCached(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	root := dataDir + "/profiles/agy/testprofile"
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}

	content := `{
		"provider_id": "agy",
		"profile_id": "testprofile",
		"status": "LIVE",
		"source": "CLI_OUTPUT",
		"fetched_at": "2026-09-07T12:00:00Z",
		"account": "test@gmail.com",
		"windows": [
			{"kind": "weekly", "group": "gemini", "remaining_percent": 50, "used_percent": 50},
			{"kind": "claude_weekly", "group": "claude_gpt", "remaining_percent": 70, "used_percent": 30}
		]
	}`
	if err := os.WriteFile(root+"/usage.json", []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	snap, ok := New().readCachedQuotaFiles(model.Profile{Provider: "agy", Name: "testprofile"})
	if !ok {
		t.Fatal("expected to read cached quota")
	}
	if snap.Status != model.UsageCached {
		t.Fatalf("status=%v want CACHED (was LIVE in file)", snap.Status)
	}
	if snap.Account != "test@gmail.com" {
		t.Fatalf("account=%v want test@gmail.com", snap.Account)
	}
	if len(snap.Windows) != 2 {
		t.Fatalf("windows=%d want 2", len(snap.Windows))
	}
}

func TestReadCachedQuotaFilesPreservesCachedStatus(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	root := dataDir + "/profiles/agy/testprofile"
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}

	content := `{
		"provider_id": "agy",
		"profile_id": "testprofile",
		"status": "CACHED",
		"source": "CLI_OUTPUT",
		"fetched_at": "2026-09-07T12:00:00Z",
		"account": "test@gmail.com",
		"windows": [
			{"kind": "weekly", "group": "gemini", "remaining_percent": 50, "used_percent": 50},
			{"kind": "claude_weekly", "group": "claude_gpt", "remaining_percent": 70, "used_percent": 30}
		]
	}`
	if err := os.WriteFile(root+"/usage.json", []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	snap, ok := New().readCachedQuotaFiles(model.Profile{Provider: "agy", Name: "testprofile"})
	if !ok {
		t.Fatal("expected to read cached quota")
	}
	if snap.Status != model.UsageCached {
		t.Fatalf("status=%v want CACHED", snap.Status)
	}
}

func TestReadCachedQuotaFilesReturnsUnknownStatus(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	root := dataDir + "/profiles/agy/testprofile"
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}

	content := `{
		"provider_id": "agy",
		"profile_id": "testprofile",
		"status": "UNKNOWN",
		"source": "NONE",
		"fetched_at": "2026-09-07T12:00:00Z",
		"windows": [
			{"kind": "weekly", "group": "gemini", "remaining_percent": 50, "used_percent": 50}
		]
	}`
	if err := os.WriteFile(root+"/usage.json", []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	snap, ok := New().readCachedQuotaFiles(model.Profile{Provider: "agy", Name: "testprofile"})
	if !ok {
		t.Fatal("readCachedQuotaFiles should return valid snapshot even with UNKNOWN status")
	}
	if snap.Status != model.UsageUnknown {
		t.Fatalf("status=%v want UNKNOWN", snap.Status)
	}
	if len(snap.Windows) != 1 {
		t.Fatalf("windows=%d want 1", len(snap.Windows))
	}
}

// --- readCachedQuotaFiles with different accounts returns different data ---

func TestReadCachedQuotaFilesReturnsDifferentDataPerProfile(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	// Profile 1: exhausted account
	root1 := dataDir + "/profiles/agy/profile1"
	if err := os.MkdirAll(root1, 0700); err != nil {
		t.Fatal(err)
	}
	content1 := `{
		"provider_id": "agy",
		"profile_id": "profile1",
		"status": "LIVE",
		"source": "CLI_OUTPUT",
		"fetched_at": "2026-09-07T12:00:00Z",
		"account": "exhausted@gmail.com",
		"windows": [
			{"kind": "weekly", "group": "gemini", "remaining_percent": 0, "used_percent": 100},
			{"kind": "claude_weekly", "group": "claude_gpt", "remaining_percent": 0, "used_percent": 100}
		]
	}`
	if err := os.WriteFile(root1+"/usage.json", []byte(content1), 0600); err != nil {
		t.Fatal(err)
	}

	// Profile 2: active account
	root2 := dataDir + "/profiles/agy/profile2"
	if err := os.MkdirAll(root2, 0700); err != nil {
		t.Fatal(err)
	}
	content2 := `{
		"provider_id": "agy",
		"profile_id": "profile2",
		"status": "LIVE",
		"source": "CLI_OUTPUT",
		"fetched_at": "2026-09-07T12:00:00Z",
		"account": "active@gmail.com",
		"windows": [
			{"kind": "5h", "group": "gemini", "remaining_percent": 100, "used_percent": 0},
			{"kind": "weekly", "group": "gemini", "remaining_percent": 18, "used_percent": 82},
			{"kind": "claude_5h", "group": "claude_gpt", "remaining_percent": 100, "used_percent": 0},
			{"kind": "claude_weekly", "group": "claude_gpt", "remaining_percent": 32, "used_percent": 68}
		]
	}`
	if err := os.WriteFile(root2+"/usage.json", []byte(content2), 0600); err != nil {
		t.Fatal(err)
	}

	snap1, ok1 := New().readCachedQuotaFiles(model.Profile{Provider: "agy", Name: "profile1"})
	snap2, ok2 := New().readCachedQuotaFiles(model.Profile{Provider: "agy", Name: "profile2"})

	if !ok1 || !ok2 {
		t.Fatalf("expected both profiles to have cached data, ok1=%v ok2=%v", ok1, ok2)
	}

	// Verify accounts are different
	if snap1.Account == snap2.Account {
		t.Fatalf("profiles should have different accounts, both got %q", snap1.Account)
	}

	// Verify window counts are different (exhausted has 2, active has 4)
	if len(snap1.Windows) == len(snap2.Windows) {
		t.Fatalf("profiles should have different window counts, both got %d", len(snap1.Windows))
	}

	// Verify exhausted profile has 0% remaining
	for _, w := range snap1.Windows {
		if w.RemainingPercent == nil {
			t.Fatalf("exhausted profile missing remaining for %s", w.Kind)
		}
		if *w.RemainingPercent != 0 {
			t.Fatalf("exhausted profile %s remaining=%v want 0", w.Kind, *w.RemainingPercent)
		}
	}

	// Verify active profile has non-zero remaining
	hasNonZero := false
	for _, w := range snap2.Windows {
		if w.RemainingPercent != nil && *w.RemainingPercent > 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Fatal("active profile should have at least one non-zero remaining")
	}
}

// --- Integration: parseAgyQuotaOutput → agyQuotaComplete → buildSnapshot round-trip ---

func TestAgyParseCompleteRoundTripPartialWeekly(t *testing.T) {
	// Simulates the exact AGY CLI output that caused the original bug:
	// only weekly windows present (5h exhausted/omitted).
	out := "Gemini Models\tWeekly Limit Remaining\t0%\t2026-09-13T23:06:34Z\n" +
		"Claude and GPT models\tWeekly Limit Remaining\t0%\t2026-09-13T23:46:17Z\n"

	windows, ok := parseAgyQuotaOutput(out)
	if !ok {
		t.Fatal("parseAgyQuotaOutput failed on partial weekly output")
	}
	if !agyQuotaComplete(windows) {
		t.Fatal("agyQuotaComplete must accept partial weekly data from real AGY CLI")
	}

	// Verify each window is usable: has a kind, group, and remaining percent.
	for _, w := range windows {
		if w.Kind == "" {
			t.Fatal("window has empty kind")
		}
		if w.Group == "" {
			t.Fatal("window has empty group")
		}
		if w.RemainingPercent == nil {
			t.Fatalf("window %s/%s has nil remaining percent", w.Group, w.Kind)
		}
		if *w.RemainingPercent < 0 || *w.RemainingPercent > 100 {
			t.Fatalf("window %s/%s remaining percent out of range: %v", w.Group, w.Kind, *w.RemainingPercent)
		}
	}
}

func TestAgyParseCompleteRoundTripFull(t *testing.T) {
	out := "Gemini Models\tWeekly Limit Remaining\t65.47%\t2026-09-11T04:31:30Z\n" +
		"Gemini Models\tFive Hour Limit Remaining\t0.00%\t2026-09-04T22:50:00Z\n" +
		"Claude and GPT models\tWeekly Limit Remaining\t100.00%\t\n" +
		"Claude and GPT models\tFive Hour Limit Remaining\t100.00%\tQuota available\n"

	windows, ok := parseAgyQuotaOutput(out)
	if !ok {
		t.Fatal("parseAgyQuotaOutput failed on full output")
	}
	if !agyQuotaComplete(windows) {
		t.Fatal("agyQuotaComplete should accept full 4-window output")
	}
	if len(windows) != 4 {
		t.Fatalf("windows=%d want 4", len(windows))
	}

	// Verify both groups are present.
	hasGemini, hasClaude := false, false
	for _, w := range windows {
		if w.Group == "gemini" {
			hasGemini = true
		}
		if w.Group == "claude_gpt" {
			hasClaude = true
		}
	}
	if !hasGemini || !hasClaude {
		t.Fatalf("both groups must be present, gemini=%v claude=%v", hasGemini, hasClaude)
	}
}

// --- Stale cache never served when live data is available ---

func TestStaleCacheNotServedAfterSuccessfulLiveParse(t *testing.T) {
	// This is the core regression test for the original bug.
	// Scenario: stale cache exists on disk, but AGY CLI returns fresh data.
	// The fresh data must be used, not the stale cache.

	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	profile := model.Profile{Provider: "agy", Name: "regression-test"}
	profileDir := filepath.Join(dataDir, "profiles", "agy", "regression-test")
	if err := os.MkdirAll(profileDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Write stale cache (old data, 100% remaining).
	staleContent := `{
		"provider_id": "agy",
		"profile_id": "regression-test",
		"status": "LIVE",
		"source": "CLI_OUTPUT",
		"fetched_at": "2026-01-01T00:00:00Z",
		"account": "stale@gmail.com",
		"windows": [
			{"kind": "weekly", "group": "gemini", "remaining_percent": 100, "used_percent": 0},
			{"kind": "claude_weekly", "group": "claude_gpt", "remaining_percent": 100, "used_percent": 0}
		]
	}`
	if err := os.WriteFile(filepath.Join(profileDir, "usage.json"), []byte(staleContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Step 1: Verify stale cache is read.
	snap, ok := New().readCachedQuotaFiles(profile)
	if !ok {
		t.Fatal("stale cache should be readable")
	}
	if snap.Account != "stale@gmail.com" {
		t.Fatalf("expected stale account, got %q", snap.Account)
	}

	// Step 2: Simulate fresh AGY CLI output (0% remaining, exhausted).
	freshOut := "Gemini Models\tWeekly Limit Remaining\t0%\t2026-09-13T23:06:34Z\n" +
		"Claude and GPT models\tWeekly Limit Remaining\t0%\t2026-09-13T23:46:17Z\n"
	windows, ok := parseAgyQuotaOutput(freshOut)
	if !ok {
		t.Fatal("parseAgyQuotaOutput failed on fresh output")
	}
	if !agyQuotaComplete(windows) {
		t.Fatal("agyQuotaComplete should accept fresh partial data")
	}

	// Step 3: Build snapshot from fresh data and write to disk.
	freshSnap := model.UsageSnapshot{
		ProviderID: profile.Provider,
		ProfileID:  profile.Name,
		Status:     model.UsageLive,
		Source:     model.SourceCLIOutput,
		FetchedAt:  time.Now().UTC(),
		Account:    "fresh@gmail.com",
		Windows:    windows,
	}
	freshData, err := json.Marshal(freshSnap)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "usage.json"), freshData, 0600); err != nil {
		t.Fatal(err)
	}

	// Step 4: Verify stale cache is gone — fresh data must be returned.
	snapAfter, ok := New().readCachedQuotaFiles(profile)
	if !ok {
		t.Fatal("fresh data should be readable after write")
	}
	if snapAfter.Account != "fresh@gmail.com" {
		t.Fatalf("stale data served! account=%q want fresh@gmail.com", snapAfter.Account)
	}
	if snapAfter.Status != model.UsageCached {
		t.Fatalf("LIVE status must be converted to CACHED, got %v", snapAfter.Status)
	}

	// Verify fresh data has 0% remaining, not 100% from stale cache.
	for _, w := range snapAfter.Windows {
		if w.RemainingPercent == nil {
			t.Fatalf("window %s/%s has nil remaining", w.Group, w.Kind)
		}
		if *w.RemainingPercent != 0 {
			t.Fatalf("stale data leaked! %s/%s remaining=%v want 0", w.Group, w.Kind, *w.RemainingPercent)
		}
	}
}

// --- formatAgyReset edge cases ---

func TestFormatAgyResetPastTimeReturnsRefreshing(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	got := formatAgyReset(past)
	if got != "Quota available" {
		t.Fatalf("past time reset=%q want 'Quota available'", got)
	}
}

func TestFormatAgyResetZeroDuration(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	got := formatAgyReset(now)
	if got != "Quota available" {
		t.Fatalf("zero duration reset=%q want 'Quota available'", got)
	}
}

func TestFormatAgyResetPassthrough(t *testing.T) {
	nonRFC := "Quota available"
	if got := formatAgyReset(nonRFC); got != nonRFC {
		t.Fatalf("non-RFC3339 text should pass through, got %q", got)
	}
	empty := ""
	if got := formatAgyReset(empty); got != "" {
		t.Fatalf("empty string should pass through, got %q", got)
	}
}

// --- Profile isolation: different profiles never share data ---

func TestProfileIsolationDifferentKeyringPaths(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	// Create two profiles with different data.
	for _, p := range []struct {
		name      string
		account   string
		remaining float64
	}{
		{"profileA", "a@gmail.com", 50},
		{"profileB", "b@gmail.com", 0},
	} {
		dir := filepath.Join(dataDir, "profiles", "agy", p.name)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		content := `{
			"provider_id": "agy",
			"profile_id": "` + p.name + `",
			"status": "LIVE",
			"source": "CLI_OUTPUT",
			"fetched_at": "2026-09-07T12:00:00Z",
			"account": "` + p.account + `",
			"windows": [
				{"kind": "weekly", "group": "gemini", "remaining_percent": ` + fmt.Sprintf("%.0f", p.remaining) + `},
				{"kind": "claude_weekly", "group": "claude_gpt", "remaining_percent": ` + fmt.Sprintf("%.0f", p.remaining) + `}
			]
		}`
		if err := os.WriteFile(filepath.Join(dir, "usage.json"), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	// Read both profiles and verify complete isolation.
	snapA, okA := New().readCachedQuotaFiles(model.Profile{Provider: "agy", Name: "profileA"})
	snapB, okB := New().readCachedQuotaFiles(model.Profile{Provider: "agy", Name: "profileB"})

	if !okA || !okB {
		t.Fatalf("both should be readable, okA=%v okB=%v", okA, okB)
	}

	// Accounts must differ.
	if snapA.Account == snapB.Account {
		t.Fatalf("profiles share account! both=%q", snapA.Account)
	}

	// Profile IDs must differ.
	if snapA.ProfileID == snapB.ProfileID {
		t.Fatalf("profiles share profile_id! both=%q", snapA.ProfileID)
	}

	// Remaining percents must differ.
	for _, wA := range snapA.Windows {
		for _, wB := range snapB.Windows {
			if wA.Kind == wB.Kind && wA.Group == wB.Group {
				if wA.RemainingPercent == nil || wB.RemainingPercent == nil {
					t.Fatalf("nil remaining for %s/%s", wA.Group, wA.Kind)
				}
				if *wA.RemainingPercent == *wB.RemainingPercent {
					t.Fatalf("profiles share remaining for %s/%s: both=%v", wA.Group, wA.Kind, *wA.RemainingPercent)
				}
			}
		}
	}
}

// --- Trustworthy threshold: cached data older than 1h is untrusted ---

func TestTrustworthyRejectsOldCache(t *testing.T) {
	oldSnap := model.UsageSnapshot{
		Status:    model.UsageCached,
		FetchedAt: time.Now().Add(-2 * time.Hour),
		Source:    model.SourceCLIOutput,
	}
	if quota.NewEngine(time.Hour).Trustworthy(oldSnap) {
		t.Fatal("2-hour-old cache should be untrustworthy")
	}
}

func TestTrustworthyAcceptsFreshCache(t *testing.T) {
	freshSnap := model.UsageSnapshot{
		Status:    model.UsageCached,
		FetchedAt: time.Now().Add(-30 * time.Minute),
		Source:    model.SourceCLIOutput,
		Windows:   []model.UsageWindow{{Kind: "weekly", Group: "gemini", RemainingPercent: func() *float64 { value := 50.0; return &value }()}},
	}
	if !quota.NewEngine(time.Hour).Trustworthy(freshSnap) {
		t.Fatal("30-minute-old cache should be trustworthy")
	}
}

func TestTrustworthyRejectsLiveStatus(t *testing.T) {
	liveSnap := model.UsageSnapshot{
		Status:    model.UsageLive,
		FetchedAt: time.Now(),
		Source:    model.SourceCLIOutput,
	}
	if quota.NewEngine(time.Hour).Trustworthy(liveSnap) {
		t.Fatal("LIVE status should be untrustworthy (must convert to CACHED first)")
	}
}

func TestTrustworthyRejectsEstimatedStatus(t *testing.T) {
	estSnap := model.UsageSnapshot{
		Status:    model.UsageEstimated,
		FetchedAt: time.Now(),
		Source:    model.SourceCLIOutput,
	}
	if quota.NewEngine(time.Hour).Trustworthy(estSnap) {
		t.Fatal("ESTIMATED status should be untrustworthy")
	}
}

func TestTrustworthyRejectsUnknownStatus(t *testing.T) {
	unkSnap := model.UsageSnapshot{
		Status:    model.UsageUnknown,
		FetchedAt: time.Now(),
		Source:    model.SourceNone,
	}
	if quota.NewEngine(time.Hour).Trustworthy(unkSnap) {
		t.Fatal("UNKNOWN status should be untrustworthy")
	}
}

// --- GetCachedUsage rejects stale data ---

func TestGetCachedUsageRejectsExpiredCache(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	profile := model.Profile{Provider: "agy", Name: "expired-test"}
	profileDir := filepath.Join(dataDir, "profiles", "agy", "expired-test")
	if err := os.MkdirAll(profileDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Write cache with old fetched_at (3 hours ago).
	oldCache := fmt.Sprintf(`{
		"provider_id": "agy",
		"profile_id": "expired-test",
		"status": "CACHED",
		"source": "CLI_OUTPUT",
		"fetched_at": %q,
		"account": "old@gmail.com",
		"windows": [
			{"kind": "weekly", "group": "gemini", "remaining_percent": 80},
			{"kind": "claude_weekly", "group": "claude_gpt", "remaining_percent": 80}
		]
	}`, time.Now().Add(-3*time.Hour).UTC().Format(time.RFC3339))
	if err := os.WriteFile(filepath.Join(profileDir, "usage.json"), []byte(oldCache), 0600); err != nil {
		t.Fatal(err)
	}

	e := quota.NewEngine(time.Hour)
	snap, ok := e.GetCachedUsage(profile.Provider, profile.Name)
	if ok {
		t.Fatalf("3-hour-old cache should be rejected, got snap with status=%v", snap.Status)
	}
	if snap.Status != model.UsageUnknown {
		t.Fatalf("rejected cache should return UNKNOWN status, got %v", snap.Status)
	}
}

func TestGetCachedUsageAcceptsFreshCache(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	profile := model.Profile{Provider: "agy", Name: "fresh-cache-test"}
	profileDir := filepath.Join(dataDir, "profiles", "agy", "fresh-cache-test")
	if err := os.MkdirAll(profileDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Write cache with recent fetched_at (30 minutes ago).
	freshCache := fmt.Sprintf(`{
		"provider_id": "agy",
		"profile_id": "fresh-cache-test",
		"status": "LIVE",
		"source": "CLI_OUTPUT",
		"fetched_at": %q,
		"account": "fresh@gmail.com",
		"windows": [
			{"kind": "weekly", "group": "gemini", "remaining_percent": 50},
			{"kind": "claude_weekly", "group": "claude_gpt", "remaining_percent": 70}
		]
	}`, time.Now().Add(-30*time.Minute).UTC().Format(time.RFC3339))
	if err := os.WriteFile(filepath.Join(profileDir, "usage.json"), []byte(freshCache), 0600); err != nil {
		t.Fatal(err)
	}

	e := quota.NewEngine(time.Hour)
	snap, ok := e.GetCachedUsage(profile.Provider, profile.Name)
	if !ok {
		t.Fatal("30-minute-old cache should be accepted")
	}
	if snap.Status != model.UsageCached {
		t.Fatalf("status=%v want CACHED", snap.Status)
	}
	if snap.Account != "fresh@gmail.com" {
		t.Fatalf("account=%v want fresh@gmail.com", snap.Account)
	}
}

// --- Account mismatch: stale cache from wrong profile is rejected ---

func TestReadCachedQuotaFilesRejectsAccountMismatch(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	// Simulate a profile whose authenticated email is "real@gmail.com"
	// but the cache file contains data for "wrong@gmail.com".
	profileDir := filepath.Join(dataDir, "profiles", "agy", "mismatch-test")
	homeDir := filepath.Join(profileDir, "home")
	if err := os.MkdirAll(homeDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Write google_accounts.json with the real authenticated email.
	accountsDir := filepath.Join(homeDir, ".gemini")
	if err := os.MkdirAll(accountsDir, 0700); err != nil {
		t.Fatal(err)
	}
	accountsContent := `{"active": "real@gmail.com"}`
	if err := os.WriteFile(filepath.Join(accountsDir, "google_accounts.json"), []byte(accountsContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Write usage.json with a DIFFERENT account (stale cross-profile cache).
	usageContent := `{
		"provider_id": "agy",
		"profile_id": "mismatch-test",
		"status": "LIVE",
		"source": "CLI_OUTPUT",
		"fetched_at": "2026-09-07T12:00:00Z",
		"account": "wrong@gmail.com",
		"windows": [
			{"kind": "weekly", "group": "gemini", "remaining_percent": 50, "used_percent": 50},
			{"kind": "claude_weekly", "group": "claude_gpt", "remaining_percent": 70, "used_percent": 30}
		]
	}`
	if err := os.WriteFile(filepath.Join(profileDir, "usage.json"), []byte(usageContent), 0600); err != nil {
		t.Fatal(err)
	}

	// readCachedQuotaFiles should reject the cache because account doesn't match.
	snap, ok := New().readCachedQuotaFiles(model.Profile{Provider: "agy", Name: "mismatch-test"})
	if ok {
		t.Fatalf("account mismatch should be rejected, got snap with account=%q", snap.Account)
	}
	if snap.Status != "" && snap.Status != model.UsageUnknown {
		t.Fatalf("rejected snap should have empty/UNKNOWN status, got %v", snap.Status)
	}
}

func TestReadCachedQuotaFilesAcceptsMatchingAccount(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	profileDir := filepath.Join(dataDir, "profiles", "agy", "match-test")
	homeDir := filepath.Join(profileDir, "home")
	if err := os.MkdirAll(homeDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Write google_accounts.json with the same email as the cache.
	accountsDir := filepath.Join(homeDir, ".gemini")
	if err := os.MkdirAll(accountsDir, 0700); err != nil {
		t.Fatal(err)
	}
	accountsContent := `{"active": "same@gmail.com"}`
	if err := os.WriteFile(filepath.Join(accountsDir, "google_accounts.json"), []byte(accountsContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Write usage.json with the SAME account.
	usageContent := `{
		"provider_id": "agy",
		"profile_id": "match-test",
		"status": "LIVE",
		"source": "CLI_OUTPUT",
		"fetched_at": "2026-09-07T12:00:00Z",
		"account": "same@gmail.com",
		"windows": [
			{"kind": "weekly", "group": "gemini", "remaining_percent": 50, "used_percent": 50},
			{"kind": "claude_weekly", "group": "claude_gpt", "remaining_percent": 70, "used_percent": 30}
		]
	}`
	if err := os.WriteFile(filepath.Join(profileDir, "usage.json"), []byte(usageContent), 0600); err != nil {
		t.Fatal(err)
	}

	snap, ok := New().readCachedQuotaFiles(model.Profile{Provider: "agy", Name: "match-test"})
	if !ok {
		t.Fatal("matching account should be accepted")
	}
	if snap.Account != "same@gmail.com" {
		t.Fatalf("account=%v want same@gmail.com", snap.Account)
	}
}

func TestReadCachedQuotaFilesAcceptsWhenNoAuthEmail(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	profileDir := filepath.Join(dataDir, "profiles", "agy", "no-auth-test")
	if err := os.MkdirAll(profileDir, 0700); err != nil {
		t.Fatal(err)
	}

	// No google_accounts.json — authEmail will be "".
	// Cache should still be accepted (no email to compare against).
	usageContent := `{
		"provider_id": "agy",
		"profile_id": "no-auth-test",
		"status": "LIVE",
		"source": "CLI_OUTPUT",
		"fetched_at": "2026-09-07T12:00:00Z",
		"account": "any@gmail.com",
		"windows": [
			{"kind": "weekly", "group": "gemini", "remaining_percent": 50, "used_percent": 50},
			{"kind": "claude_weekly", "group": "claude_gpt", "remaining_percent": 70, "used_percent": 30}
		]
	}`
	if err := os.WriteFile(filepath.Join(profileDir, "usage.json"), []byte(usageContent), 0600); err != nil {
		t.Fatal(err)
	}

	snap, ok := New().readCachedQuotaFiles(model.Profile{Provider: "agy", Name: "no-auth-test"})
	if !ok {
		t.Fatal("cache without auth email should be accepted")
	}
	if snap.Account != "any@gmail.com" {
		t.Fatalf("account=%v want any@gmail.com", snap.Account)
	}
}
