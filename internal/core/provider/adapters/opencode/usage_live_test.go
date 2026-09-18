package opencode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestOpenCodeZenProbeUnsupported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Fatalf("missing auth header")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer server.Close()
	opencodeZenURL = server.URL
	t.Cleanup(func() { opencodeZenURL = opencodeZenModelsURL })

	root := t.TempDir()
	t.Setenv("AI_DATA_DIR", root)
	home, err := config.ProfileHome("opencode", "demo")
	if err != nil {
		t.Fatal(err)
	}
	authDir := filepath.Join(home, ".local", "share", "opencode")
	if err := os.MkdirAll(authDir, 0o700); err != nil {
		t.Fatal(err)
	}
	auth := map[string]any{"opencode": map[string]string{"type": "api", "key": "zen-key"}}
	raw, _ := json.Marshal(auth)
	if err := os.WriteFile(filepath.Join(authDir, "auth.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	snap := New().RefreshUsage(context.Background(), model.Profile{Provider: "opencode", Name: "demo"})
	if snap.Status != model.UsageUnsupported {
		t.Fatalf("status=%s want UNSUPPORTED", snap.Status)
	}
	if snap.Source != model.SourceOfficialAPI {
		t.Fatalf("source=%s", snap.Source)
	}
}

func TestOpenCodeUsageFailsClosedWithoutProfileAuth(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AI_DATA_DIR", root)
	t.Setenv("NEXUS_DATA_DIR", root)
	hostHome := t.TempDir()
	t.Setenv("HOME", hostHome)
	hostAuthDir := filepath.Join(hostHome, ".local", "share", "opencode")
	if err := os.MkdirAll(hostAuthDir, 0o700); err != nil {
		t.Fatal(err)
	}
	hostAuth := map[string]any{"opencode": map[string]string{"type": "api", "key": "host-secret-key"}}
	raw, _ := json.Marshal(hostAuth)
	if err := os.WriteFile(filepath.Join(hostAuthDir, "auth.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	home, err := config.ProfileHome("opencode", "isolated")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}

	snap := New().RefreshUsage(context.Background(), model.Profile{Provider: "opencode", Name: "isolated"})
	if snap.Status != model.UsageUnknown {
		t.Fatalf("status=%s want UNKNOWN (must not use host auth.json)", snap.Status)
	}
	if !strings.Contains(snap.Error, "profile home") {
		t.Fatalf("error=%q want profile-home miss", snap.Error)
	}
}
