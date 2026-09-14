package nexus

import (
	"context"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// --- Gate 3: AgentConfig + Safe Apply ---

func TestParseAgentConfig(t *testing.T) {
	raw := `{"provider":"claude","profile":"default","model":"sonnet","options":{"temperature":0.7}}`
	cfg, err := ParseAgentConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "claude" {
		t.Errorf("provider = %q, want claude", cfg.Provider)
	}
	if cfg.Model != "sonnet" {
		t.Errorf("model = %q, want sonnet", cfg.Model)
	}
	if cfg.Options["temperature"] != 0.7 {
		t.Errorf("options.temperature = %v, want 0.7", cfg.Options["temperature"])
	}
}

func TestParseAgentConfigEmpty(t *testing.T) {
	cfg, err := ParseAgentConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "" {
		t.Errorf("empty config should have empty provider, got %q", cfg.Provider)
	}
}

func TestAnalyzeImpactNoChange(t *testing.T) {
	cfg := AgentConfig{Provider: "claude", Profile: "default"}
	impact := AnalyzeImpact(cfg, cfg)
	if len(impact.ChangedFields) != 0 {
		t.Errorf("no change should produce empty changed fields, got %v", impact.ChangedFields)
	}
	if impact.Mode != "" {
		t.Errorf("no change should have empty mode, got %q", impact.Mode)
	}
}

func TestAnalyzeImpactAgentSpecChangeRequiresNewSession(t *testing.T) {
	current := AgentConfig{Provider: "claude", Profile: "default", AgentSpec: intelligence.AgentSpec{Role: "qa"}}
	proposed := current
	proposed.AgentSpec.Instructions = []string{"Collect evidence"}
	impact := AnalyzeImpact(current, proposed)
	if impact.Mode != ImpactNewSession || !impact.RequiresRestart || !impact.RequiresNewSess {
		t.Fatalf("AgentSpec change must require a new session: %+v", impact)
	}
	if len(impact.ChangedFields) != 1 || impact.ChangedFields[0] != "agent_spec" {
		t.Fatalf("unexpected changed fields: %v", impact.ChangedFields)
	}
}

func TestAnalyzeImpactProviderChange(t *testing.T) {
	current := AgentConfig{Provider: "claude", Profile: "default"}
	proposed := AgentConfig{Provider: "openai", Profile: "default"}
	impact := AnalyzeImpact(current, proposed)
	if impact.Mode != ImpactNewSession {
		t.Errorf("provider change should be NEW_SESSION, got %q", impact.Mode)
	}
	if !impact.RequiresNewSess {
		t.Error("provider change should require new session")
	}
	found := false
	for _, f := range impact.ChangedFields {
		if f == "provider" {
			found = true
		}
	}
	if !found {
		t.Error("changed fields should include 'provider'")
	}
}

func TestAnalyzeImpactModelChange(t *testing.T) {
	current := AgentConfig{Provider: "claude", Model: "sonnet"}
	proposed := AgentConfig{Provider: "claude", Model: "opus"}
	impact := AnalyzeImpact(current, proposed)
	if impact.Mode != ImpactRestartRuntime {
		t.Errorf("model change should be RESTART_RUNTIME, got %q", impact.Mode)
	}
	if !impact.RequiresRestart {
		t.Error("model change should require restart")
	}
}

func TestAnalyzeImpactMaestroModeChange(t *testing.T) {
	current := AgentConfig{Provider: "claude", MaestroMode: "OFF"}
	proposed := AgentConfig{Provider: "claude", MaestroMode: "ASSIST"}
	impact := AnalyzeImpact(current, proposed)
	if impact.Mode != ImpactLiveSameRuntime {
		t.Errorf("maestro mode change should be LIVE_SAME_RUNTIME, got %q", impact.Mode)
	}
	if impact.RequiresRestart || impact.RequiresNewSess {
		t.Error("maestro mode change should not require restart or new session")
	}
}

func TestAnalyzeImpactModeChangeRequiresNewSession(t *testing.T) {
	current := AgentConfig{
		Provider: "claude",
		Profile:  "default",
		Options:  map[string]any{"mode": "Safe"},
	}
	proposed := AgentConfig{
		Provider: "claude",
		Profile:  "default",
		Options:  map[string]any{"mode": "YOLO"},
	}
	impact := AnalyzeImpact(current, proposed)
	if impact.Mode != ImpactNewSession {
		t.Errorf("mode change should be NEW_SESSION, got %q", impact.Mode)
	}
	if !impact.RequiresNewSess {
		t.Error("mode change should require new session")
	}
	if !impact.RequiresRestart {
		t.Error("mode change should require restart")
	}
}

func TestAnalyzeImpactContinuityPolicyNewSession(t *testing.T) {
	current := AgentConfig{Provider: "claude", Profile: "default"}
	proposed := AgentConfig{
		Provider:         "claude",
		Profile:          "default",
		ContinuityPolicy: "new_session",
	}
	impact := AnalyzeImpact(current, proposed)
	if impact.Mode != ImpactNewSession {
		t.Errorf("continuity_policy new_session should be NEW_SESSION, got %q", impact.Mode)
	}
	if !impact.RequiresNewSess {
		t.Error("continuity_policy new_session should require new session")
	}
}

func TestSafeApplyCreatesRevision(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	proposed := AgentConfig{Provider: "fake", Profile: "default", Model: "test"}
	impact, err := n.SafeApply(context.Background(), agent.ID, proposed)
	if err != nil {
		t.Fatalf("SafeApply: %v", err)
	}
	if len(impact.ChangedFields) == 0 {
		t.Error("expected changed fields after first config apply")
	}

	// Verify revision was created.
	agent, _ = st.GetAgent(agent.ID, "")
	if agent.CurrentRevisionID == "" {
		t.Fatal("agent should have a current revision after SafeApply")
	}
	rev, err := st.GetRevision(agent.CurrentRevisionID)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := ParseAgentConfig(rev.Config)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "fake" || cfg.Model != "test" {
		t.Errorf("revision config = %+v, want provider=fake model=test", cfg)
	}
}

func TestSafeApplyStoppedAgentPersistsConfigWithoutLaunching(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	proposed := AgentConfig{Provider: "fake", Profile: "default", Model: "test"}
	impact, err := n.SafeApply(context.Background(), agent.ID, proposed)
	if err != nil {
		t.Fatalf("SafeApply: %v", err)
	}
	if impact.Mode != ImpactNewSession && impact.Mode != ImpactRestartRuntime {
		t.Fatalf("impact mode = %q, want restart/new-session impact recorded for next start", impact.Mode)
	}

	agent, _ = st.GetAgent(agent.ID, "")
	if agent.Status != store.AgentStopped {
		t.Fatalf("stopped agent status = %q after config apply, want STOPPED", agent.Status)
	}
	if agent.CurrentRevisionID == "" {
		t.Fatal("stopped agent should persist a current config revision")
	}
	if _, err := st.CurrentGeneration(agent.ID); err == nil {
		t.Fatal("configuring a stopped agent must not launch a runtime generation")
	}
}

func TestStartAgentReusesPersistedConfigRevision(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	proposed := AgentConfig{Provider: "fake", Profile: "default", Model: "test"}
	if _, err := n.SafeApply(context.Background(), agent.ID, proposed); err != nil {
		t.Fatalf("SafeApply: %v", err)
	}
	before, err := st.ListRevisions(agent.ID)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := n.StartAgent(context.Background(), agent.ID, "fake", "default"); err != nil {
		t.Fatalf("StartAgent: %v", err)
	}
	after, err := st.ListRevisions(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("StartAgent created duplicate revision: before=%d after=%d", len(before), len(after))
	}
}

func TestSafeApplyNoop(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	// Apply empty config twice — second should be no-op.
	proposed := AgentConfig{}
	impact1, _ := n.SafeApply(context.Background(), agent.ID, proposed)
	impact2, _ := n.SafeApply(context.Background(), agent.ID, proposed)

	// First apply on empty agent: all fields are "changed" from zero value.
	// Second apply: no change.
	if len(impact2.ChangedFields) != 0 {
		t.Errorf("second identical apply should be no-op, got changed: %v", impact2.ChangedFields)
	}
	_ = impact1
}

func TestAllocateResourcePersistsEligibleAccount(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	project, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Dev"})

	allocation, err := n.allocateResourceFromAccounts(context.Background(), agent.ID, "codex", "pro", []ProviderAccount{{
		ID: "codex:pro", Provider: "codex", Profile: "pro", Authenticated: true, Available: true, Health: "healthy",
	}}, PolicyManual)
	if err != nil {
		t.Fatalf("allocate resource: %v", err)
	}
	if !allocation.Persisted || allocation.Decision.Selected.ID != "codex:pro" {
		t.Fatalf("allocation = %+v, want persisted codex:pro", allocation)
	}

	agent, _ = st.GetAgent(agent.ID, "")
	revision, err := st.GetRevision(agent.CurrentRevisionID)
	if err != nil {
		t.Fatal(err)
	}
	config, err := ParseAgentConfig(revision.Config)
	if err != nil {
		t.Fatal(err)
	}
	if config.Provider != "codex" || config.Profile != "pro" {
		t.Fatalf("persisted config = %+v, want codex/pro", config)
	}
}

func TestAllocateResourceRejectsUnavailableAccount(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	project, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Dev"})

	_, err := n.allocateResourceFromAccounts(context.Background(), agent.ID, "codex", "default", []ProviderAccount{{
		ID: "codex:default", Provider: "codex", Profile: "default", Authenticated: true, Available: false, Health: "healthy",
	}}, PolicyManual)
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("expected unavailable resource error, got %v", err)
	}
}

func TestResolveStartParamsRejectsUnpersistedOverride(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	project, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Dev"})

	_, _, err := n.ResolveStartParams(agent.ID, "codex", "default")
	if err == nil || !strings.Contains(err.Error(), "persisted resource") {
		t.Fatalf("expected persisted-resource error, got %v", err)
	}
}
