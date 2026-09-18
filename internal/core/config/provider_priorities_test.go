package config

import "testing"

func TestDefaultProviderPrioritiesPreferConfiguredFallbackOrder(t *testing.T) {
	cfg := NewDefaultConfig()
	if cfg.ProviderPriorities["codex"] != 1 || cfg.ProviderPriorities["agy"] != 2 || cfg.ProviderPriorities["opencode"] != 3 {
		t.Fatalf("unexpected provider priorities: %#v", cfg.ProviderPriorities)
	}
}
