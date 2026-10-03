package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nexus.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestStoreOpenMigratesIdempotently(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nexus.db")
	s1, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	v1, _ := s1.SchemaVersion()
	if v1 < 1 {
		t.Fatalf("expected schema version >= 1, got %d", v1)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("close after first open: %v", err)
	}

	// Re-open must not re-apply migrations or fail.
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()
	v2, _ := s2.SchemaVersion()
	if v2 != v1 {
		t.Fatalf("schema version changed on reopen: %d -> %d", v1, v2)
	}
}

func TestEventCorrelationMigrationIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event-correlation.db")
	s1, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	version, err := s1.SchemaVersion()
	if err != nil {
		t.Fatalf("schema version: %v", err)
	}
	if version < 15 {
		t.Fatalf("expected correlation migration, got schema version %d", version)
	}
	var columnCount int
	if err := s1.DB().QueryRow(`SELECT COUNT(*) FROM pragma_table_info('events_metadata') WHERE name='correlation_id'`).Scan(&columnCount); err != nil {
		t.Fatalf("inspect correlation column: %v", err)
	}
	if columnCount != 1 {
		t.Fatalf("expected one correlation column, got %d", columnCount)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("close after correlation migration: %v", err)
	}

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer s2.Close()
	version2, _ := s2.SchemaVersion()
	if version2 != version {
		t.Fatalf("schema version changed after reopen: %d -> %d", version, version2)
	}
}

