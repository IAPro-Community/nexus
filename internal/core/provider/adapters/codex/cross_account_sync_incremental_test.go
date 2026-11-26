package codex

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSeedCrossAccountSessionsImportsAndUpdatesWhenDestHasRollouts(t *testing.T) {
	dataDir := t.TempDir()
	hostHome := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv("NEXUS_REAL_HOME", hostHome)
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)

	// Account B already has an existing rollout in its sessions directory.
	bHome := filepath.Join(dataDir, "profiles", "codex", "acc-b", "home")
	bSession := filepath.Join(bHome, "sessions", "2026", "11", "25", "rollout-2026-11-25T09-00-00-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb.jsonl")
	if err := os.MkdirAll(filepath.Dir(bSession), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bSession, []byte(`{"type":"session_meta","payload":{"id":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","cwd":"/tmp"}}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bHome, "auth.json"), []byte(`{"tokens":{"account_id":"acct-b"}}`), 0600); err != nil {
		t.Fatal(err)
	}

	// Account A has a newer sibling rollout and its own auth.json.
	aHome := filepath.Join(dataDir, "profiles", "codex", "acc-a", "home")
	aSession := filepath.Join(aHome, "sessions", "2026", "11", "25", "rollout-2026-11-25T10-00-00-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.jsonl")
	if err := os.MkdirAll(filepath.Dir(aSession), 0700); err != nil {
		t.Fatal(err)
	}
	initialLine := `{"type":"session_meta","payload":{"id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","cwd":"/tmp"}}` + "\n"
	if err := os.WriteFile(aSession, []byte(initialLine), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aHome, "auth.json"), []byte(`{"tokens":{"account_id":"acct-a"}}`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := seedCrossAccountSessions("acc-b"); err != nil {
		t.Fatalf("seedCrossAccountSessions failed: %v", err)
	}

	syncedAtB := filepath.Join(bHome, "sessions", "2026", "11", "25", filepath.Base(aSession))
	got, err := os.ReadFile(syncedAtB)
	if err != nil {
		t.Fatalf("expected acc-b to receive acc-a rollout even though acc-b already had rollouts: %v", err)
	}
	if string(got) != initialLine {
		t.Fatalf("unexpected synced content: %q", string(got))
	}

	// Now grow acc-a's rollout with a second turn and re-seed acc-b.
	appended := initialLine + `{"type":"event_msg","payload":{"type":"user_message","message":"turn 2"}}` + "\n"
	if err := os.WriteFile(aSession, []byte(appended), 0600); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(aSession, future, future)

	if err := seedCrossAccountSessions("acc-b"); err != nil {
		t.Fatalf("second seedCrossAccountSessions failed: %v", err)
	}
	got2, err := os.ReadFile(syncedAtB)
	if err != nil {
		t.Fatal(err)
	}
	if string(got2) != appended {
		t.Fatalf("expected incremental rollout update in acc-b, got %q", string(got2))
	}

	// Ensure auth.json was never overwritten or copied from acc-a.
	authB, err := os.ReadFile(filepath.Join(bHome, "auth.json"))
	if err != nil || string(authB) != `{"tokens":{"account_id":"acct-b"}}` {
		t.Fatalf("acc-b auth.json modified: %q err=%v", string(authB), err)
	}
}
