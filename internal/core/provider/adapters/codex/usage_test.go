package codex

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func writeAuth(t *testing.T, path, accountID, email string) {
	t.Helper()
	claims := fmt.Sprintf(`{"email":%q,"https://api.openai.com/auth":{"chatgpt_account_id":%q,"chatgpt_plan_type":"plus"}}`, email, accountID)
	payload := base64.RawURLEncoding.EncodeToString([]byte(claims))
	content := fmt.Sprintf(`{"tokens":{"id_token":"hdr.%s.sig","account_id":%q}}`, payload, accountID)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func writeRollout(t *testing.T, path string, used5h, usedWeekly float64, resetsAt int64, timestamp time.Time, bodyExtra string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	ts := timestamp.UTC().Format(time.RFC3339Nano)
	content := fmt.Sprintf(`{"timestamp":%q,"type":"session_meta","payload":{"session_id":"sess-1"}}
%s{"timestamp":%q,"type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":%v,"window_minutes":300,"resets_at":%d},"secondary":{"used_percent":%v,"window_minutes":10080,"resets_at":%d}}}}
`, ts, bodyExtra, ts, used5h, resetsAt, usedWeekly, resetsAt+86400)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(path, timestamp, timestamp)
}

func seedHostAuthObservation(t *testing.T, dataDir, accountID string, at time.Time) {
	t.Helper()
	path := filepath.Join(dataDir, "codex-host-auth-history.jsonl")
	rec := hostAuthObservation{ObservedAt: at.UTC(), AccountID: accountID}
	raw, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestCodexAdapterGetUsageFromIsolatedRollout(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	adapter := New()
	p := model.Profile{Provider: "codex", Name: "testprof"}
	homeDir := filepath.Join(dataDir, "profiles", "codex", "testprof", "home")
	sessDir := filepath.Join(homeDir, "sessions", "2026", "09", "11")
	writeAuth(t, filepath.Join(homeDir, "auth.json"), "acct-omega", "test@omega.com")
	now := time.Now()
	writeRollout(t, filepath.Join(sessDir, "rollout-2026-09-11T19-50-06-sess1.jsonl"), 15, 42, now.Add(2*time.Hour).Unix(), now, "")

	snap := adapter.GetUsage(context.Background(), p)
	if snap.Status != model.UsageLive {
		t.Fatalf("expected status LIVE, got %s err=%s", snap.Status, snap.Error)
	}
	if len(snap.Windows) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(snap.Windows))
	}
	if snap.Windows[0].Kind != "5h" || snap.Windows[1].Kind != "weekly" {
		t.Fatalf("kinds=%q/%q", snap.Windows[0].Kind, snap.Windows[1].Kind)
	}
	if *snap.Windows[0].RemainingPercent != 85 || *snap.Windows[1].RemainingPercent != 58 {
		t.Fatalf("remaining=%v/%v", *snap.Windows[0].RemainingPercent, *snap.Windows[1].RemainingPercent)
	}
	if snap.FetchedAt.IsZero() || time.Since(snap.FetchedAt) > time.Minute {
		t.Fatalf("FetchedAt should come from event timestamp, got %v", snap.FetchedAt)
	}
}

