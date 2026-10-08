package codex

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

// Regression test: a stale chatgpt_subscription_active_until inside the cached
// id_token must not mark the profile as logged out. The id_token is minted
// once at login while the refresh token keeps the session alive across billing
// renewals, so the CLI keeps working after the claim goes stale.
func TestInspectAuthStaleSubscriptionClaimStaysAuthenticated(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)

	home := filepath.Join(dataDir, "profiles", "codex", "stale-sub", "home")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(map[string]any{
		"email": "stale@example.com",
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id":                "acc-stale",
			"chatgpt_plan_type":                 "plus",
			"chatgpt_subscription_active_until": "2020-01-01T00:00:00+00:00",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	jwt := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`)) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".sig"
	authFile, err := json.Marshal(map[string]any{
		"tokens": map[string]any{
			"id_token":      jwt,
			"access_token":  "acc",
			"refresh_token": "ref",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "auth.json"), authFile, 0o600); err != nil {
		t.Fatal(err)
	}

	info := New().InspectAuth(t.Context(), model.Profile{Provider: "codex", Name: "stale-sub"})
	if !info.Authenticated {
		t.Fatalf("stale subscription claim must not force logout: %+v", info)
	}
	if info.Email != "stale@example.com" {
		t.Fatalf("expected email to be parsed, got: %+v", info)
	}
	if info.ExpiresAt.IsZero() {
		t.Fatalf("expected ExpiresAt to be recorded, got: %+v", info)
	}
}
