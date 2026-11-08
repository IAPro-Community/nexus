package nexus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestAgentApplicationServiceCRUDProjectsEffectiveState(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Agents", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	service := NewAgentApplicationService(n)
	agent, err := service.Create(context.Background(), project.ID, "Reviewer", "reviewer")
	if err != nil {
		t.Fatal(err)
	}

	name := "Senior Reviewer"
	updated, err := service.Update(context.Background(), agent.ID, AgentPatch{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != name || updated.Role != "reviewer" {
		t.Fatalf("unexpected updated agent: %#v", updated)
	}

	detail, err := service.Detail(context.Background(), agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Agent.ID != agent.ID || detail.EffectiveState == "" {
		t.Fatalf("incomplete agent detail: %#v", detail)
	}
}

func TestAgentApplicationServiceRejectsCanceledContext(t *testing.T) {
	n := openTestNexus(t)
	service := NewAgentApplicationService(n)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := service.List(ctx, "project"); !errors.Is(err, context.Canceled) {
		t.Fatalf("list error = %v, want context canceled", err)
	}
}

func TestAgentApplicationServiceListScopesProjectAndProjectsEffectiveRuntimeState(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Agents", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	otherProject, err := st.CreateProject(store.Project{Name: "Other", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	working, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Runtime ended"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddGeneration(store.RuntimeGeneration{
		AgentID: working.ID, RevisionID: "rev_missing", RuntimeID: "rt_dead_for_list",
		Provider: "fake", Profile: "default", StartedAt: time.Now().UTC(), State: "RUNNING",
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetAgentStatus(working.ID, store.AgentWorking); err != nil {
		t.Fatal(err)
	}
	completed, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Completed", Status: "COMPLETED"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := st.CreateAgent(store.Agent{ProjectID: otherProject.ID, Name: "Other project"})
	if err != nil {
		t.Fatal(err)
	}

	agents, err := NewAgentApplicationService(n).List(context.Background(), project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 2 {
		t.Fatalf("project agent count = %d, want 2: %#v", len(agents), agents)
	}
	states := make(map[string]string, len(agents))
	for _, agent := range agents {
		states[agent.ID] = agent.Status
		if agent.ID == other.ID {
			t.Fatalf("list leaked agent from another project: %#v", agent)
		}
	}
	if states[working.ID] != store.AgentRecoverable {
		t.Errorf("dead runtime status = %q, want %q", states[working.ID], store.AgentRecoverable)
	}
	if states[completed.ID] != "COMPLETED" {
		t.Errorf("completed status = %q, want COMPLETED", states[completed.ID])
	}
}
