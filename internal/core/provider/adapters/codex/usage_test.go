package codex

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

func TestCodexAdapterGetUsageFromRollout(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	adapter := New()
	p := model.Profile{
		Provider: "codex",
		Name:     "testprof",
	}

	// Prepare mock profile home with sessions directory
	homeDir := filepath.Join(dataDir, "profiles", "codex", "testprof", "home")
	sessDir := filepath.Join(homeDir, ".codex", "sessions", "2026", "09", "02")
	if err := os.MkdirAll(sessDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Create a dummy auth.json for testprof
	// base64url("{\"email\":\"test@omega.com\",\"https://api.openai.com/auth\":{\"chatgpt_plan_type\":\"plus\"}}") = eyJlbWFpbCI6InRlc3RAb21lZ2EuY29tIiwiaHR0cHM6Ly9hcGkub3BlbmFpLmNvbS9hdXRoIjp7ImNoYXRncHRfcGxhbl90eXBlIjoicGx1cyJ9fQ
	authContent := `{"tokens":{"id_token":"eyJhbGciOiJIUzI1NiJ9.eyJlbWFpbCI6InRlc3RAb21lZ2EuY29tIiwiaHR0cHM6Ly9hcGkub3BlbmFpLmNvbS9hdXRoIjp7ImNoYXRncHRfcGxhbl90eXBlIjoicGx1cyJ9fQ.signature"}}`
	if err := os.WriteFile(filepath.Join(homeDir, "auth.json"), []byte(authContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Write mock rollout jsonl with token_count and rate_limits, including email in developer message
	rolloutContent := `{"timestamp":"2026-09-02T19:50:06.861Z","ordinal":1,"type":"session_meta","payload":{"session_id":"sess-1"}}
{"timestamp":"2026-09-02T19:50:06.861Z","ordinal":2,"type":"event_msg","payload":{"type":"thread_settings_applied","thread_settings":{"model":"gpt-5.6-terra"}}}
{"timestamp":"2026-09-02T19:50:06.861Z","ordinal":3,"type":"response_item","payload":{"type":"message","role":"developer","content":[{"type":"input_text","text":"Authenticated as test@omega.com"}]}}
{"timestamp":"2026-09-02T19:50:06.861Z","ordinal":4,"type":"event_msg","payload":{"type":"token_count","info":null,"rate_limits":{"primary":{"used_percent":15.0,"window_minutes":300,"resets_at":1788385419},"secondary":{"used_percent":42.0,"window_minutes":10080,"resets_at":1788748323}}}}
`
	rolloutFile := filepath.Join(sessDir, "rollout-2026-09-02T19-50-06-sess1.jsonl")
	if err := os.WriteFile(rolloutFile, []byte(rolloutContent), 0600); err != nil {
		t.Fatal(err)
	}

	snap := adapter.GetUsage(context.Background(), p)
	if snap.Status != model.UsageLive {
		t.Fatalf("expected status LIVE, got %s", snap.Status)
	}
	if snap.Source != model.SourceObservation {
		t.Fatalf("expected source OBSERVATION, got %s", snap.Source)
	}
	if len(snap.Windows) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(snap.Windows))
	}

	// Primary: 15% used -> 85% remaining (same as Codex /status "left")
	if snap.Windows[0].RemainingPercent == nil || *snap.Windows[0].RemainingPercent != 85.0 {
		t.Fatalf("expected 85%% remaining, got %v", snap.Windows[0].RemainingPercent)
	}
	if snap.Windows[0].Group != "claude_gpt" {
		t.Fatalf("expected group claude_gpt, got %s", snap.Windows[0].Group)
	}

	// Secondary: 42% used -> 58% remaining
	if snap.Windows[1].RemainingPercent == nil || *snap.Windows[1].RemainingPercent != 58.0 {
		t.Fatalf("expected 58%% remaining, got %v", snap.Windows[1].RemainingPercent)
	}
	if snap.Windows[1].Group != "claude_gpt" {
		t.Fatalf("expected group claude_gpt, got %s", snap.Windows[1].Group)
	}
}

func TestCodexSharedHostRolloutsMatchHostAuthEmail(t *testing.T) {
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

	adapter := New()
	p := model.Profile{Provider: "codex", Name: "omega"}
	homeDir := filepath.Join(dataDir, "profiles", "codex", "omega", "home")
	if err := os.MkdirAll(homeDir, 0700); err != nil {
		t.Fatal(err)
	}
	auth := `{"tokens":{"id_token":"eyJhbGciOiJIUzI1NiJ9.eyJlbWFpbCI6ImtpdmVyQG9tZWdhc2lzdGVtYXMubmV0LmJyIn0.sig"}}`
	if err := os.WriteFile(filepath.Join(homeDir, "auth.json"), []byte(auth), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(hostHome, ".codex", "sessions", "2026", "09", "04"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hostHome, ".codex", "auth.json"), []byte(auth), 0600); err != nil {
		t.Fatal(err)
	}
	// No email inside the rollout — mirrors current Codex JSONL.
	rollout := `{"type":"session_meta","payload":{"session_id":"s1"}}
{"type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":1.0,"window_minutes":300,"resets_at":1788556157},"secondary":{"used_percent":41.0,"window_minutes":10080,"resets_at":1788748886}}}}
`
	path := filepath.Join(hostHome, ".codex", "sessions", "2026", "09", "04", "rollout-shared.jsonl")
	if err := os.WriteFile(path, []byte(rollout), 0600); err != nil {
		t.Fatal(err)
	}

	snap := adapter.GetUsage(context.Background(), p)
	if snap.Status != model.UsageLive {
		t.Fatalf("status=%s want LIVE", snap.Status)
	}
	if len(snap.Windows) != 2 {
		t.Fatalf("windows=%d want 2", len(snap.Windows))
	}
	if *snap.Windows[0].RemainingPercent != 99 || *snap.Windows[1].RemainingPercent != 59 {
		t.Fatalf("remaining=%v/%v want 99/59 (used 1/41)", *snap.Windows[0].RemainingPercent, *snap.Windows[1].RemainingPercent)
	}

	// Other profile must not claim the same shared host rollouts.
	otherHome := filepath.Join(dataDir, "profiles", "codex", "gmail", "home")
	if err := os.MkdirAll(otherHome, 0700); err != nil {
		t.Fatal(err)
	}
	otherAuth := `{"tokens":{"id_token":"eyJhbGciOiJIUzI1NiJ9.eyJlbWFpbCI6ImtpdmVyLm9tZWdhZWR1QGdtYWlsLmNvbSJ9.sig"}}`
	if err := os.WriteFile(filepath.Join(otherHome, "auth.json"), []byte(otherAuth), 0600); err != nil {
		t.Fatal(err)
	}
	other := adapter.GetUsage(context.Background(), model.Profile{Provider: "codex", Name: "gmail"})
	if other.Source == model.SourceObservation {
		t.Fatalf("gmail must not inherit host rollouts owned by omega auth")
	}
}

func TestFormatCodexResetTime(t *testing.T) {
	if got := formatCodexResetTime(0); got != "Quota available" {
		t.Fatalf("expected 'Quota available', got %s", got)
	}
	now := time.Now()
	// Test same day format
	sameDay := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	gotSameDay := formatCodexResetTime(sameDay.Unix())
	if gotSameDay != fmt.Sprintf("resets %s", sameDay.Format("15:04")) {
		t.Fatalf("unexpected same day format: %s", gotSameDay)
	}

	// Test cross day format
	crossDay := sameDay.Add(48 * time.Hour)
	gotCrossDay := formatCodexResetTime(crossDay.Unix())
	expectedCross := fmt.Sprintf("resets %s on %d %s", crossDay.Format("15:04"), crossDay.Day(), crossDay.Format("Jan"))
	if gotCrossDay != expectedCross {
		t.Fatalf("unexpected cross day format: got %s, want %s", gotCrossDay, expectedCross)
	}
}
