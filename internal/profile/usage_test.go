package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/testutil"
)

func TestGetQuotaDetails(t *testing.T) {
	tempData := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", tempData)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	// Without cached data, quota must report UNKNOWN, NOT 100%
	qAgy := GetQuotaDetails("agy", "test-profile-1", "Google AI Pro", "test@gmail.com")
	if qAgy.Status != string(model.UsageUnknown) {
		t.Fatalf("expected UNKNOWN status when quota is missing, got: %+v", qAgy)
	}

	qCodex := GetQuotaDetails("codex", "test-profile-2", "ChatGPT Plus", "test@openai.com")
	if qCodex.Status != string(model.UsageUnknown) {
		t.Fatalf("expected UNKNOWN status when quota is missing, got: %+v", qCodex)
	}

	// Bar rendering test
	b := RenderBar(70.0, 20)
	if len(b) == 0 {
		t.Fatal("empty rendered bar")
	}
}

func TestGetUsageSnapshotPreservesRateLimitedLastKnown(t *testing.T) {
	tempData := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", tempData)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv("NEXUS_CODEX_APP_SERVER", "0")
	if _, err := Create("agy", "blocked"); err != nil {
		t.Fatal(err)
	}
	home, _ := Home("agy", "blocked")
	if err := os.MkdirAll(filepath.Join(home, ".gemini"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".gemini", "google_accounts.json"), []byte(`{"active":"blocked@example.test"}`), 0600); err != nil {
		t.Fatal(err)
	}

	remaining := 90.0
	used := 10.0
	eng := quota.NewEngine(time.Minute)
	scope, err := AccountScope("agy", "blocked", "blocked@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.SaveUsageForScope(scope, model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "blocked",
		Status:     model.UsageRateLimited,
		Source:     model.SourceOfficialAPI,
		FetchedAt:  time.Now().Add(-time.Hour),
		Error:      "provedor bloqueou uso incluso",
		Windows: []model.UsageWindow{{
			Kind:             "5h",
			Group:            "gemini",
			RemainingPercent: &remaining,
			UsedPercent:      &used,
		}},
	}); err != nil {
		t.Fatal(err)
	}

	snap := GetUsageSnapshot("agy", "blocked")
	if snap.Status != model.UsageRateLimited {
		t.Fatalf("status=%s, want RATE_LIMITED preserved from last-known official block", snap.Status)
	}
	view := quota.BuildQuotaView(snap, "blocked@example.test", "")
	if view.Available {
		t.Fatal("RATE_LIMITED last-known must keep Available=false")
	}
}

func TestGetUsageSnapshotPreservesStaleWindowsAsEstimatedAfterRefreshFailure(t *testing.T) {
	tempData := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", tempData)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	if _, err := Create("agy", "work"); err != nil {
		t.Fatal(err)
	}
	home, _ := Home("agy", "work")
	if err := os.MkdirAll(filepath.Join(home, ".gemini"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".gemini", "google_accounts.json"), []byte(`{"active":"work@example.test"}`), 0600); err != nil {
		t.Fatal(err)
	}

	remaining := 66.0
	used := 34.0
	eng := quota.NewEngine(time.Minute)
	scope, err := AccountScope("agy", "work", "work@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.SaveUsageForScope(scope, model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "work",
		Status:     model.UsageLive,
		Source:     model.SourceCLIOutput,
		FetchedAt:  time.Now().Add(-time.Hour),
		Windows: []model.UsageWindow{{
			Kind:             "5h",
			Group:            "gemini",
			RemainingPercent: &remaining,
			UsedPercent:      &used,
		}},
	}); err != nil {
		t.Fatal(err)
	}

	snap := GetUsageSnapshot("agy", "work")
	if snap.Status != model.UsageEstimated {
		t.Fatalf("status=%s, want ESTIMATED for stale cache fallback", snap.Status)
	}
	if len(snap.Windows) != 1 {
		t.Fatalf("expected last-known windows to be preserved, got %+v", snap.Windows)
	}
	if snap.Error == "" {
		t.Fatal("expected diagnostic explaining the degraded snapshot")
	}
}

