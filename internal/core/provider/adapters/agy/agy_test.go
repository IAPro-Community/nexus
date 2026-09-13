package agy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

func TestEmailFromIDTokenReadsURLSafeClaimsWithoutExposingToken(t *testing.T) {
	claims, err := json.Marshal(map[string]string{"email": "user@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	token := "header." + base64.RawURLEncoding.EncodeToString(claims) + ".signature"
	raw, err := json.Marshal(map[string]string{"id_token": token})
	if err != nil {
		t.Fatal(err)
	}
	if got := emailFromIDToken(raw); got != "user@example.com" {
		t.Fatalf("emailFromIDToken=%q, want user@example.com", got)
	}
	if got := emailFromIDToken([]byte(`{"id_token":"not-a-jwt"}`)); got != "" {
		t.Fatalf("malformed token must return empty email, got %q", got)
	}
}

func TestAGYSecretServiceIsOptIn(t *testing.T) {
	t.Setenv("NEXUS_AGY_ENABLE_SECRET_SERVICE", "")
	if agySecretServiceEnabled() {
		t.Fatal("Secret Service must be disabled by default to avoid recurring GNOME prompts")
	}
	t.Setenv("NEXUS_AGY_ENABLE_SECRET_SERVICE", "1")
	if !agySecretServiceEnabled() {
		t.Fatal("Secret Service opt-in was not honored")
	}
}

func TestLinkSharedAgyItemsDoesNotOverwriteProfileSettings(t *testing.T) {
	host := t.TempDir()
	profile := t.TempDir()
	if err := os.MkdirAll(filepath.Join(host, "antigravity-cli"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(profile, ".gemini", "antigravity-cli"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"settings.json": `{"model":"new-model"}`,
		"history.jsonl": "history\n",
	} {
		if err := os.WriteFile(filepath.Join(host, "antigravity-cli", name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	linkSharedAgyItems(profile, host)

	if _, err := os.Stat(filepath.Join(profile, ".gemini", "antigravity-cli", "settings.json")); !os.IsNotExist(err) {
		t.Fatalf("profile settings must remain profile-owned, stat error=%v", err)
	}
	if _, err := os.Stat(filepath.Join(profile, ".gemini", "antigravity-cli", "history.jsonl")); err != nil {
		t.Fatalf("history should still be shared: %v", err)
	}
}

// --- InspectAuth: expired token detection ---

func TestInspectAuthExpiredTokenWithoutRefreshToken(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	profileDir := filepath.Join(dataDir, "profiles", "agy", "expired-no-refresh")
	homeDir := filepath.Join(profileDir, "home")
	geminiDir := filepath.Join(homeDir, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(geminiDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Token expired 1 hour ago, no refresh_token.
	expiredTime := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339Nano)
	tokenContent := `{"token":{"access_token":"ya29.expired","refresh_token":"","expiry":"` + expiredTime + `"}}`
	if err := os.WriteFile(filepath.Join(geminiDir, "antigravity-oauth-token"), []byte(tokenContent), 0600); err != nil {
		t.Fatal(err)
	}

	info := New().InspectAuth(context.Background(), model.Profile{Provider: "agy", Name: "expired-no-refresh"})
	if info.Authenticated {
		t.Fatal("expired token without refresh_token must NOT be authenticated")
	}
	if info.Status != "Token expired" {
		t.Fatalf("status=%q want 'Token expired'", info.Status)
	}
}

func TestInspectAuthExpiredTokenWithRefreshToken(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	profileDir := filepath.Join(dataDir, "profiles", "agy", "expired-with-refresh")
	homeDir := filepath.Join(profileDir, "home")
	geminiDir := filepath.Join(homeDir, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(geminiDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Token expired 1 hour ago, HAS refresh_token.
	expiredTime := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339Nano)
	tokenContent := `{"token":{"access_token":"ya29.expired","refresh_token":"1//0hRefreshToken","expiry":"` + expiredTime + `"}}`
	if err := os.WriteFile(filepath.Join(geminiDir, "antigravity-oauth-token"), []byte(tokenContent), 0600); err != nil {
		t.Fatal(err)
	}

	info := New().InspectAuth(context.Background(), model.Profile{Provider: "agy", Name: "expired-with-refresh"})
	if !info.Authenticated {
		t.Fatal("expired token with usable refresh_token must remain authenticated so the monitor can refresh")
	}
	if info.Health != model.HealthDegraded {
		t.Fatalf("health=%q want DEGRADED", info.Health)
	}
	if info.Status != "Token expired (refreshable)" {
		t.Fatalf("status=%q want 'Token expired (refreshable)'", info.Status)
	}
}

func TestInspectAuthExpiredTokenResolvesEmailFromGoogleAccounts(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	profileDir := filepath.Join(dataDir, "profiles", "agy", "expired-with-email")
	homeDir := filepath.Join(profileDir, "home")
	geminiDir := filepath.Join(homeDir, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(geminiDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Expired token.
	expiredTime := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339Nano)
	tokenContent := `{"token":{"access_token":"ya29.expired","refresh_token":"1//0hRefreshToken","expiry":"` + expiredTime + `"}}`
	if err := os.WriteFile(filepath.Join(geminiDir, "antigravity-oauth-token"), []byte(tokenContent), 0600); err != nil {
		t.Fatal(err)
	}

	// google_accounts.json with active email.
	accountsDir := filepath.Join(homeDir, ".gemini")
	if err := os.MkdirAll(accountsDir, 0700); err != nil {
		t.Fatal(err)
	}
	accountsContent := `{"active": "user@gmail.com"}`
	if err := os.WriteFile(filepath.Join(accountsDir, "google_accounts.json"), []byte(accountsContent), 0600); err != nil {
		t.Fatal(err)
	}

	info := New().InspectAuth(context.Background(), model.Profile{Provider: "agy", Name: "expired-with-email"})
	if !info.Authenticated {
		t.Fatal("expired token with refresh_token must remain authenticated (refreshable)")
	}
	if info.Email != "user@gmail.com" {
		t.Fatalf("email=%q want 'user@gmail.com' (must resolve even when expired)", info.Email)
	}
}

func TestInspectAuthValidTokenWithRefreshToken(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	profileDir := filepath.Join(dataDir, "profiles", "agy", "valid-with-refresh")
	homeDir := filepath.Join(profileDir, "home")
	geminiDir := filepath.Join(homeDir, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(geminiDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Token expires in 1 hour, HAS refresh_token.
	futureTime := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339Nano)
	tokenContent := `{"token":{"access_token":"ya29.valid","refresh_token":"1//0hRefreshToken","expiry":"` + futureTime + `"}}`
	if err := os.WriteFile(filepath.Join(geminiDir, "antigravity-oauth-token"), []byte(tokenContent), 0600); err != nil {
		t.Fatal(err)
	}

	info := New().InspectAuth(context.Background(), model.Profile{Provider: "agy", Name: "valid-with-refresh"})
	if !info.Authenticated {
		t.Fatal("valid token with refresh_token must be authenticated")
	}
	if info.Status != "Authenticated" {
		t.Fatalf("status=%q want 'Authenticated'", info.Status)
	}
}

func TestFetchLiveQuotaRefusesExpiredAccessToken(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	// Force refresh failure: no usable oauth client credentials in this test.
	t.Setenv("NEXUS_AGY_OAUTH_CLIENT_ID", "")
	t.Setenv("NEXUS_AGY_OAUTH_CLIENT_SECRET", "")
	prevLook := agyOAuthLookPath
	agyOAuthLookPath = func(string) (string, error) { return "", fmt.Errorf("agy binary not found in test") }
	t.Cleanup(func() { agyOAuthLookPath = prevLook })

	home := filepath.Join(dataDir, "profiles", "agy", "expired-probe", "home")
	tokenDir := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(tokenDir, 0700); err != nil {
		t.Fatal(err)
	}
	expiredTime := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	tokenContent := `{"token":{"access_token":"ya29.expired","refresh_token":"1//0hRefresh","expiry":"` + expiredTime + `"}}`
	if err := os.WriteFile(filepath.Join(tokenDir, "antigravity-oauth-token"), []byte(tokenContent), 0600); err != nil {
		t.Fatal(err)
	}

	// When refresh cannot renew the access_token, the live probe must refuse to spawn.
	if _, ok := New().fetchLiveQuota(context.Background(), model.Profile{Provider: "agy", Name: "expired-probe"}); ok {
		t.Fatal("fetchLiveQuota must refuse to spawn AGY when the access token remains expired")
	}
}

func TestInspectAuthNoTokenFile(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	// Create empty profile dir with no token files.
	profileDir := filepath.Join(dataDir, "profiles", "agy", "no-token")
	if err := os.MkdirAll(filepath.Join(profileDir, "home", ".gemini"), 0700); err != nil {
		t.Fatal(err)
	}

	info := New().InspectAuth(context.Background(), model.Profile{Provider: "agy", Name: "no-token"})
	if info.Authenticated {
		t.Fatal("no token file must NOT be authenticated")
	}
}

func TestInspectAuthLoginKeyringAloneIsNotAuthenticated(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	profileDir := filepath.Join(dataDir, "profiles", "agy", "keyring-only")
	keyringsDir := filepath.Join(profileDir, "home", ".local", "share", "keyrings")
	if err := os.MkdirAll(keyringsDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(keyringsDir, "login.keyring"), []byte("not-agy-oauth"), 0600); err != nil {
		t.Fatal(err)
	}

	info := New().InspectAuth(context.Background(), model.Profile{Provider: "agy", Name: "keyring-only"})
	if info.Authenticated {
		t.Fatal("desktop login.keyring alone must NOT count as AGY authentication")
	}
}
