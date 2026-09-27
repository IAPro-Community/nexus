package store

import (
	"testing"
	"time"
)

func TestAgentProjectionReceiptLifecycle(t *testing.T) {
	s := openTestStore(t)
	project, err := s.CreateProject(Project{Name: "Projection", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := s.CreateAgent(Agent{ProjectID: project.ID, Name: "Reviewer"})
	if err != nil {
		t.Fatal(err)
	}
	revision, err := s.AddRevision(agent.ID, "{}")
	if err != nil {
		t.Fatal(err)
	}
	generation, err := s.AddGeneration(RuntimeGeneration{
		AgentID: agent.ID, RevisionID: revision.ID, RuntimeID: "rt_projection",
		Provider: "opencode", Profile: "default", StartedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}

	record := AgentProjectionReceiptRecord{
		ID:                  "aprj_test",
		RuntimeGenerationID: generation.ID,
		AgentID:             agent.ID,
		Provider:            "opencode",
		Profile:             "default",
		RoleID:              "review",
		Scope:               "NEXUS_PROFILE",
		Strategy:            "NATIVE_ON_LAUNCH",
		ManagedRoot:         t.TempDir(),
		RendererVersion:     "opencode-v1-agent-md/1",
		SourceHash:          "source",
		PayloadJSON:         `{"id":"aprj_test"}`,
		AppliedAt:           time.Now().UTC(),
	}
	if err := s.SaveAgentProjectionReceipt(record); err != nil {
		t.Fatal(err)
	}

	got, err := s.AgentProjectionReceipt(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.RuntimeGenerationID != generation.ID || got.AgentID != agent.ID || got.Provider != "opencode" {
		t.Fatalf("unexpected projection record: %#v", got)
	}

	verified := time.Now().UTC().Add(time.Second)
	if err := s.MarkAgentProjectionVerified(record.ID, verified, `{"id":"aprj_test","verified":true}`); err != nil {
		t.Fatal(err)
	}
	got, err = s.AgentProjectionReceipt(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.VerifiedAt == nil || got.PayloadJSON == record.PayloadJSON {
		t.Fatalf("verification was not persisted: %#v", got)
	}

	removed := time.Now().UTC().Add(2 * time.Second)
	if err := s.MarkAgentProjectionRemoved(record.ID, removed); err != nil {
		t.Fatal(err)
	}
	got, err = s.AgentProjectionReceipt(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.RemovedAt == nil {
		t.Fatalf("removal was not persisted: %#v", got)
	}
}

func TestAgentProjectionReceiptRejectsForeignAgent(t *testing.T) {
	s := openTestStore(t)
	project, _ := s.CreateProject(Project{Name: "Projection guard", CanonicalPath: t.TempDir()})
	agentA, _ := s.CreateAgent(Agent{ProjectID: project.ID, Name: "A"})
	agentB, _ := s.CreateAgent(Agent{ProjectID: project.ID, Name: "B"})
	revision, _ := s.AddRevision(agentA.ID, "{}")
	generation, _ := s.AddGeneration(RuntimeGeneration{
		AgentID: agentA.ID, RevisionID: revision.ID, RuntimeID: "rt_projection_guard",
		Provider: "opencode", Profile: "default", StartedAt: time.Now().UTC(),
	})
	err := s.SaveAgentProjectionReceipt(AgentProjectionReceiptRecord{
		ID:                  "aprj_foreign",
		RuntimeGenerationID: generation.ID,
		AgentID:             agentB.ID,
		Provider:            "opencode",
		Profile:             "default",
		RoleID:              "review",
		Scope:               "NEXUS_PROFILE",
		Strategy:            "NATIVE_ON_LAUNCH",
		ManagedRoot:         t.TempDir(),
		RendererVersion:     "opencode-v1-agent-md/1",
		SourceHash:          "source",
		PayloadJSON:         "{}",
		AppliedAt:           time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected foreign-agent projection receipt to be rejected")
	}
}
