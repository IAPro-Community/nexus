package profile

import (
	"testing"
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
