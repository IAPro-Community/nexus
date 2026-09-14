package store_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestEventsMetadata_RecordAndList(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_events.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer st.Close()

	// Create project and agent for foreign keys/references
	p, err := st.CreateProject(store.Project{
		ID:            "prj_01",
		Name:          "Project Alpha",
		Slug:          "alpha",
		CanonicalPath: "/tmp/alpha",
	})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	a, err := st.CreateAgent(store.Agent{
		ID:        "agt_01",
		ProjectID: p.ID,
		Name:      "Coder Agent",
		Role:      "Developer",
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)

	// Record events
	evt1 := store.EventMetadata{
		ProjectID: p.ID,
		AgentID:   a.ID,
		Kind:      "AGENT_WORKING",
		Timestamp: now.Add(-2 * time.Minute),
		Summary:   "Agent started working on task 1",
	}
	rec1, err := st.RecordEventMetadata(evt1)
	if err != nil {
		t.Fatalf("RecordEventMetadata evt1 failed: %v", err)
	}
	if rec1.ID == "" {
		t.Fatalf("expected generated ID for rec1, got empty")
	}

	evt2 := store.EventMetadata{
		ProjectID:     p.ID,
		AgentID:       a.ID,
		Kind:          "APPROVAL_REQUIRED",
		CorrelationID: "run-123",
		Timestamp:     now.Add(-1 * time.Minute),
		Summary:       "Agent requested permission to delete file",
	}
	rec2, err := st.RecordEventMetadata(evt2)
	if err != nil {
		t.Fatalf("RecordEventMetadata evt2 failed: %v", err)
	}
	if rec2.ID == "" {
		t.Fatalf("expected generated ID for rec2, got empty")
	}

	evt3 := store.EventMetadata{
		ProjectID: "prj_02",
		AgentID:   "agt_02",
		Kind:      "PROCESS_STARTED",
		Timestamp: now,
		Summary:   "Process started in prj_02",
	}
	_, err = st.RecordEventMetadata(evt3)
	if err != nil {
		t.Fatalf("RecordEventMetadata evt3 failed: %v", err)
	}

	// Query by project
	eventsProj1, err := st.ListEventsMetadata(p.ID, "", 10)
	if err != nil {
		t.Fatalf("ListEventsMetadata by project failed: %v", err)
	}
	if len(eventsProj1) != 2 {
		t.Fatalf("expected 2 events for project %s, got %d", p.ID, len(eventsProj1))
	}
	// Verify descending order by timestamp
	if eventsProj1[0].Kind != "APPROVAL_REQUIRED" {
		t.Errorf("expected first event to be latest (APPROVAL_REQUIRED), got %s", eventsProj1[0].Kind)
	}
	if eventsProj1[0].CorrelationID != "run-123" {
		t.Errorf("expected correlation ID to survive persistence, got %q", eventsProj1[0].CorrelationID)
	}
	if eventsProj1[1].Kind != "AGENT_WORKING" {
		t.Errorf("expected second event to be older (AGENT_WORKING), got %s", eventsProj1[1].Kind)
	}

	// Query by agent
	eventsAgent1, err := st.ListEventsMetadata("", a.ID, 10)
	if err != nil {
		t.Fatalf("ListEventsMetadata by agent failed: %v", err)
	}
	if len(eventsAgent1) != 2 {
		t.Fatalf("expected 2 events for agent %s, got %d", a.ID, len(eventsAgent1))
	}

	// Query with limit 1
	eventsLimit, err := st.ListEventsMetadata(p.ID, "", 1)
	if err != nil {
		t.Fatalf("ListEventsMetadata with limit failed: %v", err)
	}
	if len(eventsLimit) != 1 {
		t.Fatalf("expected 1 event with limit 1, got %d", len(eventsLimit))
	}
	if eventsLimit[0].Kind != "APPROVAL_REQUIRED" {
		t.Errorf("expected event kind APPROVAL_REQUIRED, got %s", eventsLimit[0].Kind)
	}
}
