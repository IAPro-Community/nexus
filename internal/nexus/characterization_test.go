package nexus

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// These tests freeze the useful pre-consolidation contracts before the Agent
// semantics and execution-context changes. They intentionally exercise the
// current store and compiler instead of asserting implementation details.
func TestCharacterization_LegacyAgentRoleAndGenerationRevision(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "nexus.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	project, err := st.CreateProject(store.Project{Name: "Characterization", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "QA", Role: "qa"})
	if err != nil {
		t.Fatal(err)
	}
	revision, err := st.AddRevision(agent.ID, `{"provider":"codex","profile":"default"}`)
	if err != nil {
		t.Fatal(err)
	}
	generation, err := st.AddGeneration(store.RuntimeGeneration{
		AgentID: agent.ID, RevisionID: revision.ID, RuntimeID: "runtime-characterization",
		Provider: "codex", Profile: "default", StartedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if generation.RevisionID != revision.ID {
		t.Fatalf("generation revision link changed: got %q want %q", generation.RevisionID, revision.ID)
	}
	loaded, err := st.GetAgent(agent.ID, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Role != "qa" {
		t.Fatalf("legacy role changed: got %q", loaded.Role)
	}
}

func TestCharacterization_WorkPackageRoleReachesExistingCompiler(t *testing.T) {
	engine := intelligence.NewNexusEngine(nil)
	compiled, err := engine.CompilePrompt(context.Background(), intelligence.WorkPackageOutline{
		Title: "Review release", Goal: "Review the release", Priority: "HIGH", Role: "reviewer",
		Acceptance: []string{"evidence exists"},
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if compiled.PackageTitle != "Review release" {
		t.Fatalf("compiler title changed: %q", compiled.PackageTitle)
	}
	if compiled.SystemPrompt == "" || !contains(compiled.SystemPrompt, "Role: reviewer") {
		t.Fatalf("work package role is not represented in current compiler: %q", compiled.SystemPrompt)
	}
}

func contains(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
