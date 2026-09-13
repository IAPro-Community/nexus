package claude

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

func setupClaudeProfile(t *testing.T, name string) string {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	home := filepath.Join(dataDir, "profiles", "claude", name, "home")
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0700); err != nil {
		t.Fatal(err)
	}
	return home
}

func TestInspectAuthTable(t *testing.T) {
	cases := []struct {
		name          string
		write         func(home string) error
		wantAuth      bool
		wantHealth    model.ProviderHealth
		wantStatusSub string
	}{
		{
			name: "no-credentials",
			write: func(home string) error {
				return nil
			},
			wantAuth:   false,
			wantHealth: model.HealthAuthRequired,
		},
		{
			name: "valid-flat",
			write: func(home string) error {
				future := time.Now().Add(time.Hour).UnixMilli()
				blob := map[string]any{
					"email": "user@example.com", "accessToken": "at", "refreshToken": "rt", "expiresAt": future,
				}
				raw, _ := json.Marshal(blob)
				return os.WriteFile(filepath.Join(home, ".claude", "credentials.json"), raw, 0600)
			},
			wantAuth:   true,
			wantHealth: model.HealthHealthy,
		},
		{
			name: "expired-with-refresh-nested",
			write: func(home string) error {
				past := time.Now().Add(-time.Hour).UnixMilli()
				blob := map[string]any{
					"claudeAiOauth": map[string]any{
						"accessToken": "at", "refreshToken": "rt", "expiresAt": past, "email": "nested@example.com",
					},
				}
				raw, _ := json.Marshal(blob)
				return os.WriteFile(filepath.Join(home, ".claude", ".credentials.json"), raw, 0600)
			},
			wantAuth:      true,
			wantHealth:    model.HealthDegraded,
			wantStatusSub: "refreshable",
		},
		{
			name: "expired-without-refresh",
			write: func(home string) error {
				past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
				blob := map[string]any{
					"email": "user@example.com", "accessToken": "at", "refreshToken": "", "expiresAt": past,
				}
				raw, _ := json.Marshal(blob)
				return os.WriteFile(filepath.Join(home, ".claude", "credentials.json"), raw, 0600)
			},
			wantAuth:   false,
			wantHealth: model.HealthAuthRequired,
		},
		{
			name: "no-expiry-fail-open",
			write: func(home string) error {
				blob := map[string]any{"email": "user@example.com", "accessToken": "at", "refreshToken": "rt"}
				raw, _ := json.Marshal(blob)
				return os.WriteFile(filepath.Join(home, ".claude", "credentials.json"), raw, 0600)
			},
			wantAuth:   true,
			wantHealth: model.HealthHealthy,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := setupClaudeProfile(t, tc.name)
			if err := tc.write(home); err != nil {
				t.Fatal(err)
			}
			info := New().InspectAuth(context.Background(), model.Profile{Provider: "claude", Name: tc.name})
			if info.Authenticated != tc.wantAuth {
				t.Fatalf("authenticated=%v want %v (status=%q)", info.Authenticated, tc.wantAuth, info.Status)
			}
			if info.Health != tc.wantHealth {
				t.Fatalf("health=%q want %q", info.Health, tc.wantHealth)
			}
			if tc.wantStatusSub != "" && !strings.Contains(strings.ToLower(info.Status), strings.ToLower(tc.wantStatusSub)) {
				t.Fatalf("status=%q want substring %q", info.Status, tc.wantStatusSub)
			}
		})
	}
}
