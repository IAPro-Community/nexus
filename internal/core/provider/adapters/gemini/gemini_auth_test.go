package gemini

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/oauthutil"
)

func TestValidHostGeminiAuthUsesOAuthTokensWithoutEmail(t *testing.T) {
	tests := []struct {
		name        string
		credentials map[string]any
		want        bool
	}{
		{
			name: "current Gemini CLI OAuth schema without email",
			credentials: map[string]any{
				"access_token": "access", "refresh_token": "refresh",
				"expiry_date": time.Now().Add(time.Hour).UnixMilli(),
				"id_token":    "synthetic-id-token", "scope": "openid email", "token_type": "Bearer",
			},
			want: true,
		},
		{
			name: "expired access token can refresh",
			credentials: map[string]any{
				"access_token": "expired-access", "refresh_token": "refresh",
				"expiry_date": time.Now().Add(-time.Hour).UnixMilli(),
			},
			want: true,
		},
		{
			name: "expired access token cannot refresh",
			credentials: map[string]any{
				"access_token": "expired-access",
				"expiry_date":  time.Now().Add(-time.Hour).UnixMilli(),
			},
			want: false,
		},
		{
			name:        "empty credential object",
			credentials: map[string]any{},
			want:        false,
		},
		{
			name: "malformed expiry",
			credentials: map[string]any{
				"access_token": "access", "expiry_date": "not-a-timestamp",
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(test.credentials)
			if err != nil {
				t.Fatal(err)
			}
			if got := validHostGeminiAuthAt(data, ""); got != test.want {
				t.Fatalf("validHostGeminiAuthAt() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestValidHostGeminiAuthRejectsRevokedRefreshToken(t *testing.T) {
	credentialsDir := t.TempDir()
	credentialsPath := filepath.Join(credentialsDir, "oauth_creds.json")
	credentials := map[string]any{
		"access_token": "access", "refresh_token": "revoked-refresh",
		"expiry_date": time.Now().Add(time.Hour).UnixMilli(),
	}
	data, err := json.Marshal(credentials)
	if err != nil {
		t.Fatal(err)
	}
	if err := oauthutil.WriteRevokedSentinel(credentialsDir, hashGeminiRefreshToken("revoked-refresh")); err != nil {
		t.Fatal(err)
	}
	if validHostGeminiAuthAt(data, credentialsPath) {
		t.Fatal("revoked Gemini refresh token was accepted")
	}
}

func TestGeminiInspectAuthAcceptsEmaillessSessionAndMarksExpiredTokenDegraded(t *testing.T) {
	home := setupGeminiProfile(t, "email-less")
	writeGeminiCredentials(t, home, map[string]any{
		"access_token": "expired-access", "refresh_token": "refresh",
		"expiry_date": time.Now().Add(-time.Hour).UnixMilli(),
	})

	info := New().InspectAuth(context.Background(), model.Profile{Name: "email-less"})
	if !info.Authenticated {
		t.Fatal("refreshable OAuth credentials without an email were not authenticated")
	}
	if info.Email != "" {
		t.Fatal("an email absent from OAuth claims should not be inferred")
	}
	if info.Health != model.HealthDegraded || info.Status != "Session needs refresh" {
		t.Fatalf("expired refreshable session health = %q, status = %q", info.Health, info.Status)
	}
}

func TestGeminiInspectAuthRejectsExpiredAccessTokenWithoutRefresh(t *testing.T) {
	home := setupGeminiProfile(t, "expired-unrefreshable")
	writeGeminiCredentials(t, home, map[string]any{
		"access_token": "expired-access",
		"expiry_date":  time.Now().Add(-time.Hour).UnixMilli(),
	})

	info := New().InspectAuth(context.Background(), model.Profile{Name: "expired-unrefreshable"})
	if info.Authenticated {
		t.Fatal("expired access token without a refresh token was authenticated")
	}
	if _, err := os.Stat(filepath.Join(home, ".gemini", "oauth_creds.json")); err != nil {
		t.Fatal("test credentials unexpectedly disappeared")
	}
}

func TestPrepareSelectsOAuthPersonalForImportedSession(t *testing.T) {
	home := setupGeminiProfile(t, "oauth-settings")
	writeGeminiCredentials(t, home, map[string]any{
		"access_token":  "access",
		"refresh_token": "refresh",
		"expiry_date":   time.Now().Add(time.Hour).UnixMilli(),
	})

	profile := model.Profile{Provider: "gemini", Name: "oauth-settings"}
	if err := New().Prepare(context.Background(), profile); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	settingsPath := filepath.Join(home, ".gemini", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(settings.json) error = %v", err)
	}
	var settings struct {
		Security struct {
			Auth struct {
				SelectedType string `json:"selectedType"`
			} `json:"auth"`
		} `json:"security"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Unmarshal(settings.json) error = %v", err)
	}
	if settings.Security.Auth.SelectedType != "oauth-personal" {
		t.Fatalf("selectedType = %q, want oauth-personal", settings.Security.Auth.SelectedType)
	}
}

func TestPreparePreservesExplicitGeminiAuthSettings(t *testing.T) {
	home := setupGeminiProfile(t, "existing-auth-settings")
	writeGeminiCredentials(t, home, map[string]any{
		"access_token":  "access",
		"refresh_token": "refresh",
		"expiry_date":   time.Now().Add(time.Hour).UnixMilli(),
	})
	settingsPath := filepath.Join(home, ".gemini", "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"security":{"auth":{"selectedType":"gemini-api-key"}},"theme":{"name":"dark"}}`), 0600); err != nil {
		t.Fatal(err)
	}

	profile := model.Profile{Provider: "gemini", Name: "existing-auth-settings"}
	if err := New().Prepare(context.Background(), profile); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]json.RawMessage
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	var security struct {
		Auth struct {
			SelectedType string `json:"selectedType"`
		} `json:"auth"`
	}
	if err := json.Unmarshal(settings["security"], &security); err != nil {
		t.Fatal(err)
	}
	if security.Auth.SelectedType != "gemini-api-key" {
		t.Fatalf("selectedType = %q, want existing gemini-api-key", security.Auth.SelectedType)
	}
	if string(settings["theme"]) != `{"name":"dark"}` {
		t.Fatalf("unrelated theme setting changed: %s", settings["theme"])
	}
}

func TestPrepareDoesNotSelectOAuthWithoutCredentials(t *testing.T) {
	home := setupGeminiProfile(t, "no-oauth-settings")
	profile := model.Profile{Provider: "gemini", Name: "no-oauth-settings"}
	if err := New().Prepare(context.Background(), profile); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".gemini", "settings.json")); !os.IsNotExist(err) {
		t.Fatalf("settings.json exists without OAuth credentials, stat error = %v", err)
	}
}

func TestPrepareHandlesNullGeminiSecuritySettings(t *testing.T) {
	home := setupGeminiProfile(t, "null-security-settings")
	writeGeminiCredentials(t, home, map[string]any{
		"access_token":  "access",
		"refresh_token": "refresh",
		"expiry_date":   time.Now().Add(time.Hour).UnixMilli(),
	})
	settingsPath := filepath.Join(home, ".gemini", "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"security":null}`), 0600); err != nil {
		t.Fatal(err)
	}

	profile := model.Profile{Provider: "gemini", Name: "null-security-settings"}
	if err := New().Prepare(context.Background(), profile); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var settings struct {
		Security struct {
			Auth struct {
				SelectedType string `json:"selectedType"`
			} `json:"auth"`
		} `json:"security"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	if settings.Security.Auth.SelectedType != "oauth-personal" {
		t.Fatalf("selectedType = %q, want oauth-personal", settings.Security.Auth.SelectedType)
	}
}
