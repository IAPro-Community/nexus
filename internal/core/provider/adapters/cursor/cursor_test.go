package cursor

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
)

func TestInspectAuthMemoizesStatusProbe(t *testing.T) {
	t.Setenv("AI_CLI_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	invalidateCursorAuthCache("memo")

	var calls atomic.Int32
	prev := cursorStatusRun
	cursorStatusRun = func(ctx context.Context, bin string, args []string, env []string, cwd string) (string, error) {
		calls.Add(1)
		return "Logged in as user@example.com", nil
	}
	t.Cleanup(func() { cursorStatusRun = prev })

	// Force binary lookup to succeed without a real cursor-agent by putting a
	// stub on PATH via LookPath — InspectAuth still needs Prepare + binary.
	// When binary is missing, InspectAuth returns early without caching a live
	// probe. Stub LookPath indirectly by skipping Prepare failure: create home.
	a := New()
	p := model.Profile{Provider: "cursor", Name: "memo"}
	if err := a.Prepare(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	// Without a real binary, InspectAuth returns unauthenticated without calling
	// cursorStatusRun. Inject a fake binary via PATH is heavy; instead call the
	// cache path by writing a pre-warmed entry and verifying subsequent hits.
	info := model.AccountInfo{Authenticated: true, Status: "Authenticated", Health: model.HealthHealthy}
	cursorAuthCacheMu.Lock()
	cursorAuthCache["memo"] = cursorAuthCacheEntry{info: info, fetchedAt: time.Now()}
	cursorAuthCacheMu.Unlock()

	got := a.InspectAuth(context.Background(), p)
	if !got.Authenticated || got.Status != "Authenticated" {
		t.Fatalf("cached auth = %+v", got)
	}
	if calls.Load() != 0 {
		t.Fatalf("cached InspectAuth must not spawn status probe, calls=%d", calls.Load())
	}
}

func TestListConversationsReadsCursorChatMetadata(t *testing.T) {
	dataDir, hostHome := t.TempDir(), t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("NEXUS_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)
	t.Setenv("USERPROFILE", hostHome)
	home, err := config.ProfileHome("cursor", "cursor2")
	if err != nil {
		t.Fatal(err)
	}
	chatID := "f8ebf955-eda8-482d-bbda-cefee4bfdb32"
	chatDir := filepath.Join(home, ".cursor", "chats", "workspace-hash", chatID)
	if err := os.MkdirAll(chatDir, 0700); err != nil {
		t.Fatal(err)
	}
	meta, err := json.Marshal(map[string]any{"schemaVersion": 1, "createdAtMs": int64(1790000000000), "updatedAtMs": int64(1790000001000), "hasConversation": true, "title": "Cursor shared chat", "cwd": "/workspace/cursor"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chatDir, "meta.json"), meta, 0600); err != nil {
		t.Fatal(err)
	}

	got, err := New().ListConversations(context.Background(), model.Profile{Provider: "cursor", Name: "cursor2"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != chatID || got[0].ProfileID != "cursor2" || got[0].Title != "Cursor shared chat" || got[0].Workspace != "/workspace/cursor" {
		t.Fatalf("Cursor conversations = %+v", got)
	}
	if !New().Capabilities().Conversations {
		t.Fatal("Cursor conversation discovery must be exposed as a supported capability")
	}
}

func TestInspectAuthExtractsEmail(t *testing.T) {
	t.Setenv("AI_CLI_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	invalidateCursorAuthCache("test-email")

	prev := cursorStatusRun
	cursorStatusRun = func(ctx context.Context, bin string, args []string, env []string, cwd string) (string, error) {
		return "Tip: You can start the Cursor CLI with `agent`.\n✓ Logged in as engineer@example.com\n", nil
	}
	t.Cleanup(func() { cursorStatusRun = prev })

	a := New()
	p := model.Profile{Provider: "cursor", Name: "test-email"}
	if err := a.Prepare(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	// Resolve the provider binary through the stubbed seam instead of the
	// host PATH: a hand-joined PATH (":" vs ";") and extensionless fakes
	// both break on Windows. Auth parsing is what this test owns.
	prevLook := spec.BinaryLookPath
	spec.BinaryLookPath = func(name string) (string, error) {
		return filepath.Join(t.TempDir(), name), nil
	}
	t.Cleanup(func() { spec.BinaryLookPath = prevLook })

	got := a.InspectAuth(context.Background(), p)
	if !got.Authenticated {
		t.Fatalf("expected Authenticated=true, got %+v", got)
	}
	if got.Email != "engineer@example.com" {
		t.Fatalf("expected Email='engineer@example.com', got %q", got.Email)
	}
}

func TestInspectAuthRecognizesSuccessfulLoginStatus(t *testing.T) {
	t.Setenv("AI_CLI_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	invalidateCursorAuthCache("login-success")

	previousStatusRun := cursorStatusRun
	cursorStatusRun = func(context.Context, string, []string, []string, string) (string, error) {
		return "✓ Login successful!", nil
	}
	t.Cleanup(func() { cursorStatusRun = previousStatusRun })

	previousLookPath := spec.BinaryLookPath
	spec.BinaryLookPath = func(name string) (string, error) {
		return filepath.Join(t.TempDir(), name), nil
	}
	t.Cleanup(func() { spec.BinaryLookPath = previousLookPath })

	profile := model.Profile{Provider: "cursor", Name: "login-success"}
	info := New().InspectAuth(t.Context(), profile)
	if !info.Authenticated {
		t.Fatalf("successful Cursor status must be authenticated, got %+v", info)
	}
}

func TestCursorStatusAuthenticationRequiresPositiveSignal(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{name: "successful login", output: "✓ Login successful!", want: true},
		{name: "logged in account", output: "Logged in as developer@example.test", want: true},
		{name: "not authenticated", output: "Not authenticated", want: false},
		{name: "unauthenticated", output: "Unauthenticated", want: false},
		{name: "login required", output: "Not logged in. Run cursor-agent login.", want: false},
		{name: "failed login", output: "Login failed: invalid credentials", want: false},
		{name: "empty output", output: "", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := cursorStatusAuthenticated(test.output); got != test.want {
				t.Fatalf("cursorStatusAuthenticated(%q) = %t, want %t", test.output, got, test.want)
			}
		})
	}
}

func TestInspectAuthExtractsIdentityFromCLIConfigAndJWT(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	hostHome := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("NEXUS_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)
	t.Setenv("USERPROFILE", hostHome)

	prevStatusRun := cursorStatusRun
	cursorStatusRun = func(context.Context, string, []string, []string, string) (string, error) {
		return "Authenticated", nil
	}
	t.Cleanup(func() { cursorStatusRun = prevStatusRun })

	prevLookPath := spec.BinaryLookPath
	spec.BinaryLookPath = func(name string) (string, error) {
		return filepath.Join(t.TempDir(), name), nil
	}
	t.Cleanup(func() { spec.BinaryLookPath = prevLookPath })

	a := New()

	// Case 1: cli-config.json provides email and authId
	p1 := model.Profile{Provider: "cursor", Name: "cfg-ident"}
	invalidateCursorAuthCache(p1.Name)
	if err := a.Prepare(t.Context(), p1); err != nil {
		t.Fatal(err)
	}
	home1, err := config.ProfileHome("cursor", p1.Name)
	if err != nil {
		t.Fatal(err)
	}
	cliCfg := map[string]any{
		"authInfo": map[string]string{
			"email":  "user@company.example",
			"authId": "auth0|user_from_config",
		},
	}
	cfgBytes, _ := json.Marshal(cliCfg)
	if err := os.WriteFile(filepath.Join(home1, ".cursor", "cli-config.json"), cfgBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	got1 := a.InspectAuth(t.Context(), p1)
	if !got1.Authenticated || got1.Email != "user@company.example" || got1.ExternalAccountID != "auth0|user_from_config" {
		t.Fatalf("expected cli-config identity, got %+v", got1)
	}

	// Case 2: auth.json JWT sub provides ExternalAccountID when cli-config has no authInfo
	p2 := model.Profile{Provider: "cursor", Name: "jwt-ident"}
	invalidateCursorAuthCache(p2.Name)
	if err := a.Prepare(t.Context(), p2); err != nil {
		t.Fatal(err)
	}
	home2, err := config.ProfileHome("cursor", p2.Name)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home2, ".config", "cursor"), 0o700); err != nil {
		t.Fatal(err)
	}
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"auth0|user_from_jwt"}`))
	jwtToken := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`)) + "." + payload + ".sig"
	authBytes, _ := json.Marshal(map[string]string{"accessToken": jwtToken})
	if err := os.WriteFile(filepath.Join(home2, ".config", "cursor", "auth.json"), authBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	got2 := a.InspectAuth(t.Context(), p2)
	if !got2.Authenticated || got2.ExternalAccountID != "auth0|user_from_jwt" {
		t.Fatalf("expected JWT sub identity, got %+v", got2)
	}
}

func TestDetectHostAuthMemoizesStatusProbe(t *testing.T) {
	hostHome := t.TempDir()
	t.Setenv("NEXUS_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)
	invalidateCursorHostAuthCache()

	binDir := t.TempDir()
	fakeBin := filepath.Join(binDir, "cursor-agent")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\necho 2026.08.11\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	prevLook := spec.BinaryLookPath
	spec.BinaryLookPath = func(name string) (string, error) {
		if name == "cursor-agent" || name == "agent" {
			return fakeBin, nil
		}
		return prevLook(name)
	}
	t.Cleanup(func() {
		spec.BinaryLookPath = prevLook
		invalidateCursorHostAuthCache()
	})

	var calls atomic.Int32
	prevStatus := cursorStatusRun
	cursorStatusRun = func(ctx context.Context, bin string, args []string, env []string, cwd string) (string, error) {
		calls.Add(1)
		return "Not logged in", nil
	}
	t.Cleanup(func() { cursorStatusRun = prevStatus })

	a := New()
	_, ok1 := a.DetectHostAuth(context.Background())
	_, ok2 := a.DetectHostAuth(context.Background())
	if ok1 || ok2 {
		t.Fatalf("expected unauthenticated host probe, got ok1=%v ok2=%v", ok1, ok2)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected second DetectHostAuth call to reuse cached status probe (calls=1), got %d", calls.Load())
	}
}