func TestCodexEmailInRolloutTextDoesNotStealAttribution(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	hostHome := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("HOME", hostHome)
	t.Setenv("AI_REAL_HOME", hostHome)

	omegaID, gmailID := "acct-omega", "acct-gmail"
	omegaHome := filepath.Join(dataDir, "profiles", "codex", "omega", "home")
	gmailHome := filepath.Join(dataDir, "profiles", "codex", "gmail", "home")
	writeAuth(t, filepath.Join(omegaHome, "auth.json"), omegaID, "kiver@omegasistemas.net.br")
	writeAuth(t, filepath.Join(gmailHome, "auth.json"), gmailID, "kiver.omegaedu@gmail.com")
	// Host auth is omega — gmail must NOT get host rollouts even if text cites gmail email.
	writeAuth(t, filepath.Join(hostHome, ".codex", "auth.json"), omegaID, "kiver@omegasistemas.net.br")
	// Establish host ownership before any rollout so attribution is prospective.
	seedHostAuthObservation(t, dataDir, omegaID, time.Now().Add(-time.Minute))

	hostSessions := filepath.Join(hostHome, ".codex", "sessions", "2026", "09", "11")
	now := time.Now()
	extra := `{"timestamp":"` + now.UTC().Format(time.RFC3339Nano) + `","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"talk about kiver.omegaedu@gmail.com and kivervinicius@gmail.com"}]}}` + "\n"
	writeRollout(t, filepath.Join(hostSessions, "rollout-shared.jsonl"), 1, 41, now.Add(time.Hour).Unix(), now, extra)

	adapter := New()
	omega := adapter.GetUsage(context.Background(), model.Profile{Provider: "codex", Name: "omega"})
	if omega.Status != model.UsageLive {
		t.Fatalf("omega should own host rollouts, got %s", omega.Status)
	}
	gmail := adapter.GetUsage(context.Background(), model.Profile{Provider: "codex", Name: "gmail"})
	if gmail.Source == model.SourceObservation || gmail.Status == model.UsageLive {
		t.Fatalf("gmail must not inherit host rollout via incidental email text, got status=%s", gmail.Status)
	}
}

func TestCodexAttributionInvariantAtMostOneProfilePerRollout(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	hostHome := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("HOME", hostHome)
	t.Setenv("AI_REAL_HOME", hostHome)

	ids := []string{"a1", "a2", "a3"}
	names := []string{"p1", "p2", "p3"}
	for i, name := range names {
		home := filepath.Join(dataDir, "profiles", "codex", name, "home")
		writeAuth(t, filepath.Join(home, "auth.json"), ids[i], name+"@example.com")
	}
	writeAuth(t, filepath.Join(hostHome, ".codex", "auth.json"), ids[0], "p1@example.com")
	seedHostAuthObservation(t, dataDir, ids[0], time.Now().Add(-time.Minute))
	now := time.Now()
	writeRollout(t, filepath.Join(hostHome, ".codex", "sessions", "2026", "09", "11", "rollout-x.jsonl"), 10, 20, now.Add(time.Hour).Unix(), now, "")

	adapter := New()
	claimed := 0
	for _, name := range names {
		snap := adapter.GetUsage(context.Background(), model.Profile{Provider: "codex", Name: name})
		if snap.Status == model.UsageLive {
			claimed++
		}
	}
	if claimed != 1 {
		t.Fatalf("exactly one profile may claim a shared host rollout, got %d", claimed)
	}
}

func TestCodexIsolatedProfileDoesNotSeeOtherProfileSessions(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	hostHome := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("HOME", hostHome)
	t.Setenv("AI_REAL_HOME", hostHome)

	omegaHome := filepath.Join(dataDir, "profiles", "codex", "omega", "home")
	gmailHome := filepath.Join(dataDir, "profiles", "codex", "gmail", "home")
	writeAuth(t, filepath.Join(omegaHome, "auth.json"), "acct-omega", "omega@example.com")
	writeAuth(t, filepath.Join(gmailHome, "auth.json"), "acct-gmail", "gmail@example.com")
	writeAuth(t, filepath.Join(hostHome, ".codex", "auth.json"), "acct-omega", "omega@example.com")

	now := time.Now()
	writeRollout(t, filepath.Join(omegaHome, "sessions", "2026", "09", "11", "rollout-omega.jsonl"), 4, 57, now.Add(time.Hour).Unix(), now, "")
	writeRollout(t, filepath.Join(gmailHome, "sessions", "2026", "09", "11", "rollout-gmail.jsonl"), 80, 28, now.Add(time.Hour).Unix(), now, "")

	adapter := New()
	omega := adapter.GetUsage(context.Background(), model.Profile{Provider: "codex", Name: "omega"})
	gmail := adapter.GetUsage(context.Background(), model.Profile{Provider: "codex", Name: "gmail"})
	if *omega.Windows[0].RemainingPercent != 96 {
		t.Fatalf("omega 5h remaining=%v want 96", *omega.Windows[0].RemainingPercent)
	}
	if *gmail.Windows[0].RemainingPercent != 20 {
		t.Fatalf("gmail 5h remaining=%v want 20", *gmail.Windows[0].RemainingPercent)
	}
}

