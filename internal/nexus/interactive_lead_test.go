package nexus

import (
	"context"
	"sync"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestInteractiveLeadPersistsIdentityAndRuntimeGeneration(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Interactive lead", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	lead, err := n.PrepareInteractiveLead(context.Background(), project.ID, "agy", "work")
	if err != nil {
		t.Fatal(err)
	}
	if lead.Role != interactiveLeadRole || lead.CurrentRevisionID == "" {
		t.Fatalf("lead identity was not persisted: %+v", lead)
	}
	revision, err := st.GetRevision(lead.CurrentRevisionID)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := ParseAgentConfig(revision.Config)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "agy" || cfg.Profile != "work" || cfg.AgentSpec.Role != interactiveLeadRole {
		t.Fatalf("lead runtime preference/spec mismatch: %+v", cfg)
	}

	session := &registry.RuntimeSession{RuntimeID: "interactive-lead-runtime", AgentID: lead.ID, ProviderID: "agy", ProfileID: "work"}
	if err := n.BindInteractiveLeadRuntime(context.Background(), lead.ID, session); err != nil {
		t.Fatal(err)
	}
	registry.DefaultRegistry().Delete(session.RuntimeID)
	recoveredLead, err := n.PrepareInteractiveLead(context.Background(), project.ID, "codex", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if recoveredLead.ID != lead.ID {
		t.Fatalf("dead runtime must recover the existing Lead identity: got %s, want %s", recoveredLead.ID, lead.ID)
	}
	working, err := st.GetAgent(lead.ID, project.ID)
	if err != nil || (working.Status != store.AgentStarting && working.Status != store.AgentWorking) {
		t.Fatalf("lead must remain reserved while re-launching: %+v, err=%v", working, err)
	}
	if err := n.ReleaseInteractiveLeadRuntime(context.Background(), lead.ID, session.RuntimeID); err != nil {
		t.Fatal(err)
	}
	stopped, err := st.GetAgent(lead.ID, project.ID)
	if err != nil || stopped.Status != store.AgentStopped {
		t.Fatalf("lead must be STOPPED after detach: %+v, err=%v", stopped, err)
	}
	generation, err := st.GenerationByRuntimeID(session.RuntimeID)
	if err != nil || generation.StoppedAt == nil {
		t.Fatalf("lead generation must be closed after detach: %+v, err=%v", generation, err)
	}
}

func TestConcurrentInteractiveLeadPreparationClaimsDistinctAgents(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Concurrent leads", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	results := make([]store.Agent, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index], errs[index] = n.PrepareInteractiveLead(context.Background(), project.ID, "codex", "same")
		}(i)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if results[0].ID == results[1].ID {
		t.Fatalf("concurrent launches claimed the same Lead: %s", results[0].ID)
	}
}

func TestInteractiveLeadReservationCanBeReleasedAndReclaimed(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Lead cleanup", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	lead, err := n.PrepareInteractiveLead(context.Background(), project.ID, "codex", "same")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ReleaseInteractiveLeadReservation(lead.ID); err != nil {
		t.Fatal(err)
	}
	reclaimed, err := n.PrepareInteractiveLead(context.Background(), project.ID, "codex", "same")
	if err != nil {
		t.Fatal(err)
	}
	if reclaimed.ID != lead.ID {
		t.Fatalf("released Lead was not reclaimed: got %s want %s", reclaimed.ID, lead.ID)
	}
}
