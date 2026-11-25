package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAccountScopeVersionsCredentialChanges(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	if _, err := Create("codex", "isolated"); err != nil {
		t.Fatal(err)
	}
	a, err := AccountScope("codex", "isolated", "a@example.test")
	if err != nil {
		t.Fatal(err)
	}
	b, err := AccountScope("codex", "isolated", "b@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if a.AccountID != b.AccountID || a.IdentityVersion == b.IdentityVersion {
		t.Fatalf("credential switch must preserve account record and version identity: a=%+v b=%+v", a, b)
	}
	if a.Key() == b.Key() {
		t.Fatal("credential switch must produce a distinct canonical key")
	}
}

func TestAccountScopeSkipsRewriteWhenUnchanged(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	if _, err := Create("codex", "ro"); err != nil {
		t.Fatal(err)
	}
	if _, err := AccountScope("codex", "ro", "user@example.test"); err != nil {
		t.Fatal(err)
	}
	scopePath := filepath.Join(dataDir, "profiles", "codex", "ro", "account-scope.json")
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(scopePath, past, past); err != nil {
		t.Fatal(err)
	}
	if _, err := AccountScope("codex", "ro", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := AccountScope("codex", "ro", "user@example.test"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(scopePath)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(past) {
		t.Fatalf("unchanged AccountScope modified %s (want unchanged %s)", info.ModTime(), past)
	}
	entries, err := os.ReadDir(filepath.Dir(scopePath))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "account-scope.tmp.") {
			t.Fatalf("unexpected temp file left behind: %s", entry.Name())
		}
	}
}

func TestListSkipsInvalidProfileDirectories(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", t.TempDir())
	if _, err := Create("codex", "valid"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"--bad", ".tmp"} {
		if err := os.MkdirAll(filepath.Join(dataDir, "profiles", "codex", bad), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	listed, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Name != "valid" {
		t.Fatalf("unexpected listed profiles: %+v", listed)
	}
}
