package config

import (
	"path/filepath"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestConfigLoadSaveBindings(t *testing.T) {
	cfgDir := t.TempDir()
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Strategy != "best-capacity" {
		t.Fatalf("expected default strategy best-capacity, got %s", cfg.Strategy)
	}

	// Set default profile
	if err := SetDefaultProfile("codex", "work"); err != nil {
		t.Fatal(err)
	}
	d, err := GetDefaultProfile("codex")
	if err != nil || d != "work" {
		t.Fatalf("expected default profile work, got %s (err=%v)", d, err)
	}

	// Bind workspace
	ws := filepath.Join(t.TempDir(), "project")
	if err := BindWorkspace(ws, "codex", "work"); err != nil {
		t.Fatal(err)
	}
	if bound := GetBinding(ws, "codex"); bound != "work" {
		t.Fatalf("expected bound profile work, got %s", bound)
	}

	// Unbind workspace
	if err := UnbindWorkspace(ws, "codex"); err != nil {
		t.Fatal(err)
	}
	if bound := GetBinding(ws, "codex"); bound != "" {
		t.Fatalf("expected empty bound profile after unbind, got %s", bound)
	}
}

func TestConfigValidation(t *testing.T) {
	cfg := NewDefaultConfig()
	if issues := cfg.Validate(); len(issues) > 0 {
		t.Fatalf("default config should be valid, got issues: %+v", issues)
	}

	cfg.Strategy = "invalid-strategy"
	cfg.IsolationPreset = model.IsolationPreset("invalid-preset")
	cfg.MaxConcurrency = 0

	issues := cfg.Validate()
	if len(issues) != 3 {
		t.Fatalf("expected 3 validation issues, got %d: %+v", len(issues), issues)
	}
}

func TestConfigLanguageBackwardCompatibility(t *testing.T) {
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Language != "auto" {
		t.Fatalf("language=%q", cfg.Language)
	}
	cfg.Language = "es"
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig()
	if err != nil || loaded.Language != "es" {
		t.Fatalf("language=%q err=%v", loaded.Language, err)
	}
}

func TestIntelligenceConfigDefaultsOffAndPersistsWithoutSecret(t *testing.T) {
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Intelligence.Mode != IntelligenceOff {
		t.Fatalf("expected intelligence OFF by default, got %q", cfg.Intelligence.Mode)
	}
	cfg.Intelligence = IntelligenceConfig{
		Mode: IntelligenceCLI, Provider: "claude", Profile: "work", Model: "claude-sonnet",
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Intelligence.Mode != IntelligenceCLI || loaded.Intelligence.Provider != "claude" || loaded.Intelligence.Profile != "work" {
		t.Fatalf("unexpected intelligence config: %+v", loaded.Intelligence)
	}
}

func TestConfigValidationRejectsInvalidIntelligenceMode(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Intelligence.Mode = IntelligenceMode("MAGIC")
	issues := cfg.Validate()
	if len(issues) != 1 {
		t.Fatalf("expected one intelligence validation issue, got %v", issues)
	}
}

func TestValidateProfileNameRejectsFlagLikeNames(t *testing.T) {
	if err := ValidateProfileName("--with"); err == nil {
		t.Fatal("profile names starting with '-' must be rejected")
	}
	if err := ValidateProfileName("kivergmail"); err != nil {
		t.Fatalf("valid name rejected: %v", err)
	}
}
