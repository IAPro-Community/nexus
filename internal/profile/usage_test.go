package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
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

	snap := GetUsageSnapshot("codex", "omega")
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

	snap := GetUsageSnapshot("codex", "ttl")
	if len(snap.Windows) != 1 {
		t.Fatalf("windows=%d want 1", len(snap.Windows))
	}
	// The rollout says 10% used, so 90% remains. A cached rollout-derived snapshot
	// must never hide fresher session evidence.
	if *snap.Windows[0].RemainingPercent != 90 {
		t.Fatalf("remaining=%v want the rollout 90", *snap.Windows[0].RemainingPercent)
	}
}

func TestGetUsageSnapshotRefetchesOfficialCodexReadingAfterTTL(t *testing.T) {
	codexCacheFixture(t, model.SourceOfficialAPI, 3*time.Minute)

	snap := GetUsageSnapshot("codex", "ttl")
	if len(snap.Windows) != 1 {
		t.Fatalf("windows=%d want 1", len(snap.Windows))
	}
	if *snap.Windows[0].RemainingPercent != 90 {
		t.Fatalf("remaining=%v want the rollout 90; an official read past the TTL must be re-read", *snap.Windows[0].RemainingPercent)
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

	snap := GetUsageSnapshot("codex", "aged")
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
