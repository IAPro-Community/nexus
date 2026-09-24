package store

import (
	"testing"
	"time"
)

func TestRuntimeExecutionRoleBindingDoesNotCreateAgentRevision(t *testing.T) {
	s := openTestStore(t)
	project, err := s.CreateProject(Project{Name: "Agent Interop", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := s.CreateAgent(Agent{ProjectID: project.ID, Name: "Senior Developer", Role: "senior-developer"})
	if err != nil {
		t.Fatal(err)
	}
	revision, err := s.AddRevision(agent.ID, `{"provider":"codex"}`)
	if err != nil {
		t.Fatal(err)
	}
	generation, err := s.AddGeneration(RuntimeGeneration{
		AgentID: agent.ID, RevisionID: revision.ID, RuntimeID: "rt_interop",
		Provider: "codex", Profile: "default", StartedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}

	before, err := s.ListRevisions(agent.ID)
	if err != nil {
		t.Fatal(err)
	}

	first, err := s.SetRuntimeExecutionRole(RuntimeExecutionRoleBinding{
		RuntimeGenerationID: generation.ID,
		RoleID:              "review",
		ActivationStrategy:  "NEXUS_OVERLAY",
		Enforcement:         "ENFORCED",
		ActivatedBy:         "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.AgentID != agent.ID || first.Scope != "RUNTIME" {
		t.Fatalf("unexpected binding: %#v", first)
	}

	second, err := s.SetRuntimeExecutionRole(RuntimeExecutionRoleBinding{
		RuntimeGenerationID: generation.ID,
		AgentID:             agent.ID,
		RoleID:              "build",
		ActivationStrategy:  "NEXUS_OVERLAY",
		Enforcement:         "UNVERIFIED",
		ActivatedBy:         "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.RoleID != "build" {
		t.Fatalf("role switch not persisted: %#v", second)
	}
	if second.ID != first.ID {
		t.Fatalf("role switch changed binding identity: first=%s second=%s", first.ID, second.ID)
	}

	got, err := s.RuntimeExecutionRoleByRuntimeID("rt_interop")
	if err != nil {
		t.Fatal(err)
	}
	if got.RoleID != "build" || got.AgentID != agent.ID {
		t.Fatalf("unexpected role binding: %#v", got)
	}

	after, err := s.ListRevisions(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("execution role switch created AgentRevision: before=%d after=%d", len(before), len(after))
	}
}

func TestRuntimeExecutionRoleRejectsForeignAgent(t *testing.T) {
	s := openTestStore(t)
	project, _ := s.CreateProject(Project{Name: "Interop Guard", CanonicalPath: t.TempDir()})
	agentA, _ := s.CreateAgent(Agent{ProjectID: project.ID, Name: "A"})
	agentB, _ := s.CreateAgent(Agent{ProjectID: project.ID, Name: "B"})
	revision, _ := s.AddRevision(agentA.ID, "{}")
	generation, _ := s.AddGeneration(RuntimeGeneration{
		AgentID: agentA.ID, RevisionID: revision.ID, RuntimeID: "rt_guard",
		Provider: "codex", Profile: "default", StartedAt: time.Now().UTC(),
	})

	if _, err := s.SetRuntimeExecutionRole(RuntimeExecutionRoleBinding{
		RuntimeGenerationID: generation.ID,
		AgentID:             agentB.ID,
		RoleID:              "review",
	}); err == nil {
		t.Fatal("expected foreign-agent binding to be rejected")
	}
}

func TestDetachRuntimeExecutionRoleProjectionDoesNotClearNewerReceipt(t *testing.T) {
	s := openTestStore(t)
	project, _ := s.CreateProject(Project{Name: "Projection binding", CanonicalPath: t.TempDir()})
	agent, _ := s.CreateAgent(Agent{ProjectID: project.ID, Name: "A"})
	revision, _ := s.AddRevision(agent.ID, "{}")
	generation, _ := s.AddGeneration(RuntimeGeneration{
		AgentID: agent.ID, RevisionID: revision.ID, RuntimeID: "rt_projection_binding",
		Provider: "opencode", Profile: "default", StartedAt: time.Now().UTC(),
	})
	if _, err := s.SetRuntimeExecutionRole(RuntimeExecutionRoleBinding{
		RuntimeGenerationID: generation.ID,
		AgentID:             agent.ID,
		RoleID:              "review",
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.AttachRuntimeExecutionRoleProjection(generation.ID, "aprj_old"); err != nil {
		t.Fatal(err)
	}
	if err := s.AttachRuntimeExecutionRoleProjection(generation.ID, "aprj_new"); err != nil {
		t.Fatal(err)
	}
	if err := s.DetachRuntimeExecutionRoleProjection(generation.ID, "aprj_old"); err != ErrNotFound {
		t.Fatalf("stale receipt must not detach newer projection, got %v", err)
	}
	got, err := s.RuntimeExecutionRole(generation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ProjectionReceiptID != "aprj_new" {
		t.Fatalf("newer projection link was lost: %#v", got)
	}
	if err := s.DetachRuntimeExecutionRoleProjection(generation.ID, "aprj_new"); err != nil {
		t.Fatal(err)
	}
	got, err = s.RuntimeExecutionRole(generation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ProjectionReceiptID != "" {
		t.Fatalf("current projection link was not detached: %#v", got)
	}
}
