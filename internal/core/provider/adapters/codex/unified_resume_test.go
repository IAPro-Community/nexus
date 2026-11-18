package codex

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUnifiedResumeDiscoversAccountsAndUnindexedLegacySessions(t *testing.T) {
	data, host, cfg := t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", data)
	t.Setenv("NEXUS_CONFIG_DIR", cfg)
	t.Setenv("NEXUS_REAL_HOME", host)
	t.Setenv("AI_REAL_HOME", host)
	profiles := filepath.Join(data, "profiles", "codex")
	for _, p := range []struct{ name, account, email string }{
		{"sistemas", "account-sistemas", "sistemas@example.test"},
		{"alias", "account-sistemas", "sistemas@example.test"},
		{"edu", "account-edu", "edu@example.test"},
	} {
		root := filepath.Join(profiles, p.name)
		if err := os.MkdirAll(root, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "profile.json"), []byte(fmt.Sprintf(`{"provider":"codex","name":%q}`, p.name)), 0600); err != nil {
			t.Fatal(err)
		}
		writeAuth(t, filepath.Join(root, "home", "auth.json"), p.account, p.email)
	}
	ids := []string{"aaaaaaaa-1111-4111-8111-111111111111", "bbbbbbbb-2222-4222-8222-222222222222", "cccccccc-3333-4333-8333-333333333333"}
	for i, item := range []struct{ home, cwd string }{
		{filepath.Join(profiles, "sistemas", "home"), "/projects/one"},
		{filepath.Join(profiles, "edu", "home"), "/projects/two"},
		{filepath.Join(host, ".codex-accounts", "old"), "/projects/legacy"},
	} {
		if i == 2 {
			writeAuth(t, filepath.Join(item.home, "auth.json"), "account-sistemas", "sistemas@example.test")
		}
		path := filepath.Join(item.home, "sessions", "2020", "01", "01", "rollout-2020-01-01T00-00-00-"+ids[i]+".jsonl")
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		body := fmt.Sprintf("{\"timestamp\":\"2020-01-01T00:00:00Z\",\"type\":\"session_meta\",\"payload\":{\"id\":%q,\"cwd\":%q}}\n", ids[i], item.cwd)
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, time.Unix(int64(100+i), 0), time.Unix(int64(100+i), 0)); err != nil {
			t.Fatal(err)
		}
	}
	index := fmt.Sprintf("{\"id\":%q,\"thread_name\":\"Plano de conta\",\"updated_at\":\"2025-01-01T00:00:00Z\"}\n", ids[0])
	if err := os.WriteFile(filepath.Join(profiles, "sistemas", "home", "session_index.jsonl"), []byte(index), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := ListUnifiedResumeSessions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d sessions: %+v", len(got), got)
	}
	byID := make(map[string]UnifiedResumeSession)
	for _, s := range got {
		byID[s.ID] = s
	}
	if s := byID[ids[0]]; s.Title != "Plano de conta" || s.Workspace != "/projects/one" || s.Account != "sistemas@example.test" || s.Profile != "sistemas" {
		t.Fatalf("indexed origin: %+v", s)
	}
	if s := byID[ids[1]]; s.Profile != "edu" || s.Workspace != "/projects/two" || s.UnavailableReason != "" {
		t.Fatalf("other account: %+v", s)
	}
	if s := byID[ids[2]]; s.Account != "sistemas@example.test" || s.Profile == "" || s.UnavailableReason != "" {
		t.Fatalf("legacy recovery: %+v", s)
	}
}

func TestUnifiedResumeReadsSQLiteTitlesWithoutJSONIndex(t *testing.T) {
	data, host := t.TempDir(), t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", data)
	t.Setenv("NEXUS_CONFIG_DIR", t.TempDir())
	t.Setenv("NEXUS_REAL_HOME", host)
	root := filepath.Join(data, "profiles", "codex", "work")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "profile.json"), []byte(`{"provider":"codex","name":"work"}`), 0600); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "home")
	writeAuth(t, filepath.Join(home, "auth.json"), "account-work", "work@example.test")
	id := "aaaaaaaa-1111-4111-8111-111111111111"
	writeDiscoveryRollout(t, filepath.Join(home, "sessions"), "rollout-2020-01-01T00-00-00-"+id+".jsonl", "/project", time.Now())
	db, err := sql.Open("sqlite", filepath.Join(home, "state_5.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE threads (id TEXT, title TEXT, cwd TEXT, updated_at INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO threads VALUES (?, ?, ?, ?)`, id, "Título pesquisável", "/project", 1700000000); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	list, err := ListUnifiedResumeSessions(context.Background())
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %+v %v", list, err)
	}
	if list[0].Title != "Título pesquisável" {
		t.Fatalf("sqlite title lost: %+v", list[0])
	}
}

