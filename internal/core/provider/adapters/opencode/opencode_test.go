package opencode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestOpenCode_Capabilities(t *testing.T) {
	adapter := New()
	caps := adapter.Capabilities()
	if !caps.Login {
		t.Errorf("expected Login=true, got false")
	}
	if !caps.Logout {
		t.Errorf("expected Logout=true, got false")
	}
	if !caps.IsolatedRuntime {
		t.Errorf("expected IsolatedRuntime=true, got false")
	}
}

func TestOpenCode_InspectAuth_LocalLoginless(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	emptyHome := t.TempDir()
	t.Setenv("HOME", emptyHome)

	adapter := New()
	info := adapter.InspectAuth(context.Background(), model.Profile{Provider: "opencode", Name: "local-test"})

	if !info.Authenticated {
		t.Errorf("expected Authenticated=true for local loginless OpenCode, got false")
	}
	if info.Health != model.HealthHealthy {
		t.Errorf("expected HealthHealthy, got %v", info.Health)
	}
	if info.Plan != "OpenCode Local (Ollama/Env)" {
		t.Errorf("unexpected plan: %q", info.Plan)
	}
}

func TestOpenCode_InspectAuth_ZenProvider(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	hostHome := t.TempDir()
	t.Setenv("HOME", hostHome)

	home, err := config.ProfileHome("opencode", "zen-test")
	if err != nil {
		t.Fatal(err)
	}
	authDir := filepath.Join(home, ".local", "share", "opencode")
	if err := os.MkdirAll(authDir, 0o700); err != nil {
		t.Fatal(err)
	}

	authMap := map[string]any{
		"opencode": map[string]string{
			"type": "api",
			"key":  "zen-test-key-12345",
		},
	}
	raw, _ := json.Marshal(authMap)
	if err := os.WriteFile(filepath.Join(authDir, "auth.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	adapter := New()
	info := adapter.InspectAuth(context.Background(), model.Profile{Provider: "opencode", Name: "zen-test"})

	if !info.Authenticated {
		t.Errorf("expected Authenticated=true, got false")
	}
	if info.Health != model.HealthHealthy {
		t.Errorf("expected HealthHealthy, got %v", info.Health)
	}
	if info.Plan != "OpenCode Zen" {
		t.Errorf("expected plan 'OpenCode Zen', got %q", info.Plan)
	}
}

func TestOpenCode_InspectAuth_ExternalProvider(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	hostHome := t.TempDir()
	t.Setenv("HOME", hostHome)

	home, err := config.ProfileHome("opencode", "deepseek-test")
	if err != nil {
		t.Fatal(err)
	}
	authDir := filepath.Join(home, ".local", "share", "opencode")
	if err := os.MkdirAll(authDir, 0o700); err != nil {
		t.Fatal(err)
	}

	authMap := map[string]any{
		"deepseek": map[string]string{
			"type": "api",
			"key":  "sk-deepseek-test-key",
		},
	}
	raw, _ := json.Marshal(authMap)
	if err := os.WriteFile(filepath.Join(authDir, "auth.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	adapter := New()
	info := adapter.InspectAuth(context.Background(), model.Profile{Provider: "opencode", Name: "deepseek-test"})

	if !info.Authenticated {
		t.Errorf("expected Authenticated=true, got false")
	}
	if info.Health != model.HealthHealthy {
		t.Errorf("expected HealthHealthy, got %v", info.Health)
	}
	if info.Plan != "OpenCode (deepseek)" {
		t.Errorf("expected plan 'OpenCode (deepseek)', got %q", info.Plan)
	}
	if info.Email != "deepseek" {
		t.Errorf("expected email 'deepseek', got %q", info.Email)
	}
}

func TestOpenCode_Prepare_CopiesHostAuth(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	hostHome := t.TempDir()
	t.Setenv("HOME", hostHome)

	hostAuthDir := filepath.Join(hostHome, ".local", "share", "opencode")
	if err := os.MkdirAll(hostAuthDir, 0o700); err != nil {
		t.Fatal(err)
	}
	hostAuthMap := map[string]any{
		"deepseek": map[string]string{
			"type": "api",
			"key":  "sk-host-key",
		},
	}
	raw, _ := json.Marshal(hostAuthMap)
	if err := os.WriteFile(filepath.Join(hostAuthDir, "auth.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	adapter := New()
	p := model.Profile{Provider: "opencode", Name: "sync-test"}
	if err := adapter.Prepare(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	home, err := config.ProfileHome("opencode", "sync-test")
	if err != nil {
		t.Fatal(err)
	}
	profileAuthFile := filepath.Join(home, ".local", "share", "opencode", "auth.json")
	copiedData, err := os.ReadFile(profileAuthFile)
	if err != nil {
		t.Fatalf("expected profile auth.json to be copied: %v", err)
	}
	if string(copiedData) != string(raw) {
		t.Fatalf("copied auth content mismatch: got %s, want %s", string(copiedData), string(raw))
	}
}
