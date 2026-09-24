package nexus

import (
	"context"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestRuntimeExecutionRoleLifecyclePreservesAgentRevision(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Interop", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Persistent", Role: "senior-developer"})
	if err != nil {
		t.Fatal(err)
	}
	revision, err := st.AddRevision(agent.ID, `{"provider":"codex","profile":"default"}`)
	if err != nil {
		t.Fatal(err)
	}
	agent.CurrentRevisionID = revision.ID
	if err := st.UpdateAgent(agent); err != nil {
		t.Fatal(err)
	}
	_, err = st.AddGeneration(store.RuntimeGeneration{
		AgentID: agent.ID, RevisionID: revision.ID, RuntimeID: "rt_role_service",
		Provider: "codex", Profile: "default", StartedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}

	initial, err := n.EffectiveRuntimeExecutionRole("rt_role_service")
	if err != nil {
		t.Fatal(err)
	}
	if initial.Persisted || initial.Role.ID != "build" {
		t.Fatalf("expected non-persisted build default, got %#v", initial)
	}

	before, _ := st.ListRevisions(agent.ID)
	review, err := n.SetRuntimeExecutionRole(context.Background(), "rt_role_service", "review", "test")
	if err != nil {
		t.Fatal(err)
	}
	if !review.Persisted || review.Role.ID != "review" || review.Binding.Enforcement != EnforcementBestEffort {
		t.Fatalf("unexpected review binding: %#v", review)
	}

	build, err := n.SetRuntimeExecutionRole(context.Background(), "rt_role_service", "build", "test")
	if err != nil {
		t.Fatal(err)
	}
	if build.Role.ID != "build" || build.Binding.Enforcement != EnforcementUnverified {
		t.Fatalf("unexpected build binding: %#v", build)
	}

	after, _ := st.ListRevisions(agent.ID)
	if len(after) != len(before) || after[0].ID != before[0].ID {
		t.Fatalf("role lifecycle mutated AgentRevision: before=%#v after=%#v", before, after)
	}
}

func TestSetRuntimeExecutionRoleRejectsUnknownRole(t *testing.T) {
	n := openTestNexus(t)
	if _, err := n.SetRuntimeExecutionRole(context.Background(), "rt_missing", "does-not-exist", "test"); err == nil {
		t.Fatal("expected unknown role to be rejected before persistence")
	}
}
