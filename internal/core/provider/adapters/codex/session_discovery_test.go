package codex

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeDiscoveryRollout(t *testing.T, dir, name string, metaCWD string, modTime time.Time) {
	t.Helper()
	path := filepath.Join(dir, name)
	body := ""
	if metaCWD != "" {
		body = "{\"type\":\"session_meta\",\"payload\":{\"id\":\"\",\"cwd\":\"" + metaCWD + "\"}}\n"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
}

func TestFindActiveSessionID_AdoptsNewestRolloutAfterSince(t *testing.T) {
	home := t.TempDir()
	sessions := filepath.Join(home, "sessions", "2026", "11", "11")
	now := time.Now()

	writeDiscoveryRollout(t, sessions, "rollout-2026-11-11T09-00-00-11111111-2222-3333-4444-555555555555.jsonl", "", now.Add(-2*time.Hour))
	writeDiscoveryRollout(t, sessions, "rollout-2026-11-11T10-00-00-aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.jsonl", "", now.Add(-time.Minute))

	id, err := FindActiveSessionID(home, now.Add(-time.Hour), "")
	if err != nil {
		t.Fatalf("FindActiveSessionID: %v", err)
	}
	if id != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Fatalf("session id = %q, want newest rollout id", id)
	}
}

func TestFindActiveSessionID_IgnoresRolloutsBeforeSince(t *testing.T) {
	home := t.TempDir()
	sessions := filepath.Join(home, "sessions")
	writeDiscoveryRollout(t, sessions, "rollout-2026-11-11T10-00-00-aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.jsonl", "", time.Now().Add(-2*time.Hour))

	id, err := FindActiveSessionID(home, time.Now().Add(-time.Hour), "")
	if err != nil {
		t.Fatalf("FindActiveSessionID: %v", err)
	}
	if id != "" {
		t.Fatalf("session id = %q, want empty for pre-since rollouts", id)
	}
}

func TestFindActiveSessionID_NeverAdoptsCrossAccountImports(t *testing.T) {
	home := t.TempDir()
	sessions := filepath.Join(home, "sessions")
	writeDiscoveryRollout(t, sessions, "rollout-2026-11-11T10-00-00-aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.jsonl", "", time.Now().Add(-time.Minute))

	// Mark the rollout as a cross-account resume import: its session id is a
	// copy of another profile's thread and must never be adopted.
	marker := filepath.Join(home, crossAccountMarkerFile)
	record := "{\"session_id\":\"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee\",\"source_profile\":\"other\",\"adopted_at\":\"2026-11-11T10:00:00Z\"}\n"
	if err := os.WriteFile(marker, []byte(record), 0o600); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	id, err := FindActiveSessionID(home, time.Now().Add(-time.Hour), "")
	if err != nil {
		t.Fatalf("FindActiveSessionID: %v", err)
	}
	if id != "" {
		t.Fatalf("session id = %q, want empty: cross-account imports are not this session", id)
	}
}

func TestFindActiveSessionID_EmptyHomeIsError(t *testing.T) {
	if _, err := FindActiveSessionID("", time.Now(), ""); err == nil {
		t.Fatal("expected error for empty home")
	}
}

func TestFindActiveSessionID_NoRolloutsIsEmptyNotError(t *testing.T) {
	id, err := FindActiveSessionID(t.TempDir(), time.Now().Add(-time.Hour), "")
	if err != nil {
		t.Fatalf("FindActiveSessionID: %v", err)
	}
	if id != "" {
		t.Fatalf("session id = %q, want empty", id)
	}
}
