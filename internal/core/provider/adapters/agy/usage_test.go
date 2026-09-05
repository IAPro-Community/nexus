package agy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

func TestGetUsageDoesNotPrepareOrInvokeAGY(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", configDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", configDir)
	t.Setenv("AI_CLI_CONFIG_DIR", configDir)

	snap := New().GetUsage(context.Background(), model.Profile{Provider: "agy", Name: "profile"})
	if snap.Status != model.UsageUnknown {
		t.Fatalf("status=%s want UNKNOWN when no read-only cache exists", snap.Status)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "profiles")); !os.IsNotExist(err) {
		t.Fatalf("GetUsage created profile data: err=%v", err)
	}
}
