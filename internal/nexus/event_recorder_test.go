package nexus

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestEventMetadataRecorderPersistsTimelineIdentity(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "timeline.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	project, err := st.CreateProject(store.Project{
		ID: "prj_timeline", Name: "Timeline", Slug: "timeline", CanonicalPath: "/tmp/timeline",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	agent, err := st.CreateAgent(store.Agent{ID: "agt_timeline", ProjectID: project.ID, Name: "Runner"})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	generation, err := st.AddGeneration(store.RuntimeGeneration{
		ID: "gen_timeline", AgentID: agent.ID, RevisionID: "rev_timeline", RuntimeID: "runtime_timeline",
		Provider: "codex", Profile: "default", StartedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("add generation: %v", err)
	}

	recorder := newEventMetadataRecorder(st)
	recorder(events.NewEventWithCorrelation(
		"lineage-1", generation.RuntimeID, "codex", "default",
		events.EventRuntimeStarted, "Runtime started", map[string]any{"project_id": project.ID},
	))
	recorder(events.NewEventWithCorrelation(
		"", "", "codex", "default",
		events.EventQuotaLow, "Quota low", nil,
	))
	if _, err := st.RecordEventMetadata(store.EventMetadata{
		ID: "evt-legacy-runtime", RuntimeID: generation.RuntimeID,
		Kind: "RUNTIME_STARTED", Timestamp: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("record legacy runtime event: %v", err)
	}

	rows, err := st.ListEventsMetadata(project.ID, "", 10)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected two project events, got %d", len(rows))
	}
	var row store.EventMetadata
	for _, item := range rows {
		if item.ID == "evt-legacy-runtime" {
			row = item
		}
	}
	if row.Kind != "RUNTIME_STARTED" {
		t.Fatalf("expected RUNTIME_STARTED, got %q", row.Kind)
	}
	if row.Source != "" {
		t.Fatalf("expected legacy source to stay empty, got %q", row.Source)
	}
	if row.RuntimeID != generation.RuntimeID {
		t.Fatalf("expected runtime %q, got %q", generation.RuntimeID, row.RuntimeID)
	}
	if row.RuntimeGenerationID != generation.ID {
		t.Fatalf("expected generation %q, got %q", generation.ID, row.RuntimeGenerationID)
	}
	if row.AgentID != agent.ID {
		t.Fatalf("expected agent %q, got %q", agent.ID, row.AgentID)
	}
	if row.ProjectID != project.ID {
		t.Fatalf("expected project %q, got %q", project.ID, row.ProjectID)
	}

	recorded, err := st.ListEventsMetadata(project.ID, agent.ID, 10)
	if err != nil {
		t.Fatalf("list agent events: %v", err)
	}
	if len(recorded) != 2 {
		t.Fatalf("expected two agent events, got %d", len(recorded))
	}
	for _, item := range recorded {
		if item.ID == "evt-legacy-runtime" {
			continue
		}
		row = item
	}
	if row.Source != events.SourceRuntime {
		t.Fatalf("expected source %q, got %q", events.SourceRuntime, row.Source)
	}
	if row.CorrelationID != "lineage-1" {
		t.Fatalf("expected correlation lineage-1, got %q", row.CorrelationID)
	}

	all, err := st.ListEventsMetadata("", "", 10)
	if err != nil {
		t.Fatalf("list all events: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected three durable events, got %d", len(all))
	}
	for _, item := range all {
		if item.Kind == "QUOTA_LOW" && item.Source != events.SourceQuota {
			t.Fatalf("expected quota source, got %q", item.Source)
		}
		if item.Kind == "QUOTA_LOW" && item.AgentID != "" {
			t.Fatalf("expected no agent attribution for account quota event, got %q", item.AgentID)
		}
	}
}
