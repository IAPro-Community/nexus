package nexus

import "testing"

func TestReadOnlyAgentModeRejectsUnsafeConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options map[string]any
	}{
		{name: "yolo", options: map[string]any{"mode": "YOLO"}},
		{name: "custom command", options: map[string]any{"execution_adapter": "command_template"}},
		{name: "dangerous args", options: map[string]any{"extra_args": []any{"--dangerously-skip-permissions"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := AgentConfig{AgentSpec: mustAgentMode(t, "reviewer").AgentSpec, Options: tc.options}
			if err := validateAgentModeConfiguration(cfg); err == nil {
				t.Fatal("read-only mode accepted unsafe execution configuration")
			}
		})
	}
}

func TestReadOnlyAgentModeAllowsSafeNativeConfiguration(t *testing.T) {
	cfg := AgentConfig{
		AgentSpec: mustAgentMode(t, "investigator").AgentSpec,
		Options:   map[string]any{"mode": "Safe", "execution_adapter": "native_alias"},
	}
	if err := validateAgentModeConfiguration(cfg); err != nil {
		t.Fatalf("safe native read-only config rejected: %v", err)
	}
}

func TestReadOnlyAgentModeForcesHeadlessReviewPolicy(t *testing.T) {
	cfg := AgentConfig{AgentSpec: mustAgentMode(t, "security-reviewer").AgentSpec}
	policy := applyAgentModeExecutionPolicy(cfg, agentPromptPolicy{})
	if !policy.Review {
		t.Fatal("read-only Agent Mode did not force provider ReviewMode")
	}

	developer := AgentConfig{AgentSpec: mustAgentMode(t, "developer").AgentSpec}
	if applyAgentModeExecutionPolicy(developer, agentPromptPolicy{}).Review {
		t.Fatal("developer mode was incorrectly forced into read-only review")
	}
}

func mustAgentMode(t *testing.T, id string) AgentMode {
	t.Helper()
	mode, ok := AgentModeForID(id)
	if !ok {
		t.Fatalf("Agent Mode %q not found", id)
	}
	return mode
}
