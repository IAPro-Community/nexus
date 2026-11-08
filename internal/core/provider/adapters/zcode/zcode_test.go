package zcode

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestZCodeCapabilities(t *testing.T) {
	a := New()
	caps := a.Capabilities()
	if !caps.Login || !caps.Logout || !caps.Resume {
		t.Errorf("unexpected capabilities: %+v", caps)
	}
	if caps.Usage {
		t.Errorf("usage should be false by default")
	}
}

func TestValidZCodeCredentials(t *testing.T) {
	if validZCodeCredentials([]byte(`{}`)) {
		t.Errorf("empty json should be invalid")
	}
	if validZCodeCredentials([]byte(`not json`)) {
		t.Errorf("invalid json should be invalid")
	}
	valid := []byte(`{"oauth:active_provider": "zai", "oauth:zai:access_token": "token123"}`)
	if !validZCodeCredentials(valid) {
		t.Errorf("valid json with token should pass")
	}
}

func TestZCodeInspectAuthUnauthenticated(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", tmpDir)
	t.Setenv("AI_REAL_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	a := New()
	info := a.InspectAuth(context.Background(), model.Profile{Name: "test-prof", Provider: "zcode"})
	if info.Authenticated {
		t.Errorf("expected not authenticated, got %+v", info)
	}
}

func TestZCodeImportHostAuth(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	hostHome := t.TempDir()
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)

	// Create fake host zcode credentials
	zcodeDir := filepath.Join(hostHome, ".zcode", "v2")
	if err := os.MkdirAll(zcodeDir, 0700); err != nil {
		t.Fatal(err)
	}
	credsData := []byte(`{"oauth:active_provider": "zai", "oauth:zai:access_token": "secret"}`)
	if err := os.WriteFile(filepath.Join(zcodeDir, "credentials.json"), credsData, 0600); err != nil {
		t.Fatal(err)
	}

	a := New()
	// Test DetectHostAuth
	det, ok := a.DetectHostAuth(context.Background())
	if !ok || !det.Available {
		t.Errorf("expected DetectHostAuth to detect host auth, got %+v, %v", det, ok)
	}

	// Test ImportHostAuth
	imported, err := a.ImportHostAuth(context.Background(), model.Profile{Name: "default", Provider: "zcode"})
	if err != nil || !imported {
		t.Fatalf("ImportHostAuth failed: %v, imported=%v", err, imported)
	}

	info := a.InspectAuth(context.Background(), model.Profile{Name: "default", Provider: "zcode"})
	if !info.Authenticated {
		t.Errorf("expected profile to be authenticated after import, got %+v", info)
	}
}
