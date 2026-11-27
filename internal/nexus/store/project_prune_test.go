package store

import (
	"context"
	"testing"
	"time"
)

func createPruneTestProject(t *testing.T, s *Store, name string) Project {
	t.Helper()
	created, err := s.CreateProject(Project{
		Name:          name,
		CanonicalPath: "/tmp/prune-fixtures/" + name,
	})
	if err != nil {
		t.Fatalf("create project %s: %v", name, err)
	}
	return created
}

func TestListProjectPruneCandidatesFlagsRowsWithoutEvidence(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	implicit := createPruneTestProject(t, s, "implicit")
	opened := createPruneTestProject(t, s, "opened")
	if err := s.TouchProject(opened.ID); err != nil {
		t.Fatalf("touch opened: %v", err)
	}
	withLayout := createPruneTestProject(t, s, "with-layout")
	if err := s.SaveLayout(withLayout.ID, `{"panels":["terminals"]}`); err != nil {
		t.Fatalf("save layout: %v", err)
	}
	withAgent := createPruneTestProject(t, s, "with-agent")
	agent := Agent{ID: "agt_prune_fixture", ProjectID: withAgent.ID, Name: "Agent"}
	if _, err := s.CreateAgent(agent); err != nil {
		t.Fatalf("create agent: %v", err)
	}

	candidates, err := s.ListProjectPruneCandidates(ctx)
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}
	byID := map[string]ProjectPruneCandidate{}
	for _, candidate := range candidates {
		byID[candidate.ID] = candidate
	}
	if len(byID) != 4 {
		t.Fatalf("expected every project once, got %d", len(byID))
	}

	if got := byID[implicit.ID]; got.Reason == "" {
		t.Fatalf("implicit project must be a prune candidate: %+v", got)
	}
	for _, project := range []Project{opened, withLayout, withAgent} {
		if byID[project.ID].Reason != "" {
			t.Fatalf("%s must keep evidence, got reason %q", project.Name, byID[project.ID].Reason)
		}
	}
	if byID[withAgent.ID].AgentCount != 1 {
		t.Fatalf("agent count = %d, want 1", byID[withAgent.ID].AgentCount)
	}
	if !byID[withLayout.ID].LayoutSaved {
		t.Fatal("saved layout must be reported")
	}
	if byID[opened.ID].LastOpenedAt == nil {
		t.Fatal("touched project must report last_opened_at")
	}
}

func TestDeleteProjectRowRemovesOnlyThatProject(t *testing.T) {
	s := openTestStore(t)
	keep := createPruneTestProject(t, s, "keep")
	prune := createPruneTestProject(t, s, "prune")

	if err := s.DeleteProjectRow(context.Background(), prune.ID); err != nil {
		t.Fatalf("delete row: %v", err)
	}
	if _, err := s.GetProject(prune.ID); err == nil {
		t.Fatal("pruned project must be gone")
	}
	if _, err := s.GetProject(keep.ID); err != nil {
		t.Fatalf("sibling project must survive: %v", err)
	}
}

func TestDeleteProjectRowHonoursCanceledContext(t *testing.T) {
	s := openTestStore(t)
	project := createPruneTestProject(t, s, "canceled")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.DeleteProjectRow(ctx, project.ID); err == nil {
		t.Fatal("canceled context must not delete")
	}
	if _, err := s.GetProject(project.ID); err != nil {
		t.Fatalf("project must survive a canceled delete: %v", err)
	}
}

func TestPruneCandidateCountsEventsAndIgnoresEmptyLayout(t *testing.T) {
	s := openTestStore(t)
	project := createPruneTestProject(t, s, "with-events")
	if _, err := s.RecordEventMetadata(EventMetadata{
		ID:        "evt_prune_fixture",
		ProjectID: project.ID,
		Kind:      "project.detected",
		Timestamp: time.Now().UTC(),
		Summary:   "fixture",
	}); err != nil {
		t.Fatalf("record event: %v", err)
	}
	candidates, err := s.ListProjectPruneCandidates(context.Background())
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}
	for _, candidate := range candidates {
		if candidate.ID != project.ID {
			continue
		}
		if candidate.EventCount == 0 || candidate.Reason != "" {
			t.Fatalf("events must keep the project out of the candidate list: %+v", candidate)
		}
		return
	}
	t.Fatal("project missing from candidates")
}
