package codex

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/authcopy"
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

func TestImportHostAuthCopiesOnlyValidatedCredentialWithoutReplacingIt(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	hostHome := t.TempDir()
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)
	if err := os.MkdirAll(filepath.Join(hostHome, ".codex"), 0o700); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(map[string]any{
		"email":                       "codex-host@example.test",
		"https://api.openai.com/auth": map[string]string{"chatgpt_account_id": "acct-host"},
	})
	if err != nil {
		t.Fatal(err)
	}
	jwt := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`)) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".sig"
	hostAuth, err := json.Marshal(map[string]any{"tokens": map[string]string{
		"id_token": jwt, "access_token": "access", "refresh_token": "refresh",
	}})
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(hostHome, ".codex", "auth.json")
	if err := os.WriteFile(source, hostAuth, 0o600); err != nil {
		t.Fatal(err)
	}

	profile := model.Profile{Provider: "codex", Name: "imported"}
	adapter := New()
	copied, err := adapter.ImportHostAuth(t.Context(), profile)
	if err != nil || !copied {
		t.Fatalf("import = %v, %v; want true, nil", copied, err)
	}
	if !adapter.InspectAuth(t.Context(), profile).Authenticated {
		t.Fatal("imported Codex profile was not authenticated")
	}
	profileHome, err := config.ProfileHome("codex", profile.Name)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(profileHome, "auth.json")
	if got, err := os.ReadFile(destination); err != nil || string(got) != string(hostAuth) {
		t.Fatalf("imported auth differs from source: err=%v", err)
	}
	if got, err := os.ReadFile(source); err != nil || string(got) != string(hostAuth) {
		t.Fatalf("host auth changed during import: err=%v", err)
	}
	if err := os.WriteFile(destination, []byte("keep-existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if copied, err := adapter.ImportHostAuth(t.Context(), profile); copied || !errors.Is(err, authcopy.ErrDestinationExists) {
		t.Fatalf("second import = %v, %v; want no overwrite", copied, err)
	}
	if got, err := os.ReadFile(destination); err != nil || string(got) != "keep-existing" {
		t.Fatalf("existing profile auth was replaced: %q, %v", got, err)
	}
}