func TestCodexFetchedAtFromEventNotNow(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	home := filepath.Join(dataDir, "profiles", "codex", "old", "home")
	writeAuth(t, filepath.Join(home, "auth.json"), "acct-old", "old@example.com")
	eventTime := time.Now().Add(-2 * time.Hour)
	writeRollout(t, filepath.Join(home, "sessions", "rollout-old.jsonl"), 50, 50, time.Now().Add(time.Hour).Unix(), eventTime, "")

	snap := New().GetUsage(context.Background(), model.Profile{Provider: "codex", Name: "old"})
	if snap.Status != model.UsageEstimated {
		t.Fatalf("status=%s want ESTIMATED for event older than live TTL", snap.Status)
	}
	delta := snap.FetchedAt.Sub(eventTime).Abs()
	if delta > time.Minute {
		t.Fatalf("FetchedAt=%v want near %v", snap.FetchedAt, eventTime)
	}
}

func TestCodexPastResetsAtNotExhausted(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	home := filepath.Join(dataDir, "profiles", "codex", "past", "home")
	writeAuth(t, filepath.Join(home, "auth.json"), "acct-past", "past@example.com")
	now := time.Now()
	writeRollout(t, filepath.Join(home, "sessions", "rollout-past.jsonl"), 100, 50, now.Add(-time.Hour).Unix(), now, "")

	snap := New().GetUsage(context.Background(), model.Profile{Provider: "codex", Name: "past"})
	if len(snap.Windows) < 1 {
		t.Fatal("expected windows")
	}
	if snap.Windows[0].ResetDescription != "window rolled over" {
		t.Fatalf("reset desc=%q", snap.Windows[0].ResetDescription)
	}
	if snap.Windows[0].ResetTime == nil || !snap.Windows[0].ResetTime.Before(now) {
		t.Fatal("expected past ResetTime")
	}
}

func TestCodexWindowMinutesMapsKind(t *testing.T) {
	limits := rolloutRateLimits{
		Primary:   &rateLimitWindow{UsedPercent: 10, WindowMinutes: 10080, ResetsAt: time.Now().Add(time.Hour).Unix()},
		Secondary: &rateLimitWindow{UsedPercent: 20, WindowMinutes: 300, ResetsAt: time.Now().Add(time.Hour).Unix()},
	}
	windows := buildWindowsFromRateLimits(limits)
	if len(windows) != 2 {
		t.Fatalf("windows=%d", len(windows))
	}
	if windows[0].Kind != "weekly" || windows[1].Kind != "5h" {
		t.Fatalf("kinds=%s/%s (window_minutes must win over primary/secondary order)", windows[0].Kind, windows[1].Kind)
	}
}

