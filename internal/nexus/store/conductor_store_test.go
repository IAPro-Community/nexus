package store

import (
	"testing"
	"time"
)

func TestStoreMigrationsApplyConductorPolicy(t *testing.T) {
	st := openTestStore(t)
	project, err := st.CreateProject(Project{Name: "C", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	v, err := st.SchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	if v < 19 {
		t.Fatalf("expected schema version >= 19, got %d", v)
	}

	// identity_kind should default to PERSISTENT.
	agent, err := st.CreateAgent(Agent{ProjectID: project.ID, Name: "Agent A"})
	if err != nil {
		t.Fatal(err)
	}
	if agent.IdentityKind != "" && agent.IdentityKind != "PERSISTENT" {
		t.Fatalf("unexpected identity_kind %q", agent.IdentityKind)
	}

	if _, err := st.GetSetting("nonexistent"); err == nil {
		t.Fatal("expected not-found error for missing setting")
	}
}

func TestStoreAgentIdentityKindRoundTrip(t *testing.T) {
	st := openTestStore(t)
	project, err := st.CreateProject(Project{Name: "Identity", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	agent, err := st.CreateAgent(Agent{ProjectID: project.ID, Name: "W1"})
	if err != nil {
		t.Fatal(err)
	}
	if agent.IdentityKind != "" && agent.IdentityKind != "PERSISTENT" {
		t.Fatalf("expected default identity_kind, got %q", agent.IdentityKind)
	}
	agent.IdentityKind = "EPHEMERAL_WORKER"
	if err := st.UpdateAgent(agent); err != nil {
		t.Fatal(err)
	}
	fetched, err := st.GetAgent(agent.ID, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.IdentityKind != "EPHEMERAL_WORKER" {
		t.Fatalf("expected EPHEMERAL_WORKER, got %q", fetched.IdentityKind)
	}
}

func TestStorePurgeExpiredEventMetadata(t *testing.T) {
	st := openTestStore(t)
	project, err := st.CreateProject(Project{Name: "Purge", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	_, err = st.RecordEventMetadata(EventMetadata{
		AgentID:   "a1",
		ProjectID: project.ID,
		Kind:      "test-old",
		Timestamp: now.AddDate(0, 0, -100),
		Summary:   "old",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.RecordEventMetadata(EventMetadata{
		AgentID:   "a2",
		ProjectID: project.ID,
		Kind:      "test-new",
		Timestamp: now.AddDate(0, 0, -5),
		Summary:   "new",
	})
	if err != nil {
		t.Fatal(err)
	}

	n, err := st.PurgeExpiredEventsSince(now, 90)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 purged event, got %d", n)
	}
	events, err := st.ListEventsMetadata(project.ID, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Kind != "test-new" {
		t.Fatalf("expected 1 remaining event, got %d", len(events))
	}
}

func TestStorePutSettingIdempotent(t *testing.T) {
	st := openTestStore(t)
	if err := st.PutSetting("test_key", "v1"); err != nil {
		t.Fatal(err)
	}
	if err := st.PutSetting("test_key", "v2"); err != nil {
		t.Fatal(err)
	}
	val, err := st.GetSetting("test_key")
	if err != nil {
		t.Fatal(err)
	}
	if val != "v2" {
		t.Fatalf("expected v2, got %q", val)
	}
}

func TestStoreEphemeralWorkerFiltering(t *testing.T) {
	st := openTestStore(t)
	project, err := st.CreateProject(Project{Name: "Filter", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	// Create a persistent agent.
	persistent, err := st.CreateAgent(Agent{ProjectID: project.ID, Name: "Persistent", IdentityKind: "PERSISTENT"})
	if err != nil {
		t.Fatal(err)
	}

	// Create an ephemeral worker.
	ephemeral, err := st.CreateAgent(Agent{ProjectID: project.ID, Name: "Worker", IdentityKind: "EPHEMERAL_WORKER"})
	if err != nil {
		t.Fatal(err)
	}

	// ListAgents should exclude ephemeral workers by default.
	agents, err := st.ListAgents(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].ID != persistent.ID {
		t.Fatalf("expected persistent agent, got %s", agents[0].ID)
	}

	// ListAgentsFiltered with empty string should also exclude.
	agents, err = st.ListAgentsFiltered(project.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent with empty filter, got %d", len(agents))
	}

	// ListAgentsFiltered with EPHEMERAL_WORKER should include only ephemeral.
	agents, err = st.ListAgentsFiltered(project.ID, "EPHEMERAL_WORKER")
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 || agents[0].ID != ephemeral.ID {
		t.Fatalf("expected 1 ephemeral agent, got %d", len(agents))
	}

	// ListAgentsFiltered with PERSISTENT should include only persistent.
	agents, err = st.ListAgentsFiltered(project.ID, "PERSISTENT")
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 || agents[0].ID != persistent.ID {
		t.Fatalf("expected 1 persistent agent, got %d", len(agents))
	}
}

func TestStoreReconcileEphemeralWorkers(t *testing.T) {
	st := openTestStore(t)
	project, err := st.CreateProject(Project{Name: "Reconcile", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	// Create a stopped ephemeral worker.
	_, err = st.CreateAgent(Agent{ProjectID: project.ID, Name: "Stopped", IdentityKind: "EPHEMERAL_WORKER", Status: AgentStopped})
	if err != nil {
		t.Fatal(err)
	}

	// Create a working ephemeral worker (should not be deleted).
	_, err = st.CreateAgent(Agent{ProjectID: project.ID, Name: "Working", IdentityKind: "EPHEMERAL_WORKER", Status: AgentWorking})
	if err != nil {
		t.Fatal(err)
	}

	// Create a persistent agent (should not be deleted).
	_, err = st.CreateAgent(Agent{ProjectID: project.ID, Name: "Persistent", IdentityKind: "PERSISTENT", Status: AgentStopped})
	if err != nil {
		t.Fatal(err)
	}

	n, err := st.ReconcileEphemeralWorkers()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 reconciled, got %d", n)
	}

	// Verify remaining agents: empty filter returns persistent only,
	// explicit EPHEMERAL_WORKER filter returns the surviving worker.
	agents, err := st.ListAgentsFiltered(project.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 || agents[0].Name != "Persistent" {
		t.Fatalf("expected 1 persistent agent after reconciliation, got %d", len(agents))
	}
	workers, err := st.ListAgentsFiltered(project.ID, "EPHEMERAL_WORKER")
	if err != nil {
		t.Fatal(err)
	}
	if len(workers) != 1 || workers[0].Name != "Working" {
		t.Fatalf("expected surviving working ephemeral, got %+v", workers)
	}
}

// Regression: agents.current_revision_id is nullable (legacy rows). Reads
// must map NULL to "" instead of failing with converting NULL to string.
func TestStoreAgentNullRevisionID(t *testing.T) {
	st := openTestStore(t)
	project, err := st.CreateProject(Project{Name: "NullRev", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.DB().Exec(`INSERT INTO agents(id,project_id,name,role,status,current_revision_id,continuity_status,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?)`,
		"ag_nullrev", project.ID, "Legacy", "implementer", "STOPPED", nil, "LIVE_SAME_RUNTIME",
		"2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.GetAgent("ag_nullrev", project.ID)
	if err != nil {
		t.Fatalf("get agent with NULL current_revision_id: %v", err)
	}
	if got.CurrentRevisionID != "" {
		t.Fatalf("expected empty revision id, got %q", got.CurrentRevisionID)
	}
}
