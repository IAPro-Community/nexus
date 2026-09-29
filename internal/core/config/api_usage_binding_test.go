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
