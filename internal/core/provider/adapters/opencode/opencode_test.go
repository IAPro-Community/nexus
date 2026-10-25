package opencode

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
)

// stubOpenCodeBinary decouples auth tests from host state: the presence of
// the provider binary is stubbed instead of depending on PATH, /usr/local,
// or developer directories of the machine running the test.
func stubOpenCodeBinary(t *testing.T) {
	t.Helper()
	prev := spec.BinaryLookPath
	spec.BinaryLookPath = func(name string) (string, error) {
		if name == "opencode" {
			return filepath.Join(t.TempDir(), "opencode"), nil
		}
		return prev(name)
	}
	t.Cleanup(func() { spec.BinaryLookPath = prev })
}

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
	stubOpenCodeBinary(t)
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
	stubOpenCodeBinary(t)
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
	stubOpenCodeBinary(t)
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
	// getBaseHome prefers NEXUS/AI_REAL_HOME, then USERPROFILE (always set
	// on Windows runners), and only then $HOME: pin the override so host
	// auth resolution stays hermetic on every OS.
	t.Setenv("NEXUS_REAL_HOME", hostHome)

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

func TestOpenCode_ImportHostAuthConfirmsIsolatedCopy(t *testing.T) {
	stubOpenCodeBinary(t)
	dataDir := t.TempDir()
	t.Setenv("AI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	hostHome := t.TempDir()
	t.Setenv("HOME", hostHome)
	t.Setenv("NEXUS_REAL_HOME", hostHome)
	authDir := filepath.Join(hostHome, ".local", "share", "opencode")
	if err := os.MkdirAll(authDir, 0o700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{
		"anthropic": map[string]string{"type": "api", "key": "anthropic-host-key"},
	})
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(authDir, "auth.json")
	if err := os.WriteFile(source, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	profile := model.Profile{Provider: "opencode", Name: "imported"}
	adapter := New()
	copied, err := adapter.ImportHostAuth(t.Context(), profile)
	if err != nil || !copied {
		t.Fatalf("import = %v, %v; want true, nil", copied, err)
	}
	if !adapter.InspectAuth(t.Context(), profile).Authenticated {
		t.Fatal("imported OpenCode profile was not authenticated")
	}
	home, err := config.ProfileHome("opencode", profile.Name)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(home, ".local", "share", "opencode", "auth.json")
	imported, err := os.ReadFile(destination)
	if err != nil || string(imported) != string(raw) {
		t.Fatalf("destination did not receive host auth: err=%v", err)
	}
	unchanged, err := os.ReadFile(source)
	if err != nil || string(unchanged) != string(raw) {
		t.Fatalf("host OpenCode auth changed: err=%v", err)
	}
}

// TestOpenCode_InspectAuth_BinaryMissingFailsClosed proves an installed
// credential without a launchable binary is reported unavailable instead of
// healthy: absence of the binary must never read as authenticated.
func TestOpenCode_InspectAuth_BinaryMissingFailsClosed(t *testing.T) {
	prev := spec.BinaryLookPath
	spec.BinaryLookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	t.Cleanup(func() { spec.BinaryLookPath = prev })

	dataDir := t.TempDir()
	t.Setenv("AI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("HOME", t.TempDir())

	adapter := New()
	info := adapter.InspectAuth(context.Background(), model.Profile{Provider: "opencode", Name: "missing-bin"})
	if info.Authenticated {
		t.Error("missing binary must not authenticate")
	}
	if info.Health != model.HealthUnavailable {
		t.Errorf("missing binary health = %v, want UNAVAILABLE", info.Health)
	}
}