func TestCodexSharedSymlinkMigratedAway(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	hostHome := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("HOME", hostHome)
	t.Setenv("AI_REAL_HOME", hostHome)

	hostSessions := filepath.Join(hostHome, ".codex", "sessions")
	if err := os.MkdirAll(hostSessions, 0700); err != nil {
		t.Fatal(err)
	}
	writeAuth(t, filepath.Join(hostHome, ".codex", "auth.json"), "acct-host", "host@example.com")

	home := filepath.Join(dataDir, "profiles", "codex", "mig", "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	writeAuth(t, filepath.Join(home, "auth.json"), "acct-host", "host@example.com")
	if err := os.Symlink(hostSessions, filepath.Join(home, "sessions")); err != nil {
		t.Fatal(err)
	}

	if err := New().Prepare(context.Background(), model.Profile{Provider: "codex", Name: "mig"}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(filepath.Join(home, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatal("sessions symlink to host must be replaced with a real directory")
	}
	if !fi.IsDir() {
		t.Fatal("sessions must be a directory")
	}
}

func TestCodexReverseReadStopsEarly(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	home := filepath.Join(dataDir, "profiles", "codex", "perf", "home")
	writeAuth(t, filepath.Join(home, "auth.json"), "acct-perf", "perf@example.com")
	sess := filepath.Join(home, "sessions")
	now := time.Now()
	// One large-ish rollout with padding lines before the final token_count.
	var b strings.Builder
	ts := now.UTC().Format(time.RFC3339Nano)
	b.WriteString(`{"timestamp":"` + ts + `","type":"session_meta","payload":{"session_id":"big"}}` + "\n")
	pad := strings.Repeat("x", 200)
	for i := 0; i < 2000; i++ {
		b.WriteString(`{"timestamp":"` + ts + `","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"` + pad + `"}]}}` + "\n")
	}
	fmt.Fprintf(&b, `{"timestamp":%q,"type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":3,"window_minutes":300,"resets_at":%d}}}}`+"\n", ts, now.Add(time.Hour).Unix())
	path := filepath.Join(sess, "rollout-big.jsonl")
	if err := os.MkdirAll(sess, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0600); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(path, now, now)

	snap := New().GetUsage(context.Background(), model.Profile{Provider: "codex", Name: "perf"})
	if snap.Status != model.UsageLive || len(snap.Windows) < 1 {
		t.Fatalf("expected LIVE from reverse read, got %s windows=%d", snap.Status, len(snap.Windows))
	}
	if *snap.Windows[0].RemainingPercent != 97 {
		t.Fatalf("remaining=%v", *snap.Windows[0].RemainingPercent)
	}
}

func TestFormatCodexResetTime(t *testing.T) {
	if got := formatCodexResetTime(0); got != "Quota available" {
		t.Fatalf("expected 'Quota available', got %s", got)
	}
	now := time.Now()
	sameDay := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
	if sameDay.Before(now) {
		sameDay = sameDay.Add(24 * time.Hour)
	}
	gotSameDay := formatCodexResetTime(sameDay.Unix())
	if !strings.HasPrefix(gotSameDay, "resets ") {
		t.Fatalf("unexpected same day format: %s", gotSameDay)
	}
	past := now.Add(-time.Hour)
	if got := formatCodexResetTime(past.Unix()); got != "window rolled over" {
		t.Fatalf("past reset=%q", got)
	}
}

func TestCodexHostSameAccountAdoptsWhenNoIsolatedRollout(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	hostHome := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("HOME", hostHome)
	t.Setenv("AI_REAL_HOME", hostHome)

	gmailID, omegaID := "acct-gmail", "acct-omega"
	gmailHome := filepath.Join(dataDir, "profiles", "codex", "kivergmail", "home")
	omegaHome := filepath.Join(dataDir, "profiles", "codex", "omega", "home")
	writeAuth(t, filepath.Join(gmailHome, "auth.json"), gmailID, "kiver.omegaedu@gmail.com")
	writeAuth(t, filepath.Join(omegaHome, "auth.json"), omegaID, "kiver@omegasistemas.net.br")
	writeAuth(t, filepath.Join(hostHome, ".codex", "auth.json"), gmailID, "kiver.omegaedu@gmail.com")

	now := time.Now()
	// Simulate ledger switch omega → gmail, then a fresh host rollout for gmail.
	ledger := filepath.Join(dataDir, "codex-host-auth-history.jsonl")
	_ = os.MkdirAll(filepath.Dir(ledger), 0700)
	omegaObs := now.Add(-2 * time.Hour).UTC().Format(time.RFC3339Nano)
	gmailObs := now.Add(-5 * time.Minute).UTC().Format(time.RFC3339Nano)
	content := fmt.Sprintf(`{"observed_at":%q,"account_id":%q}
{"observed_at":%q,"account_id":%q}
`, omegaObs, omegaID, gmailObs, gmailID)
	if err := os.WriteFile(ledger, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	resetWeekly := now.Add(72 * time.Hour).Unix()
	writeRollout(t, filepath.Join(hostHome, ".codex", "sessions", "2026", "09", "11",
		"rollout-2026-09-11T19-00-00-01a092b4-a0fd-7d40-84e2-8823d02ca804.jsonl"),
		100, 44, resetWeekly, now, "")

	adapter := New()
	gmail := adapter.GetUsage(context.Background(), model.Profile{Provider: "codex", Name: "kivergmail"})
	if gmail.Status != model.UsageLive && gmail.Status != model.UsageEstimated {
		t.Fatalf("gmail should ingest host same-account rollout, got %s err=%s", gmail.Status, gmail.Error)
	}
	if len(gmail.Windows) < 2 {
		t.Fatalf("expected 5h+weekly, got %#v", gmail.Windows)
	}
	if *gmail.Windows[0].RemainingPercent != 0 {
		t.Fatalf("5h remaining=%v want 0", *gmail.Windows[0].RemainingPercent)
	}
	if *gmail.Windows[1].RemainingPercent != 56 {
		t.Fatalf("weekly remaining=%v want 56", *gmail.Windows[1].RemainingPercent)
	}

	omega := adapter.GetUsage(context.Background(), model.Profile{Provider: "codex", Name: "omega"})
	if omega.Status == model.UsageLive || omega.Status == model.UsageEstimated {
		if len(omega.Windows) > 0 && omega.Windows[0].RemainingPercent != nil && *omega.Windows[0].RemainingPercent == 0 {
			t.Fatal("omega must not inherit gmail 0% 5h from host")
		}
	}
}

func TestCodexRolloutRankedByEventTimestampNotModTime(t *testing.T) {
	dataDir := codexTestEnv(t)
	home := filepath.Join(dataDir, "profiles", "codex", "ranked", "home")
	writeAuth(t, filepath.Join(home, "auth.json"), "acct-ranked", "ranked@example.com")
	sess := filepath.Join(home, "sessions")
	now := time.Now()

	// Newest mtime but an old rate_limits event.
	stale := filepath.Join(sess, "rollout-stale.jsonl")
	writeRollout(t, stale, 90, 90, now.Add(time.Hour).Unix(), now.Add(-40*time.Minute), "")
	_ = os.Chtimes(stale, now, now)

	// Older mtime but the most recent rate_limits event.
	fresh := filepath.Join(sess, "rollout-fresh.jsonl")
	writeRollout(t, fresh, 20, 30, now.Add(time.Hour).Unix(), now.Add(-1*time.Minute), "")
	_ = os.Chtimes(fresh, now.Add(-30*time.Minute), now.Add(-30*time.Minute))

	snap := New().GetUsage(context.Background(), model.Profile{Provider: "codex", Name: "ranked"})
	if *snap.Windows[0].RemainingPercent != 80 {
		t.Fatalf("5h remaining=%v want 80 from the newest event, not the newest file", *snap.Windows[0].RemainingPercent)
	}
	if snap.Status != model.UsageLive {
		t.Fatalf("status=%s want LIVE for a 1 minute old event", snap.Status)
	}
}

func TestCodexTolerantToPartiallyWrittenLastLine(t *testing.T) {
	dataDir := codexTestEnv(t)
	home := filepath.Join(dataDir, "profiles", "codex", "partial", "home")
	writeAuth(t, filepath.Join(home, "auth.json"), "acct-partial", "partial@example.com")
	path := filepath.Join(home, "sessions", "rollout-partial.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	ts := now.UTC().Format(time.RFC3339Nano)
	// A complete event followed by a half-flushed line, as seen right after a
	// session ends while Codex is still appending.
	body := fmt.Sprintf(`{"timestamp":%q,"type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":12,"window_minutes":300,"resets_at":%d}}}}
{"timestamp":%q,"type":"event_msg","payload":{"type":"token_c`, ts, now.Add(time.Hour).Unix(), ts)
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(path, now, now)

	snap := New().GetUsage(context.Background(), model.Profile{Provider: "codex", Name: "partial"})
	if snap.Status != model.UsageLive {
		t.Fatalf("status=%s want LIVE; a truncated trailing line must be skipped", snap.Status)
	}
	if *snap.Windows[0].RemainingPercent != 88 {
		t.Fatalf("remaining=%v want 88", *snap.Windows[0].RemainingPercent)
	}
}

func TestHostOwnedAtFailsClosedBeforeFirstObservation(t *testing.T) {
	dataDir := codexTestEnv(t)
	path := filepath.Join(dataDir, "codex-host-auth-history.jsonl")
	now := time.Now().UTC()
	entry := hostAuthObservation{ObservedAt: now, AccountID: "acct-current"}
	raw, _ := json.Marshal(entry)
	if err := os.WriteFile(path, append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if hostOwnedAt("acct-current", now.Add(-time.Hour), "acct-current") {
		t.Fatal("rollouts before the first ledger observation must not be attributed")
	}
	if !hostOwnedAt("acct-current", now.Add(time.Minute), "acct-current") {
		t.Fatal("rollouts after the observation should belong to the recorded owner")
	}
}

func TestHostCurrentOwnerAllowsFailsClosedWithoutLedgerEvidence(t *testing.T) {
	dataDir := codexTestEnv(t)
	// No ledger file at all: being the current host login is not evidence.
	if _, err := os.Stat(filepath.Join(dataDir, "codex-host-auth-history.jsonl")); err == nil {
		t.Fatal("expected no ledger for this test")
	}
	if hostCurrentOwnerAllows("acct-x", time.Now().Add(-time.Hour), "acct-x") {
		t.Fatal("ownership must fail closed when the ledger has no observation")
	}
}

func TestCodexRateLimitsWithoutTokenCountType(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	home := filepath.Join(dataDir, "profiles", "codex", "splash", "home")
	writeAuth(t, filepath.Join(home, "auth.json"), "acct-splash", "splash@example.com")
	now := time.Now()
	ts := now.UTC().Format(time.RFC3339Nano)
	path := filepath.Join(home, "sessions", "rollout-splash.jsonl")
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	// Splash-style event: rate_limits present without type=token_count.
	body := fmt.Sprintf(`{"timestamp":%q,"type":"event_msg","payload":{"type":"status_dump","rate_limits":{"primary":{"used_percent":100,"window_minutes":300,"resets_at":%d},"secondary":{"used_percent":44,"window_minutes":10080,"resets_at":%d}}}}
`, ts, now.Add(time.Hour).Unix(), now.Add(72*time.Hour).Unix())
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(path, now, now)

	snap := New().GetUsage(context.Background(), model.Profile{Provider: "codex", Name: "splash"})
	if snap.Status != model.UsageLive {
		t.Fatalf("status=%s want LIVE from non-token_count rate_limits", snap.Status)
	}
	if *snap.Windows[0].RemainingPercent != 0 || *snap.Windows[1].RemainingPercent != 56 {
		t.Fatalf("remaining=%v/%v", *snap.Windows[0].RemainingPercent, *snap.Windows[1].RemainingPercent)
	}
}

func TestCodexZeroUsedMeansHundredPercentRemaining(t *testing.T) {
	limits := rolloutRateLimits{
		Primary: &rateLimitWindow{UsedPercent: 0, WindowMinutes: 300, ResetsAt: time.Now().Add(time.Hour).Unix()},
	}
	windows := buildWindowsFromRateLimits(limits)
	if len(windows) != 1 || windows[0].RemainingPercent == nil || windows[0].UsedPercent == nil {
		t.Fatalf("expected one fully measured Codex window, got %#v", windows)
	}
	if *windows[0].RemainingPercent != 100 || *windows[0].UsedPercent != 0 {
		t.Fatalf("Codex used=0 must normalize to remaining=100 used=0, got remaining=%v used=%v", *windows[0].RemainingPercent, *windows[0].UsedPercent)
	}
}
