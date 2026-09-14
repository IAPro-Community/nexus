package nexus

import (
	"context"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestNormalizeAgentSpec_LegacyRoleCreatesMinimalCompatibleSpec(t *testing.T) {
	cfg := AgentConfig{Provider: "codex", Profile: "default"}
	normalized := NormalizeAgentSpec(store.Agent{Role: "qa"}, cfg)
	if normalized.AgentSpec.Role != "qa" {
		t.Fatalf("legacy role was not normalized: %#v", normalized.AgentSpec)
	}
	if len(normalized.AgentSpec.Instructions) != 0 {
		t.Fatalf("legacy agent received invented instructions: %#v", normalized.AgentSpec.Instructions)
	}
}

func TestAgentSpecRoundTripsInRevisionConfig(t *testing.T) {
	want := intelligence.AgentSpec{
		Role: "devops", Instructions: []string{"Inspect deployment evidence"},
		Responsibilities: []string{"release validation"}, Capabilities: []string{"inspect", "test"},
		VerificationPolicy: intelligence.VerificationPolicy{RequireEvidence: true, RequireTests: true},
	}
	cfg := AgentConfig{Provider: "claude", Profile: "work", AgentSpec: want}
	got, err := ParseAgentConfig(cfg.ConfigJSON())
	if err != nil {
		t.Fatal(err)
	}
	if got.AgentSpec.Role != want.Role || len(got.AgentSpec.Instructions) != 1 || !got.AgentSpec.VerificationPolicy.RequireTests {
		t.Fatalf("agent spec did not round trip: %#v", got.AgentSpec)
	}
}

func TestAgentSpecProviderSwitchPreservesPersistentSpecialization(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Provider switch", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "QA", Role: "qa"})
	if err != nil {
		t.Fatal(err)
	}
	want := intelligence.AgentSpec{Role: "qa", Instructions: []string{"Collect evidence"}}
	if _, err := n.SafeApply(context.Background(), agent.ID, AgentConfig{Provider: "codex", Profile: "a", AgentSpec: want}); err != nil {
		t.Fatal(err)
	}
	agent, _ = st.GetAgent(agent.ID, "")
	cfg, err := currentAgentConfig(st, agent)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Provider, cfg.Profile = "claude", "b"
	cfg = NormalizeAgentSpec(agent, cfg)
	if cfg.AgentSpec.Role != "qa" || len(cfg.AgentSpec.Instructions) != 1 {
		t.Fatalf("provider switch changed AgentSpec: %#v", cfg.AgentSpec)
	}
}
