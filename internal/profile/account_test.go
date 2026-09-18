package profile

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetAccountInfo(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_CLI_CONFIG_DIR", cfg)

	// Create AGY profile — google_accounts.json alone is not auth evidence.
	if _, err := Create("agy", "google-acc"); err != nil {
		t.Fatal(err)
	}
	homeAgy, _ := Home("agy", "google-acc")
	tokenDir := filepath.Join(homeAgy, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(tokenDir, 0700); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(homeAgy, ".gemini", "google_accounts.json"), []byte(`{"active":"test@gmail.com"}`), 0600)
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	token := `{"token":{"access_token":"ya29.valid","refresh_token":"1//refresh","expiry":"` + future + `"}}`
	if err := os.WriteFile(filepath.Join(tokenDir, "antigravity-oauth-token"), []byte(token), 0600); err != nil {
		t.Fatal(err)
	}

	infoAgy := GetAccountInfo("agy", "google-acc")
	if infoAgy.Email != "test@gmail.com" || !infoAgy.Authenticated {
		t.Fatalf("unexpected AGY account info: %+v", infoAgy)
	}

	accountsOnly := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", accountsOnly)
	if _, err := Create("agy", "accounts-only"); err != nil {
		t.Fatal(err)
	}
	homeOnly, _ := Home("agy", "accounts-only")
	_ = os.MkdirAll(filepath.Join(homeOnly, ".gemini"), 0700)
	_ = os.WriteFile(filepath.Join(homeOnly, ".gemini", "google_accounts.json"), []byte(`{"active":"stale@gmail.com"}`), 0600)
	infoStale := GetAccountInfo("agy", "accounts-only")
	if infoStale.Authenticated {
		t.Fatalf("google_accounts.json alone must not authenticate: %+v", infoStale)
	}

	// Create Codex profile
	t.Setenv("AI_CLI_DATA_DIR", data)
	if _, err := Create("codex", "openai-acc"); err != nil {
		t.Fatal(err)
	}
	homeCodex, _ := Home("codex", "openai-acc")
	// JWT with email claim
	jwt := "eyJhbGciOiJub25lIn0.eyJlbWFpbCI6ImNvZGV4QGV4YW1wbGUuY29tIiwiaHR0cHM6Ly9hcGkub3BlbmFpLmNvbS9hdXRoIjp7ImNoYXRncHRfcGxhbl90eXBlIjoicHJvIn19.sig"
	_ = os.WriteFile(filepath.Join(homeCodex, "auth.json"), []byte(`{"auth_mode":"chatgpt","tokens":{"id_token":"`+jwt+`","access_token":"acc"}}`), 0600)

	infoCodex := GetAccountInfo("codex", "openai-acc")
	if infoCodex.Email != "codex@example.com" || infoCodex.Plan != "ChatGPT Pro" || !infoCodex.Authenticated {
		t.Fatalf("unexpected Codex account info: %+v", infoCodex)
	}
}
