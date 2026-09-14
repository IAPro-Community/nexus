package nexus

import (
	"context"
	"errors"
	"testing"

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
