package app

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/config"
)

func TestConfigAPIUsageCmdBindsAndUnbindsWithoutRawSecret(t *testing.T) {
	t.Setenv("NEXUS_CONFIG_DIR", t.TempDir())

	if err := configAPIUsageCmd([]string{
		"bind", "codex", "work",
		"--project", "proj_123",
		"--key-env", "OPENAI_ADMIN_KEY_WORK",
	}); err != nil {
		t.Fatal(err)
	}
	binding, ok := config.GetAPIUsageBinding("codex", "work")
	if !ok || binding.Provider != "openai" || binding.APIKeyEnv != "OPENAI_ADMIN_KEY_WORK" ||
		len(binding.ProjectIDs) != 1 || binding.ProjectIDs[0] != "proj_123" {
		t.Fatalf("binding=%+v ok=%v", binding, ok)
	}

	if err := configAPIUsageCmd([]string{"bind", "codex", "bad", "--raw-key", "sk-admin-secret"}); err == nil {
		t.Fatal("CLI must reject raw secret flags")
	}

	if err := configAPIUsageCmd([]string{"unbind", "codex", "work"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := config.GetAPIUsageBinding("codex", "work"); ok {
		t.Fatal("binding must be removed")
	}
}
