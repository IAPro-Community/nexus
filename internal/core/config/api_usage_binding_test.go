package config

import "testing"

func TestAPIUsageBindingRequiresExplicitAttributionAndKnownProvider(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.APIUsageBindings["codex"] = map[string]APIUsageBinding{
		"work": {
			Provider:   "openai",
			APIKeyEnv:  "OPENAI_ADMIN_KEY",
			ProjectIDs: []string{"proj_123"},
		},
	}
	if issues := cfg.Validate(); len(issues) != 0 {
		t.Fatalf("valid API usage binding issues=%v", issues)
	}

	cfg.APIUsageBindings["codex"]["bad"] = APIUsageBinding{Provider: "openai"}
	if issues := cfg.Validate(); len(issues) == 0 {
		t.Fatal("binding without project/user/api-key attribution must fail validation")
	}

	cfg.APIUsageBindings["codex"]["bad"] = APIUsageBinding{Provider: "unknown", ProjectIDs: []string{"proj_123"}}
	if issues := cfg.Validate(); len(issues) == 0 {
		t.Fatal("unknown API usage provider must fail validation")
	}
}

func TestDefaultConfigInitializesAPIUsageBindings(t *testing.T) {
	cfg := NewDefaultConfig()
	if cfg.APIUsageBindings == nil {
		t.Fatal("APIUsageBindings must be initialized for safe mutation")
	}
}


func TestAPIUsageBindingRoundTripDoesNotPersistSecret(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)

	binding := APIUsageBinding{
		Provider: "openai",
		APIKeyEnv: "MY_OPENAI_ADMIN_KEY",
		ProjectIDs: []string{"proj_123"},
	}
	if err := SetAPIUsageBinding("codex", "work", binding); err != nil {
		t.Fatal(err)
	}
	got, ok := GetAPIUsageBinding("codex", "work")
	if !ok || got.APIKeyEnv != "MY_OPENAI_ADMIN_KEY" || len(got.ProjectIDs) != 1 || got.ProjectIDs[0] != "proj_123" {
		t.Fatalf("binding=%+v ok=%v", got, ok)
	}
	if err := RemoveAPIUsageBinding("codex", "work"); err != nil {
		t.Fatal(err)
	}
	if _, ok := GetAPIUsageBinding("codex", "work"); ok {
		t.Fatal("binding must be removed")
	}
}