func TestPrepareUnifiedResumeCopiesOnlySelectedSameAccountSession(t *testing.T) {
	data, host, cfg := t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", data)
	t.Setenv("NEXUS_CONFIG_DIR", cfg)
	t.Setenv("NEXUS_REAL_HOME", host)
	t.Setenv("AI_REAL_HOME", host)
	profileRoot := filepath.Join(data, "profiles", "codex", "active")
	if err := os.MkdirAll(profileRoot, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileRoot, "profile.json"), []byte(`{"provider":"codex","name":"active"}`), 0600); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(profileRoot, "home")
	legacy := filepath.Join(host, ".codex-accounts", "legacy")
	writeAuth(t, filepath.Join(active, "auth.json"), "account-same", "same@example.test")
	writeAuth(t, filepath.Join(legacy, "auth.json"), "account-same", "same@example.test")
	authBefore, err := os.ReadFile(filepath.Join(active, "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{"aaaaaaaa-1111-4111-8111-111111111111", "bbbbbbbb-2222-4222-8222-222222222222"}
	for _, id := range ids {
		path := filepath.Join(legacy, "sessions", "2020", "01", "01", "rollout-2020-01-01T00-00-00-"+id+".jsonl")
		writeDiscoveryRollout(t, filepath.Dir(path), filepath.Base(path), "/other-project", time.Now())
	}
	list, err := ListUnifiedResumeSessions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var selected UnifiedResumeSession
	for _, s := range list {
		if s.ID == ids[0] {
			selected = s
		}
	}
	if err := PrepareUnifiedResume(context.Background(), selected); err != nil {
		t.Fatal(err)
	}
	if findRolloutInTree(filepath.Join(active, "sessions"), ids[0]) == "" {
		t.Fatal("selected rollout was not recovered")
	}
	recovered := filepath.Join(active, "sessions", "2020", "01", "01", "rollout-2020-01-01T00-00-00-"+ids[0]+".jsonl")
	if _, err := os.Stat(recovered); err != nil {
		t.Fatalf("recovery must preserve the Codex date hierarchy: %v", err)
	}
	if findRolloutInTree(filepath.Join(active, "sessions"), ids[1]) != "" {
		t.Fatal("unselected rollout was copied")
	}
	authAfter, err := os.ReadFile(filepath.Join(active, "auth.json"))
	if err != nil || !bytes.Equal(authBefore, authAfter) {
		t.Fatal("credentials changed")
	}
	if findRolloutInTree(filepath.Join(legacy, "sessions"), ids[0]) == "" {
		t.Fatal("original was removed")
	}
}

func TestUnifiedResumePrefersOriginalAndRejectsAmbiguousOrigins(t *testing.T) {
	original := resumeCandidate{UnifiedResumeSession: UnifiedResumeSession{
		ID: "thread", SourcePath: "/original", Profile: "owner", AccountID: "account-one", Account: "one@example.test",
	}, active: true}
	imported := resumeCandidate{UnifiedResumeSession: UnifiedResumeSession{
		ID: "thread", SourcePath: "/imported", Profile: "other", AccountID: "account-two", Account: "two@example.test",
	}, imported: true, active: true}
	got := chooseResumeCandidate([]resumeCandidate{imported, original}, nil, "")
	if got.SourcePath != original.SourcePath || got.Profile != "owner" || got.UnavailableReason != "" {
		t.Fatalf("original not preferred: %+v", got)
	}
	imported.imported = false
	got = chooseResumeCandidate([]resumeCandidate{imported, original}, nil, "")
	if got.Account != "Desconhecida" || got.Profile != "" || got.UnavailableReason == "" {
		t.Fatalf("conflicting origins were attributed: %+v", got)
	}
	original.active = false
	got = chooseResumeCandidate([]resumeCandidate{original}, nil, "")
	if got.UnavailableReason == "" || got.Profile != "" {
		t.Fatalf("unavailable account was selectable: %+v", got)
	}
}

func TestPrepareUnifiedResumeRejectsChangedCredentials(t *testing.T) {
	data, host := t.TempDir(), t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", data)
	t.Setenv("NEXUS_CONFIG_DIR", t.TempDir())
	t.Setenv("NEXUS_REAL_HOME", host)
	root := filepath.Join(data, "profiles", "codex", "active")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "profile.json"), []byte(`{"provider":"codex","name":"active"}`), 0600); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "home")
	writeAuth(t, filepath.Join(home, "auth.json"), "account-one", "one@example.test")
	id := "aaaaaaaa-1111-4111-8111-111111111111"
	writeDiscoveryRollout(t, filepath.Join(home, "sessions"), "rollout-2020-01-01T00-00-00-"+id+".jsonl", "/project", time.Now())
	list, err := ListUnifiedResumeSessions(context.Background())
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %+v %v", list, err)
	}
	writeAuth(t, filepath.Join(home, "auth.json"), "account-two", "two@example.test")
	if err := PrepareUnifiedResume(context.Background(), list[0]); err == nil {
		t.Fatal("changed credentials must block resume")
	}
}