func TestGetUsageSnapshotKeepsCodexLiveRolloutWithoutPrefixedScope(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	if _, err := Create("codex", "omega"); err != nil {
		t.Fatal(err)
	}
	home, err := Home("codex", "omega")
	if err != nil {
		t.Fatal(err)
	}
	sessDir := filepath.Join(home, ".codex", "sessions", "2026", "09", "10")
	if err := os.MkdirAll(sessDir, 0o700); err != nil {
		t.Fatal(err)
	}
	auth := `{"tokens":{"id_token":"eyJhbGciOiJIUzI1NiJ9.eyJlbWFpbCI6ImtpdmVyQG9tZWdhc2lzdGVtYXMubmV0LmJyIiwiaHR0cHM6Ly9hcGkub3BlbmFpLmNvbS9hdXRoIjp7ImNoYXRncHRfcGxhbl90eXBlIjoicGx1cyJ9fQ.sig"}}`
	if err := os.WriteFile(filepath.Join(home, "auth.json"), []byte(auth), 0o600); err != nil {
		t.Fatal(err)
	}
	rollout := `{"type":"session_meta","payload":{"session_id":"s1"}}
{"type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":10.0,"window_minutes":300,"resets_at":1788556157},"secondary":{"used_percent":20.0,"window_minutes":10080,"resets_at":1788748886}}}}
`
	if err := os.WriteFile(filepath.Join(sessDir, "rollout-live.jsonl"), []byte(rollout), 0o600); err != nil {
		t.Fatal(err)
	}

	snap := RefreshUsageSnapshot("codex", "omega")
	if snap.Status != model.UsageLive {
		t.Fatalf("status=%s want LIVE (adapter rollout must not be discarded for missing AccountScope)", snap.Status)
	}
	if len(snap.Windows) != 2 {
		t.Fatalf("windows=%d want 2", len(snap.Windows))
	}
	if !snap.AccountScope.Verifiable() {
		t.Fatal("live Codex snapshot should be stamped with the profile account scope")
	}
}

