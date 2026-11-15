package claude

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

// Claude Code now nests its identity state under .claude/.claude.json while
// legacy builds wrote $HOME/.claude.json. When only the nested file exists,
// InspectAuth must still resolve the account email: without it the profile
// usage gate attributes no snapshot and live quota evidence is discarded,
// which fail-closed the automatic launch for fully authenticated accounts.
func TestInspectAuthReadsNestedClaudeJSON(t *testing.T) {
	home := setupClaudeProfile(t, "work")
	credentials := map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken":      "tok",
			"refreshToken":     "ref",
			"subscriptionType": "pro",
			"expiresAt":        "2099-01-01T00:00:00Z",
		},
	}
	credBytes, err := json.Marshal(credentials)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude", ".credentials.json"), credBytes, 0600); err != nil {
		t.Fatal(err)
	}
	identity := `{"oauthAccount":{"emailAddress":"user@example.com","accountUuid":"u1"},"userID":"uid-1"}`
	if err := os.WriteFile(filepath.Join(home, ".claude", ".claude.json"), []byte(identity), 0600); err != nil {
		t.Fatal(err)
	}

	a := &Adapter{}
	info := a.InspectAuth(context.Background(), model.Profile{Provider: "claude", Name: "work"})
	if info.Email != "user@example.com" {
		t.Fatalf("email = %q, want nested .claude/.claude.json identity", info.Email)
	}
	if !info.Authenticated {
		t.Fatal("expected authenticated from nested credentials")
	}
}

func TestClaudeAccountEmailAcceptsEmailAddressField(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(home, ".claude", ".claude.json"),
		[]byte(`{"oauthAccount":{"emailAddress":"addr@example.com"}}`),
		0600,
	); err != nil {
		t.Fatal(err)
	}
	if got := claudeAccountEmail(home, ""); got != "addr@example.com" {
		t.Fatalf("claudeAccountEmail = %q, want emailAddress from nested file", got)
	}
}

func TestReadClaudeIdentityJSONPrefersNestedAndFallsBack(t *testing.T) {
	home := t.TempDir()
	if _, err := readClaudeIdentityJSON(home); err == nil {
		t.Fatal("expected missing identity to error")
	}
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"src":"legacy"}`), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := readClaudeIdentityJSON(home)
	if err != nil || !strings.Contains(string(data), "legacy") {
		t.Fatalf("legacy fallback failed: %v %s", err, data)
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude", ".claude.json"), []byte(`{"src":"nested"}`), 0600); err != nil {
		t.Fatal(err)
	}
	data, err = readClaudeIdentityJSON(home)
	if err != nil || !strings.Contains(string(data), "nested") {
		t.Fatalf("nested location should win: %v %s", err, data)
	}
}
