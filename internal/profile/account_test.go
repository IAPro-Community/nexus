package profile

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
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

func TestGetAccountInfoContextStopsSlowCursorAuthenticationProbe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Linux/macOS-specific")
	}

	dataDir := t.TempDir()
	configDir := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", configDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("NEXUS_CURSOR_AUTH_FIXTURE_BINARY", testBinary)
	readyFile := filepath.Join(t.TempDir(), "cursor-status-ready")
	t.Setenv("NEXUS_CURSOR_AUTH_FIXTURE_READY_FILE", readyFile)

	if _, err := Create("cursor", "slow-auth-context"); err != nil {
		t.Fatal(err)
	}
	fixture := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo fixture; exit 0; fi\nif [ \"$1\" = \"status\" ]; then if [ -f \"$NEXUS_CURSOR_AUTH_FIXTURE_READY_FILE\" ]; then echo 'authenticated'; exit 0; fi; NEXUS_CURSOR_AUTH_FIXTURE=1 exec \"$NEXUS_CURSOR_AUTH_FIXTURE_BINARY\" -test.run='^TestCursorAuthProbeFixture$'; fi\nexit 0\n"
	if err := os.WriteFile(filepath.Join(binDir, "cursor-agent"), []byte(fixture), 0700); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	started := time.Now()
	info, err := GetAccountInfoContext(ctx, "cursor", "slow-auth-context")
	elapsed := time.Since(started)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline error from auth probe, got %v (info: %+v)", err, info)
	}
	if elapsed > time.Second {
		t.Fatalf("auth probe ignored its context deadline for %v", elapsed)
	}
	if info.Authenticated || info.Health != model.HealthUnknown {
		t.Fatalf("timed-out probe must stay unverified, got %+v", info)
	}
	if strings.Contains(info.Status, "authenticated") {
		t.Fatalf("timed-out probe must not claim an authentication result: %+v", info)
	}
	if err := os.WriteFile(readyFile, []byte("ready"), 0600); err != nil {
		t.Fatal(err)
	}
	verified, err := GetAccountInfoContext(context.Background(), "cursor", "slow-auth-context")
	if err != nil {
		t.Fatalf("second auth probe should complete: %v", err)
	}
	if !verified.Authenticated {
		t.Fatalf("timed-out probe must not poison the auth cache: %+v", verified)
	}
}

func TestCursorAuthProbeFixture(t *testing.T) {
	if os.Getenv("NEXUS_CURSOR_AUTH_FIXTURE") != "1" {
		return
	}
	select {}
}
