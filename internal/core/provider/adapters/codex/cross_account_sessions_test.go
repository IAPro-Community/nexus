package codex

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestAdoptSessionFromSiblingProfile(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	host := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("NEXUS_CONFIG_DIR", cfg)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)
	t.Setenv("AI_CLI_CONFIG_DIR", cfg)
	t.Setenv("HOME", host)
	t.Setenv("AI_REAL_HOME", host)

	sessionID := "01a0sibling-1111-2222-3333-444444444444"
	omegaHome := filepath.Join(data, "profiles", "codex", "omega", "home")
	gmailHome := filepath.Join(data, "profiles", "codex", "gmail", "home")
	omegaSess := filepath.Join(omegaHome, "sessions", "2026", "09", "12")
	if err := os.MkdirAll(omegaSess, 0700); err != nil {
		t.Fatal(err)
	}
	writeAuth(t, filepath.Join(omegaHome, "auth.json"), "acct-omega", "omega@example.com")
	writeAuth(t, filepath.Join(gmailHome, "auth.json"), "acct-gmail", "gmail@example.com")
	now := time.Now().UTC()
	src := filepath.Join(omegaSess, "rollout-2026-09-12T10-00-00-"+sessionID+".jsonl")
	writeRollout(t, src, 10, 20, now.Add(time.Hour).Unix(), now, "")
	indexLine := `{"id":"` + sessionID + `","thread_name":"Sibling thread","updated_at":"` + now.Format(time.RFC3339Nano) + `"}` + "\n"
	if err := os.WriteFile(filepath.Join(omegaHome, "session_index.jsonl"), []byte(indexLine), 0600); err != nil {
		t.Fatal(err)
	}

	if err := adoptSessionIntoProfile("gmail", sessionID); err != nil {
		t.Fatalf("adoptSessionIntoProfile: %v", err)
	}
	got := findRolloutInTree(filepath.Join(gmailHome, "sessions"), sessionID)
	if got == "" {
		t.Fatal("expected sibling rollout to be adopted into gmail sessions")
	}
	idx, err := os.ReadFile(filepath.Join(gmailHome, "session_index.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(idx), sessionID) {
		t.Fatalf("session_index missing adopted id: %s", idx)
	}
	if !readCrossAccountSessionIDs(gmailHome)[sessionID] {
		t.Fatal("adopted sibling session must be marked cross-account")
	}
}

func TestPrepareSeedsSiblingSessionsForPicker(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	host := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("NEXUS_CONFIG_DIR", cfg)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)
	t.Setenv("AI_CLI_CONFIG_DIR", cfg)
	t.Setenv("HOME", host)
	t.Setenv("AI_REAL_HOME", host)

	sessionID := "01a0seeded-aaaa-bbbb-cccc-ddddeeeeffff"
	omegaHome := filepath.Join(data, "profiles", "codex", "omega", "home")
	gmailHome := filepath.Join(data, "profiles", "codex", "gmail", "home")
	omegaSess := filepath.Join(omegaHome, "sessions", "2026", "09", "12")
	if err := os.MkdirAll(omegaSess, 0700); err != nil {
		t.Fatal(err)
	}
	writeAuth(t, filepath.Join(omegaHome, "auth.json"), "acct-omega", "omega@example.com")
	writeAuth(t, filepath.Join(gmailHome, "auth.json"), "acct-gmail", "gmail@example.com")
	now := time.Now().UTC()
	writeRollout(t, filepath.Join(omegaSess, "rollout-2026-09-12T11-00-00-"+sessionID+".jsonl"), 5, 15, now.Add(time.Hour).Unix(), now, "")

	previousBudget := crossAccountSeedBudget
	crossAccountSeedBudget = 5 * time.Second
	t.Cleanup(func() { crossAccountSeedBudget = previousBudget })
	if err := New().Prepare(context.Background(), model.Profile{Provider: "codex", Name: "gmail"}); err != nil {
		t.Fatal(err)
	}
	if findRolloutInTree(filepath.Join(gmailHome, "sessions"), sessionID) == "" {
		t.Fatal("Prepare must seed sibling rollouts into destination sessions")
	}
	idx, err := os.ReadFile(filepath.Join(gmailHome, "session_index.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(idx), sessionID) {
		t.Fatalf("Prepare must merge/synthesize session_index for picker: %s", idx)
	}
}