// codexCacheFixture prepares an isolated codex profile whose only local evidence
// is a rollout, plus a cached snapshot with deliberately different numbers so the
// two sources can be told apart in the result.
func codexCacheFixture(t *testing.T, cacheSource model.UsageSource, age time.Duration) {
	t.Helper()
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("NEXUS_CODEX_APP_SERVER", "0")

	if _, err := Create("codex", "ttl"); err != nil {
		t.Fatal(err)
	}
	home, err := Home("codex", "ttl")
	if err != nil {
		t.Fatal(err)
	}
	auth := `{"tokens":{"id_token":"eyJhbGciOiJIUzI1NiJ9.eyJlbWFpbCI6InR0bEBleGFtcGxlLmNvbSIsImh0dHBzOi8vYXBpLm9wZW5haS5jb20vYXV0aCI6eyJjaGF0Z3B0X3BsYW5fdHlwZSI6InBsdXMifX0.sig"}}`
	if err := os.WriteFile(filepath.Join(home, "auth.json"), []byte(auth), 0o600); err != nil {
		t.Fatal(err)
	}
	sessDir := filepath.Join(home, "sessions")
	if err := os.MkdirAll(sessDir, 0o700); err != nil {
		t.Fatal(err)
	}
	rollout := fmt.Sprintf(`{"timestamp":%q,"type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":10,"window_minutes":300,"resets_at":%d}}}}
`, time.Now().UTC().Format(time.RFC3339Nano), time.Now().Add(time.Hour).Unix())
	if err := os.WriteFile(filepath.Join(sessDir, "rollout-ttl.jsonl"), []byte(rollout), 0o600); err != nil {
		t.Fatal(err)
	}

	scope, err := AccountScope("codex", "ttl", "ttl@example.com")
	if err != nil {
		t.Fatal(err)
	}
	cachedRemaining := 42.0
	cachedUsed := 58.0
	if err := quota.NewEngine(time.Minute).SaveUsageForScope(scope, model.UsageSnapshot{
		ProviderID: "codex",
		ProfileID:  "ttl",
		Status:     model.UsageLive,
		Source:     cacheSource,
		FetchedAt:  time.Now().Add(-age),
		Windows: []model.UsageWindow{{
			Kind:             "5h",
			Group:            "claude_gpt",
			RemainingPercent: &cachedRemaining,
			UsedPercent:      &cachedUsed,
		}},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestCodexQuotaOlderThanWeeklyWindowIsNotDisplayedAsCurrent(t *testing.T) {
	codexCacheFixture(t, model.SourceOfficialAPI, 25*24*time.Hour)

	// The persisted evidence remains available for history, but usage/status
	// cannot display an old 12% weekly value as if it were current.
	snap := GetUsageSnapshot("codex", "ttl")
	if snap.Status != model.UsageUnknown {
		t.Fatalf("25-day-old snapshot status=%s, want UNKNOWN", snap.Status)
	}
	if len(snap.Windows) != 0 {
		t.Fatalf("25-day-old snapshot contains misleading quota windows: %+v", snap.Windows)
	}

	// A live, isolated Codex rollout can still replace the historical reading.
	refreshed := RefreshUsageSnapshot("codex", "ttl")
	if refreshed.Status != model.UsageLive {
		t.Fatalf("refreshed status=%s, want LIVE", refreshed.Status)
	}
	if len(refreshed.Windows) != 1 || refreshed.Windows[0].RemainingPercent == nil || *refreshed.Windows[0].RemainingPercent != 90 {
		t.Fatalf("refreshed quota = %+v, want current rollout remaining 90%%", refreshed.Windows)
	}
}

func TestGetUsageSnapshotReusesFreshOfficialCodexReading(t *testing.T) {
	codexCacheFixture(t, model.SourceOfficialAPI, 5*time.Second)

	snap := GetUsageSnapshot("codex", "ttl")
	if len(snap.Windows) != 1 {
		t.Fatalf("windows=%d want 1", len(snap.Windows))
	}
	if *snap.Windows[0].RemainingPercent != 42 {
		t.Fatalf("remaining=%v want the cached official 42; an official read inside the TTL must not re-probe", *snap.Windows[0].RemainingPercent)
	}
}

func TestGetUsageSnapshotIgnoresRolloutDerivedCodexCache(t *testing.T) {
	codexCacheFixture(t, model.SourceLocalFiles, 5*time.Second)

	// GetUsageSnapshot never probes — it returns the trustworthy cache as-is.
	// Rollout re-reads belong in RefreshUsageSnapshot.
	snap := GetUsageSnapshot("codex", "ttl")
	if len(snap.Windows) != 1 {
		t.Fatalf("windows=%d want 1", len(snap.Windows))
	}
	if *snap.Windows[0].RemainingPercent != 42 {
		t.Fatalf("remaining=%v want cached 42; paint path must not walk rollouts", *snap.Windows[0].RemainingPercent)
	}
}

func TestGetUsageSnapshotRefetchesOfficialCodexReadingAfterTTL(t *testing.T) {
	codexCacheFixture(t, model.SourceOfficialAPI, 3*time.Minute)

	// Past TTL is still cache for GetUsageSnapshot; live re-probe is Refresh only.
	snap := GetUsageSnapshot("codex", "ttl")
	if len(snap.Windows) != 1 {
		t.Fatalf("windows=%d want 1", len(snap.Windows))
	}
	if *snap.Windows[0].RemainingPercent != 42 {
		t.Fatalf("remaining=%v want cached official 42; GetUsageSnapshot must not re-probe", *snap.Windows[0].RemainingPercent)
	}
}

func TestRefreshUsageSnapshotReprobesCodexAfterStaleOfficialCache(t *testing.T) {
	codexCacheFixture(t, model.SourceOfficialAPI, 3*time.Minute)

	snap := RefreshUsageSnapshot("codex", "ttl")
	if len(snap.Windows) != 1 {
		t.Fatalf("windows=%d want 1", len(snap.Windows))
	}
	if *snap.Windows[0].RemainingPercent != 90 {
		t.Fatalf("remaining=%v want the rollout 90; Refresh must re-read", *snap.Windows[0].RemainingPercent)
	}
}

func TestPeekUsageSnapshotKeepsOfficialCacheWithoutLiveProbe(t *testing.T) {
	codexCacheFixture(t, model.SourceOfficialAPI, 3*time.Minute)

	snap := PeekUsageSnapshot("codex", "ttl")
	if len(snap.Windows) != 1 {
		t.Fatalf("windows=%d want 1", len(snap.Windows))
	}
	if *snap.Windows[0].RemainingPercent != 42 {
		t.Fatalf("remaining=%v want cached official 42; launch must not spawn app-server or walk rollouts", *snap.Windows[0].RemainingPercent)
	}
}

func TestPeekUsageSnapshotCacheMissDoesNotProbeLive(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	if _, err := Create("agy", "empty"); err != nil {
		t.Fatal(err)
	}
	home, err := Home("agy", "empty")
	if err != nil {
		t.Fatal(err)
	}
	// Authenticated identity without any usage cache — a live AGY probe would
	// spawn `agy --print=/quota` and take seconds.
	if err := os.WriteFile(filepath.Join(home, ".gemini", "google_accounts.json"), []byte(`{"active":"empty@example.com"}`), 0o600); err != nil {
		_ = os.MkdirAll(filepath.Join(home, ".gemini"), 0o700)
		if err := os.WriteFile(filepath.Join(home, ".gemini", "google_accounts.json"), []byte(`{"active":"empty@example.com"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	start := time.Now()
	snap := PeekUsageSnapshot("agy", "empty")
	elapsed := time.Since(start)
	if elapsed > 500*time.Millisecond {
		t.Fatalf("Peek took %s — cache miss must not spawn live AGY quota", elapsed)
	}
	if snap.Status != model.UsageUnknown && snap.Status != model.UsageEstimated {
		t.Fatalf("status=%s want UNKNOWN/ESTIMATED on cache miss", snap.Status)
	}
	if len(snap.Windows) != 0 {
		t.Fatalf("windows=%d want 0 on cache miss Peek", len(snap.Windows))
	}
}

func TestGetUsageSnapshotKeepsAdapterEstimatedInsteadOfSemDados(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	if _, err := Create("codex", "aged"); err != nil {
		t.Fatal(err)
	}
	home, err := Home("codex", "aged")
	if err != nil {
		t.Fatal(err)
	}
	// Email-only auth keeps the official probe out of the way; the rollout below
	// is the only evidence available.
	auth := `{"tokens":{"id_token":"eyJhbGciOiJIUzI1NiJ9.eyJlbWFpbCI6ImFnZWRAZXhhbXBsZS5jb20iLCJodHRwczovL2FwaS5vcGVuYWkuY29tL2F1dGgiOnsiY2hhdGdwdF9wbGFuX3R5cGUiOiJwbHVzIn19.sig"}}`
	if err := os.WriteFile(filepath.Join(home, "auth.json"), []byte(auth), 0o600); err != nil {
		t.Fatal(err)
	}
	sessDir := filepath.Join(home, "sessions")
	if err := os.MkdirAll(sessDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// An observation 20 minutes old: past the 5 minute trust window, but real.
	eventTime := time.Now().Add(-20 * time.Minute)
	rollout := fmt.Sprintf(`{"timestamp":%q,"type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":30,"window_minutes":300,"resets_at":%d}}}}
`, eventTime.UTC().Format(time.RFC3339Nano), time.Now().Add(time.Hour).Unix())
	path := filepath.Join(sessDir, "rollout-aged.jsonl")
	if err := os.WriteFile(path, []byte(rollout), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, eventTime, eventTime); err != nil {
		t.Fatal(err)
	}

	snap := RefreshUsageSnapshot("codex", "aged")
	if snap.Status != model.UsageEstimated {
		t.Fatalf("status=%s want ESTIMATED; a 20 minute old observation must not become SEM DADOS", snap.Status)
	}
	if len(snap.Windows) != 1 {
		t.Fatalf("windows=%d want the observed window preserved", len(snap.Windows))
	}
	if *snap.Windows[0].RemainingPercent != 70 {
		t.Fatalf("remaining=%v want 70", *snap.Windows[0].RemainingPercent)
	}
	if snap.Error == "" {
		t.Fatal("expected a freshness diagnostic on an estimated snapshot")
	}
}

func TestPeekUsageSnapshotCacheMissDoesNotProbe(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	if _, err := Create("claude", "empty"); err != nil {
		t.Fatal(err)
	}
	// No usage.json and no OAuth credentials: Peek must return UNKNOWN without
	// attempting Anthropic HTTP (which would hang or fail without a stub).
	start := time.Now()
	snap := PeekUsageSnapshot("claude", "empty")
	if time.Since(start) > 2*time.Second {
		t.Fatalf("Peek took %s; cache miss must not live-probe", time.Since(start))
	}
	if snap.Status != model.UsageUnknown {
		t.Fatalf("status=%s want UNKNOWN on cache miss", snap.Status)
	}
}

func TestSnapshotBelongsToProfileRejectsForeignScope(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	if _, err := Create("codex", "omega"); err != nil {
		t.Fatal(err)
	}
	scope, err := AccountScope("codex", "omega", "kiver@omegasistemas.net.br")
	if err != nil {
		t.Fatal(err)
	}
	foreign := scope
	foreign.AccountID = "other-account"
	snap := model.UsageSnapshot{
		ProviderID:   "codex",
		ProfileID:    "omega",
		Account:      "kiver@omegasistemas.net.br",
		AccountScope: foreign,
		Status:       model.UsageLive,
	}
	if snapshotBelongsToProfile(snap, "codex", "omega", "kiver@omegasistemas.net.br") {
		t.Fatal("foreign verifiable scope must be rejected")
	}
	snap.AccountScope = model.AccountScope{}
	if !snapshotBelongsToProfile(snap, "codex", "omega", "kiver@omegasistemas.net.br") {
		t.Fatal("live adapter snapshot without AccountScope must be accepted when email matches")
	}
}

func TestSnapshotBelongsToProfileAllowsCodexIdentityVersionBump(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	if _, err := Create("codex", "gmail"); err != nil {
		t.Fatal(err)
	}
	// First scope with email identity.
	oldScope, err := AccountScope("codex", "gmail", "kiver.omegaedu@gmail.com")
	if err != nil {
		t.Fatal(err)
	}
	// Bump identity to chatgpt_account_id (same durable AccountID, new version).
	newScope, err := AccountScope("codex", "gmail", "9cba8371-db95-468c-ad00-02e9ceddcb3e")
	if err != nil {
		t.Fatal(err)
	}
	if oldScope.AccountID != newScope.AccountID {
		t.Fatalf("durable AccountID changed: %s → %s", oldScope.AccountID, newScope.AccountID)
	}
	if oldScope.IdentityVersion == newScope.IdentityVersion {
		t.Fatal("expected identity version bump")
	}
	snap := model.UsageSnapshot{
		ProviderID:   "codex",
		ProfileID:    "gmail",
		Account:      "kiver.omegaedu@gmail.com",
		AccountScope: oldScope,
		Status:       model.UsageLive,
		Windows:      []model.UsageWindow{{Kind: "5h"}},
	}
	if !snapshotBelongsToProfile(snap, "codex", "gmail", "9cba8371-db95-468c-ad00-02e9ceddcb3e") {
		t.Fatal("Codex email→account_id identity version bump must still accept last-known usage")
	}
}

func TestSnapshotBelongsToProfileRejectsCodexIdentityOutsideHistory(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	if _, err := Create("codex", "gmail"); err != nil {
		t.Fatal(err)
	}
	oldScope, err := AccountScope("codex", "gmail", "known@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AccountScope("codex", "gmail", "current-account-id"); err != nil {
		t.Fatal(err)
	}
	snap := model.UsageSnapshot{
		ProviderID:   "codex",
		ProfileID:    "gmail",
		Account:      "foreign@example.test",
		AccountScope: oldScope,
	}
	if snapshotBelongsToProfile(snap, "codex", "gmail", "current-account-id") {
		t.Fatal("Codex identity outside persisted history must be rejected")
	}
}

// agyAccountSwitchFixture prepares an isolated AGY profile authenticated as
// account A, seeds A's scoped cache with a distinct remaining percentage, and
// installs a fake `agy` CLI on PATH whose only job is to switch the profile
// identity to account B in the middle of the quota probe and then answer with
// B's quota evidence (77%). The switch happens inside the probe subprocess —
// no sleeps, no goroutines, fully deterministic.
func agyAccountSwitchFixture(t *testing.T) (scopeA model.AccountScope, home string) {
	t.Helper()
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("NEXUS_CODEX_APP_SERVER", "0")

	if _, err := Create("agy", "swap"); err != nil {
		t.Fatal(err)
	}
	home, err := Home("agy", "swap")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".gemini"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".gemini", "google_accounts.json"), []byte(`{"active":"a@example.test"}`), 0600); err != nil {
		t.Fatal(err)
	}
	// A valid, non-expired OAuth token is the only authentication evidence the
	// live probe accepts (google_accounts.json alone fails closed).
	token := fmt.Sprintf(`{"token":{"access_token":"at-test","refresh_token":"rt-test","expiry":%q}}`, time.Now().Add(2*time.Hour).UTC().Format(time.RFC3339Nano))
	if err := os.WriteFile(filepath.Join(home, ".gemini", "antigravity-oauth-token"), []byte(token), 0600); err != nil {
		t.Fatal(err)
	}

	scopeA, err = AccountScope("agy", "swap", "a@example.test")
	if err != nil {
		t.Fatal(err)
	}
	remaining := 42.0
	used := 58.0
	if err := quota.NewEngine(time.Minute).SaveUsageForScope(scopeA, model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "swap",
		Status:     model.UsageLive,
		Source:     model.SourceCLIOutput,
		Account:    "a@example.test",
		FetchedAt:  time.Now(),
		Windows: []model.UsageWindow{{
			Kind:             "5h",
			Group:            "gemini",
			RemainingPercent: &remaining,
			UsedPercent:      &used,
		}},
	}); err != nil {
		t.Fatal(err)
	}

	fakeBinDir := t.TempDir()
	_ = testutil.WriteSpecBinary(t, fakeBinDir, "agy")
	t.Setenv("FIXTURE_CLI_SPEC", fmt.Sprintf(
		`{"files":{%q:"{\"active\":\"b@example.test\"}",%q:"{\"account\":\"b@example.test\"}",%q:"{\"account\":\"b@example.test\"}"},"stdout":["gemini\t5h\t77%%\t2h"]}`,
		filepath.Join(home, ".gemini", "google_accounts.json"),
		filepath.Join(home, "usage.json"),
		filepath.Join(home, "..", "usage.json")))
	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return scopeA, home
}

// TestRefreshUsageSnapshotDiscardsMidFlightAccountSwitch covers the F2.8
// scenario (issue #228): a refresh started for account A observes a snapshot
// identified as account B (the Google account switched while the probe was in
// flight). The B response must be discarded by snapshotBelongsToProfile and
// must never be served or persisted as account A's current quota; A's own
// last-known evidence (42%) must survive the failed refresh.
func TestRefreshUsageSnapshotDiscardsMidFlightAccountSwitch(t *testing.T) {
	scopeA, home := agyAccountSwitchFixture(t)

	snap := RefreshUsageSnapshot("agy", "swap")

	// Sanity check: the fake probe really ran and really switched the identity
	// mid-flight (this is what makes the B-identified observation possible).
	switched, err := os.ReadFile(filepath.Join(home, ".gemini", "google_accounts.json"))
	if err != nil || !strings.Contains(string(switched), "b@example.test") {
		t.Fatalf("fake probe did not perform the mid-flight account switch: %q (%v)", switched, err)
	}

	// The discarded B response must not reach the caller as live data.
	if snap.Status == model.UsageLive || snap.Status == model.UsageCached {
		t.Fatalf("status=%s; the foreign B snapshot must be discarded, not served as current", snap.Status)
	}
	if snap.Account == "b@example.test" {
		t.Fatalf("account=%q; the B-identified response must not be attributed to this profile", snap.Account)
	}
	for _, w := range snap.Windows {
		if w.RemainingPercent != nil && *w.RemainingPercent == 77 {
			t.Fatalf("the discarded B quota evidence (77%%) leaked into the refresh response: %+v", snap.Windows)
		}
	}

	// The failed refresh falls back to A's preserved last-known observation.
	if snap.Status != model.UsageEstimated {
		t.Fatalf("status=%s, want ESTIMATED built from A's preserved last-known", snap.Status)
	}
	if snap.Account != "a@example.test" {
		t.Fatalf("account=%q, want a@example.test (A's last-known identity)", snap.Account)
	}
	if len(snap.Windows) != 1 || snap.Windows[0].RemainingPercent == nil || *snap.Windows[0].RemainingPercent != 42 {
		t.Fatalf("windows=%+v, want A's preserved 42%% observation", snap.Windows)
	}
	if snap.Error == "" {
		t.Fatal("expected a diagnostic explaining the degraded snapshot")
	}

	// The scoped cache for A still holds A's evidence — the B observation must
	// not have overwritten it.
	cached, found := quota.NewEngine(time.Minute).GetCachedUsageForScope(scopeA)
	if !found {
		t.Fatal("A's scoped cache vanished after the discarded mid-flight switch")
	}
	if cached.Account != "a@example.test" {
		t.Fatalf("cached account=%q, want a@example.test", cached.Account)
	}
	for _, w := range cached.Windows {
		if w.RemainingPercent != nil && *w.RemainingPercent == 77 {
			t.Fatalf("B's 77%% evidence was persisted into A's scoped cache: %+v", cached.Windows)
		}
	}

	// The on-disk cache file itself must never have carried B's evidence. The
	// adapter now refuses to persist snapshots whose identity is not bound to
	// the profile (F2.8, issue #228), so the transient window in which B's
	// quota sat under A's scope key before the ESTIMATED re-save is closed at
	// the source.
	cacheRoot, err := config.ProfileRoot("agy", "swap")
	if err != nil {
		t.Fatal(err)
	}
	rawCache, err := os.ReadFile(filepath.Join(cacheRoot, "usage.json"))
	if err != nil {
		t.Fatalf("A's cache file missing after the refresh: %v", err)
	}
	if strings.Contains(string(rawCache), "b@example.test") || strings.Contains(string(rawCache), "\"remaining_percent\": 77") {
		t.Fatalf("B's evidence found in A's on-disk cache file: %s", rawCache)
	}
	if !strings.Contains(string(rawCache), "a@example.test") {
		t.Fatalf("A's on-disk cache file should still hold A's evidence: %s", rawCache)
	}

	// B never owns persisted quota evidence under its own identity either.
	scopeB, err := AccountScope("agy", "swap", "b@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if _, foundB := quota.NewEngine(time.Minute).GetCachedUsageForScope(scopeB); foundB {
		t.Fatal("the discarded B snapshot must not be persisted under B's own identity")
	}

	// The identity guard that powers the discard rejects the exact B snapshot
	// the probe produced.
	foreign := model.UsageSnapshot{
		ProviderID:   "agy",
		ProfileID:    "swap",
		Account:      "b@example.test",
		Status:       model.UsageLive,
		Source:       model.SourceCLIOutput,
		FetchedAt:    time.Now(),
		AccountScope: scopeA,
	}
	if snapshotBelongsToProfile(foreign, "agy", "swap", "a@example.test") {
		t.Fatal("a snapshot identified as account B must be rejected for a profile resolved as A")
	}
}
