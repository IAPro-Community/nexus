package app

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/config"
)

func TestConfigFallbackRequiresAndPersistsExplicitOptIn(t *testing.T) {
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv("AI_CLI_DATA_DIR", t.TempDir())

	if err := configCmd([]string{"fallback", "enable"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AutomaticFallback {
		t.Fatal("config fallback enable did not persist authorization")
	}

	if err := configCmd([]string{"fallback", "disable"}); err != nil {
		t.Fatal(err)
	}
	cfg, err = config.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AutomaticFallback {
		t.Fatal("config fallback disable did not revoke authorization")
	}
}