func TestPrepareWithManyRolloutsStaysUnderBudget(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	host := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("NEXUS_CONFIG_DIR", cfg)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)
	t.Setenv("AI_CLI_CONFIG_DIR", cfg)
	t.Setenv("HOME", host)
	t.Setenv("AI_REAL_HOME", host)

	omegaHome := filepath.Join(data, "profiles", "codex", "omega", "home")
	gmailHome := filepath.Join(data, "profiles", "codex", "gmail", "home")
	omegaSess := filepath.Join(omegaHome, "sessions", "2026", "09", "12")
	if err := os.MkdirAll(omegaSess, 0700); err != nil {
		t.Fatal(err)
	}
	writeAuth(t, filepath.Join(omegaHome, "auth.json"), "acct-omega", "omega@example.com")
	writeAuth(t, filepath.Join(gmailHome, "auth.json"), "acct-gmail", "gmail@example.com")
	// Destination already has local sessions: Prepare must skip the sibling walk.
	localSess := filepath.Join(gmailHome, "sessions", "2026", "09", "12")
	if err := os.MkdirAll(localSess, 0700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	writeRollout(t, filepath.Join(localSess, "rollout-2026-09-12T10-00-00-01a0local-aaaa-bbbb-cccc-ddddeeeeffff.jsonl"), 1, 1, now.Add(time.Hour).Unix(), now, "")
	for i := 0; i < 400; i++ {
		id := fmt.Sprintf("01a0bulk-%04d-aaaa-bbbb-ccccddddeeee", i)
		writeRollout(t, filepath.Join(omegaSess, fmt.Sprintf("rollout-2026-09-12T11-%02d-00-%s.jsonl", i%60, id)), 5, 15, now.Add(time.Hour).Unix(), now, "")
	}

	start := time.Now()
	if err := New().Prepare(context.Background(), model.Profile{Provider: "codex", Name: "gmail"}); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("Prepare took %s with hundreds of sibling rollouts; launch seed must stay under 500ms", elapsed)
	}
}

func TestPrepareEmptyDestWithManyRolloutsRespectsBudget(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	host := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("NEXUS_CONFIG_DIR", cfg)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)
	t.Setenv("AI_CLI_CONFIG_DIR", cfg)
	t.Setenv("HOME", host)
	t.Setenv("AI_REAL_HOME", host)

	omegaHome := filepath.Join(data, "profiles", "codex", "omega", "home")
	gmailHome := filepath.Join(data, "profiles", "codex", "gmail", "home")
	omegaSess := filepath.Join(omegaHome, "sessions", "2026", "09", "12")
	if err := os.MkdirAll(omegaSess, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(gmailHome, "sessions"), 0700); err != nil {
		t.Fatal(err)
	}
	writeAuth(t, filepath.Join(omegaHome, "auth.json"), "acct-omega", "omega@example.com")
	writeAuth(t, filepath.Join(gmailHome, "auth.json"), "acct-gmail", "gmail@example.com")
	now := time.Now().UTC()
	for i := 0; i < 500; i++ {
		id := fmt.Sprintf("01a0empty-%04d-aaaa-bbbb-ccccddddeeee", i)
		writeRollout(t, filepath.Join(omegaSess, fmt.Sprintf("rollout-2026-09-12T12-%02d-00-%s.jsonl", i%60, id)), 5, 15, now.Add(time.Hour).Unix(), now, "")
	}

	start := time.Now()
	if err := New().Prepare(context.Background(), model.Profile{Provider: "codex", Name: "gmail"}); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)

	// Prepare applies a best-effort 200 ms work budget. Under parallel Windows
	// package tests, scheduler and filesystem latency can exceed a strict wall
	// clock assertion, so verify it does not import the whole large sibling set.
	imported := 0
	destRoot := filepath.Join(gmailHome, "sessions")
	if err := filepath.WalkDir(destRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasPrefix(entry.Name(), "rollout-") && strings.HasSuffix(entry.Name(), ".jsonl") {
			imported++
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if imported >= 500 {
		t.Fatalf("Prepare imported all 500 sibling rollouts instead of stopping at its work budget (elapsed %s)", elapsed)
	}
	t.Logf("Prepare imported %d of 500 sibling rollouts within its best-effort work budget (elapsed %s)", imported, elapsed)
}

func TestForeignAccountRolloutIgnoredForQuota(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	host := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("NEXUS_CONFIG_DIR", cfg)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfg)
	t.Setenv("AI_CLI_CONFIG_DIR", cfg)
	t.Setenv("HOME", host)
	t.Setenv("AI_REAL_HOME", host)
	t.Setenv("NEXUS_CODEX_APP_SERVER", "0")

	sessionID := "01a0foreign-zzzz-yyyy-xxxx-wwwwvvvvuuuu"
	gmailHome := filepath.Join(data, "profiles", "codex", "gmail", "home")
	sess := filepath.Join(gmailHome, "sessions")
	if err := os.MkdirAll(sess, 0700); err != nil {
		t.Fatal(err)
	}
	writeAuth(t, filepath.Join(gmailHome, "auth.json"), "acct-gmail", "gmail@example.com")
	now := time.Now().UTC()
	// High used_percent that must NOT become gmail quota.
	writeRollout(t, filepath.Join(sess, "rollout-2026-09-12T12-00-00-"+sessionID+".jsonl"), 95, 80, now.Add(time.Hour).Unix(), now, "")
	if err := recordCrossAccountSession(gmailHome, sessionID, "omega", "omega/sessions/x.jsonl"); err != nil {
		t.Fatal(err)
	}

	snap := New().GetUsage(context.Background(), model.Profile{Provider: "codex", Name: "gmail"})
	if len(snap.Windows) != 0 {
		t.Fatalf("cross-account resume rollout must not feed quota, got %+v", snap)
	}
}

func TestRolloutBelongsRejectsMismatchedAccountID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout-acct.jsonl")
	content := `{"type":"session_meta","payload":{"id":"sess","chatgpt_account_id":"acct-other"}}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if rolloutBelongsToProfile(path, time.Now(), false, dir, []string{dir}, "acct-self", "") {
		t.Fatal("rollout with foreign chatgpt_account_id must not belong to profile")
	}
}
