package nexus

import (
	"context"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestPruneProjectsRemovesOnlyRowsWithoutEvidence(t *testing.T) {
	n := openTestNexus(t)
	service := NewProjectApplicationService(n)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	implicit, err := st.CreateProject(store.Project{
		Name:          "implicit",
		CanonicalPath: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := service.Create(context.Background(), "Opened", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.TouchProject(opened.ID); err != nil {
		t.Fatal(err)
	}

	candidates, err := service.ListProjectsToPrune(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected both projects inspected, got %d", len(candidates))
	}

	result, err := service.PruneProjects(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Removed) != 1 || result.Removed[0].ID != implicit.ID {
		t.Fatalf("unexpected removed set: %+v", result.Removed)
	}
	if _, err := st.GetProject(implicit.ID); err == nil {
		t.Fatal("implicit project must be deleted")
	}
	if _, err := st.GetProject(opened.ID); err != nil {
		t.Fatalf("project opened from the UI must survive: %v", err)
	}
}

func TestPruneProjectsRequiresOptInForOpenedProjects(t *testing.T) {
	n := openTestNexus(t)
	service := NewProjectApplicationService(n)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	opened, err := service.Create(context.Background(), "Opened", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.TouchProject(opened.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := service.PruneProjects(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetProject(opened.ID); err != nil {
		t.Fatalf("opened project must survive the default prune: %v", err)
	}

	if _, err := service.PruneProjects(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetProject(opened.ID); err == nil {
		t.Fatal("--include-opened must prune an opened project with no other evidence")
	}
}

func TestPruneProjectsKeepsProjectsWithPromptHistory(t *testing.T) {
	n := openTestNexus(t)
	service := NewProjectApplicationService(n)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := service.Create(context.Background(), "WithEvents", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.RecordEventMetadata(store.EventMetadata{
		ProjectID: project.ID,
		Kind:      "runtime.started",
		Summary:   "fixture",
	}); err != nil {
		t.Fatal(err)
	}

	result, err := service.PruneProjects(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	for _, removed := range result.Removed {
		if removed.ID == project.ID {
			t.Fatal("a project with recorded events must never be pruned")
		}
	}
	if _, err := st.GetProject(project.ID); err != nil {
		t.Fatalf("event-bearing project must survive: %v", err)
	}
}