func TestProjectCRUD(t *testing.T) {
	s := openTestStore(t)
	dir := t.TempDir()

	p, err := s.CreateProject(Project{Name: "Omega API", CanonicalPath: dir})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if !strings.HasPrefix(p.ID, "prj_") {
		t.Errorf("expected prj_ id prefix, got %q", p.ID)
	}
	if p.Slug != "omega-api" {
		t.Errorf("expected slug omega-api, got %q", p.Slug)
	}
	if p.MaestroMode != MaestroOff {
		t.Errorf("default maestro mode should be OFF, got %q", p.MaestroMode)
	}
	if p.DefaultIsolation != "project" {
		t.Errorf("default isolation should be project, got %q", p.DefaultIsolation)
	}

	got, err := s.GetProject(p.ID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if got.CanonicalPath != dir {
		t.Errorf("canonical path mismatch: %q != %q", got.CanonicalPath, dir)
	}

	// MRU ordering
	p2dir := t.TempDir()
	p2, err := s.CreateProject(Project{Name: "Omnia", CanonicalPath: p2dir})
	if err != nil {
		t.Fatalf("create second project: %v", err)
	}
	_ = s.TouchProject(p.ID) // p becomes most recent
	list, err := s.ListProjects()
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if len(list) < 2 || list[0].ID != p.ID {
		t.Errorf("expected project %s first (MRU), got %+v", p.ID, list)
	}

	// Delete
	if err := s.DeleteProject(p2.ID); err != nil {
		t.Fatalf("delete project: %v", err)
	}
	if _, err := s.GetProject(p2.ID); err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestCanonicalPathValidation(t *testing.T) {
	if _, err := CanonicalPath(""); err == nil {
		t.Error("empty path must be rejected")
	}
	if _, err := CanonicalPath("/definitely/not/exists-xyz"); err == nil {
		t.Error("non-existent path must be rejected")
	}
	f := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(f, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CanonicalPath(f); err == nil {
		t.Error("file path must be rejected (not a directory)")
	}
}

func TestListAgentSummaries(t *testing.T) {
	s := openTestStore(t)
	emptyProj, err := s.CreateProject(Project{Name: "Empty", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatalf("create empty project: %v", err)
	}
	activeProj, err := s.CreateProject(Project{Name: "Active", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatalf("create active project: %v", err)
	}

	working, err := s.CreateAgent(Agent{ProjectID: activeProj.ID, Name: "Worker", Status: "WORKING"})
	if err != nil {
		t.Fatalf("create working agent: %v", err)
	}
	if err := s.SetAgentStatus(working.ID, "WORKING"); err != nil {
		t.Fatalf("set working status: %v", err)
	}
	if _, err := s.CreateAgent(Agent{ProjectID: activeProj.ID, Name: "Idle", Status: "STOPPED"}); err != nil {
		t.Fatalf("create stopped agent: %v", err)
	}

	summaries, err := s.ListAgentSummaries()
	if err != nil {
		t.Fatalf("list agent summaries: %v", err)
	}

	byID := map[string]ProjectAgentSummary{}
	for _, summary := range summaries {
		byID[summary.ProjectID] = summary
	}
	if _, ok := byID[emptyProj.ID]; ok {
		t.Fatalf("projects with zero agents must be omitted, found %s", emptyProj.ID)
	}
	got, ok := byID[activeProj.ID]
	if !ok {
		t.Fatalf("expected summary for active project %s", activeProj.ID)
	}
	if got.AgentCount != 2 {
		t.Fatalf("expected agent_count 2, got %d", got.AgentCount)
	}
	if got.WorkingCount != 1 {
		t.Fatalf("expected working_count 1, got %d", got.WorkingCount)
	}
}

func TestAgentLifecycle(t *testing.T) {
	s := openTestStore(t)
	projDir := t.TempDir()
	proj, err := s.CreateProject(Project{Name: "Alpha", CanonicalPath: projDir})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	a, err := s.CreateAgent(Agent{ProjectID: proj.ID, Name: "Backend Developer"})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if !strings.HasPrefix(a.ID, "agt_") {
		t.Errorf("expected agt_ prefix, got %q", a.ID)
	}

	// Config revisions
	rev1, err := s.AddRevision(a.ID, `{"provider":"auto","model":"default"}`)
	if err != nil {
		t.Fatalf("add revision 1: %v", err)
	}
	rev2, err := s.AddRevision(a.ID, `{"provider":"auto","model":"deep"}`)
	if err != nil {
		t.Fatalf("add revision 2: %v", err)
	}
	if rev1.Revision != 1 || rev2.Revision != 2 {
		t.Errorf("expected revisions 1,2 got %d,%d", rev1.Revision, rev2.Revision)
	}
	revs, err := s.ListRevisions(a.ID)
	if err != nil || len(revs) != 2 {
		t.Fatalf("expected 2 revisions, got %d (%v)", len(revs), err)
	}
	if revs[0].Revision != 2 {
		t.Errorf("newest revision first, got %d", revs[0].Revision)
	}

	// Runtime generations
	g1, err := s.AddGeneration(RuntimeGeneration{
		AgentID: a.ID, RevisionID: rev1.ID, RuntimeID: "rt_abc", Provider: "codex",
		Profile: "personal", ProviderSession: "sess-1", StartedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("add generation: %v", err)
	}
	gens, err := s.ListGenerations(a.ID)
	if err != nil || len(gens) != 1 {
		t.Fatalf("expected 1 generation, got %d (%v)", len(gens), err)
	}
	cur, err := s.CurrentGeneration(a.ID)
	if err != nil || cur.ID != g1.ID {
		t.Fatalf("current generation mismatch: %v %+v", err, cur)
	}

	// Lineage
	if err := s.AddLineage(LineageEntry{
		AgentID: a.ID, Relation: "CONTEXT_HANDOFF", SourceRuntime: "rt_abc",
		SourceSession: "sess-1", TargetRuntime: "rt_def", TargetSession: "sess-9",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("add lineage: %v", err)
	}
	lin, err := s.ListLineage(a.ID)
	if err != nil || len(lin) != 1 {
		t.Fatalf("expected 1 lineage entry, got %d (%v)", len(lin), err)
	}

	// IDOR guard: reading agent under a foreign project must fail.
	otherProj, _ := s.CreateProject(Project{Name: "Other", CanonicalPath: t.TempDir()})
	if _, err := s.GetAgent(a.ID, otherProj.ID); err != ErrNotFound {
		t.Errorf("expected ErrNotFound for cross-project agent access, got %v", err)
	}

	// Update + delete
	a.Status = "WORKING"
	if err := s.UpdateAgent(a); err != nil {
		t.Fatalf("update agent: %v", err)
	}
	if err := s.DeleteAgent(a.ID, proj.ID); err != nil {
		t.Fatalf("delete agent: %v", err)
	}
	if _, err := s.GetAgent(a.ID, proj.ID); err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestProjectLayoutPersistence(t *testing.T) {
	s := openTestStore(t)
	proj, _ := s.CreateProject(Project{Name: "L", CanonicalPath: t.TempDir()})
	if err := s.SaveLayout(proj.ID, `{"openAgents":["agt_a"],"splits":[]}`); err != nil {
		t.Fatalf("save layout: %v", err)
	}
	layout, err := s.GetLayout(proj.ID)
	if err != nil || !strings.Contains(layout, "agt_a") {
		t.Fatalf("layout roundtrip failed: %q (%v)", layout, err)
	}
}

func TestProjectLayoutV4MonotonicRevision(t *testing.T) {
	s := openTestStore(t)
	proj, err := s.CreateProject(Project{Name: "L4", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	// 1. Initial get returns revision 0 and empty layout
	rec, err := s.GetLayoutRecord(proj.ID)
	if err != nil {
		t.Fatalf("initial get layout record: %v", err)
	}
	if rec.Revision != 0 {
		t.Fatalf("expected initial revision 0, got %d", rec.Revision)
	}

	// 2. First save with expected revision 0 creates revision 1
	rec1, err := s.SaveLayoutWithRevision(proj.ID, `{"version":4,"model":{}}`, 0)
	if err != nil {
		t.Fatalf("first save: %v", err)
	}
	if rec1.Revision != 1 {
		t.Fatalf("expected revision 1 after first save, got %d", rec1.Revision)
	}

	// 3. Second save with expected revision 1 advances to revision 2
	rec2, err := s.SaveLayoutWithRevision(proj.ID, `{"version":4,"model":{"updated":true}}`, 1)
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	if rec2.Revision != 2 {
		t.Fatalf("expected revision 2 after second save, got %d", rec2.Revision)
	}

	// 4. Stale save with expected revision 1 fails with ErrRevisionConflict
	_, err = s.SaveLayoutWithRevision(proj.ID, `{"version":4,"model":{"stale":true}}`, 1)
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("expected ErrRevisionConflict on stale revision 1, got %v", err)
	}

	// 5. Unconditional save (expectedRevision = 0) advances to revision 3
	rec3, err := s.SaveLayoutWithRevision(proj.ID, `{"version":4,"model":{"force":true}}`, 0)
	if err != nil {
		t.Fatalf("unconditional save: %v", err)
	}
	if rec3.Revision != 3 {
		t.Fatalf("expected revision 3 after unconditional save, got %d", rec3.Revision)
	}
}

func TestProjectLayoutV4SurvivesProcessRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nexus.db")
	s1, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	project, err := s1.CreateProject(Project{Name: "Restart", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"version":4,"model":{"active":"flow:F5"},"presentation":{"mode":"MOSAIC"}}`
	if _, err := s1.SaveLayoutWithRevision(project.ID, want, 0); err != nil {
		t.Fatal(err)
	}
	if err := s1.Close(); err != nil {
		t.Fatal(err)
	}
	s2, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	record, err := s2.GetLayoutRecord(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.Revision != 1 || record.Layout != want {
		t.Fatalf("layout did not survive restart: revision=%d layout=%q", record.Revision, record.Layout)
	}
}


func TestProjectSettingPatchPreservesUnrelatedSettings(t *testing.T) {
	s := openTestStore(t)
	project, err := s.CreateProject(Project{
		Name:          "Settings patch",
		CanonicalPath: t.TempDir(),
		Settings:      `{"appearance":{"density":"compact"},"other":true}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	binding := map[string]any{
		"target_id":      "devbox",
		"approval_token": "token-1",
	}
	if err := s.SetProjectSetting(project.ID, "managed_execution_target", binding); err != nil {
		t.Fatal(err)
	}

	updated, err := s.GetProject(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]json.RawMessage
	if err := json.Unmarshal([]byte(updated.Settings), &settings); err != nil {
		t.Fatal(err)
	}
	if string(settings["other"]) != "true" {
		t.Fatalf("unrelated setting was lost: %s", updated.Settings)
	}
	var appearance map[string]string
	if err := json.Unmarshal(settings["appearance"], &appearance); err != nil {
		t.Fatal(err)
	}
	if appearance["density"] != "compact" {
		t.Fatalf("appearance setting changed: %+v", appearance)
	}
	var gotBinding map[string]string
	if err := json.Unmarshal(settings["managed_execution_target"], &gotBinding); err != nil {
		t.Fatal(err)
	}
	if gotBinding["target_id"] != "devbox" || gotBinding["approval_token"] != "token-1" {
		t.Fatalf("binding=%+v", gotBinding)
	}

	if err := s.DeleteProjectSetting(project.ID, "managed_execution_target"); err != nil {
		t.Fatal(err)
	}
	updated, err = s.GetProject(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	settings = map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(updated.Settings), &settings); err != nil {
		t.Fatal(err)
	}
	if _, exists := settings["managed_execution_target"]; exists {
		t.Fatalf("managed setting was not removed: %s", updated.Settings)
	}
	if _, exists := settings["appearance"]; !exists {
		t.Fatalf("unrelated setting was removed: %s", updated.Settings)
	}
}
