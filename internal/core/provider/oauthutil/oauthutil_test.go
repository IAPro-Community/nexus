package oauthutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMutexForSameKey(t *testing.T) {
	a := MutexFor("profile-a")
	b := MutexFor("profile-a")
	c := MutexFor("profile-b")
	if a != b {
		t.Fatal("expected same mutex for same key")
	}
	if a == c {
		t.Fatal("expected different mutex for different key")
	}
}

func TestAtomicWriteAndRevokedSentinel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	if err := AtomicWriteFile(path, []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != `{"ok":true}` {
		t.Fatalf("read=%q err=%v", data, err)
	}
	if err := WriteRevokedSentinel(dir, "invalid_grant"); err != nil {
		t.Fatal(err)
	}
	ok, reason := IsRevoked(dir)
	if !ok || reason != "invalid_grant" {
		t.Fatalf("revoked=%v reason=%q", ok, reason)
	}
}

func TestKillSwitchAndHTTPClient(t *testing.T) {
	t.Setenv("NEXUS_TEST_KILL", "0")
	if !KillSwitch("NEXUS_TEST_KILL") {
		t.Fatal("expected kill switch")
	}
	t.Setenv("NEXUS_TEST_KILL", "1")
	if KillSwitch("NEXUS_TEST_KILL") {
		t.Fatal("expected kill switch off")
	}
	client := HTTPClient(2 * time.Second)
	if client == nil || client.Timeout != 2*time.Second {
		t.Fatalf("client=%v", client)
	}
}
