package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestWorkPlan_CRUD_And_Revisions(t *testing.T) {
	tmpDir := t.TempDir()
	st, err := Open(filepath.Join(tmpDir, "test_nexus.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	proj, err := st.CreateProject(Project{
		CanonicalPath: tmpDir,
		Name:          "Test Project",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	// 1. Create WorkPlan
	plan := WorkPlan{
		ProjectID:   proj.ID,
		Title:       "Sprint 1 Foundation",
		Description: "Initial modular foundation",
		Phases: []PlanPhase{
			{
				ID:    "phase-1",
				Title: "Phase 1: DB & API",
				Order: 1,
				Packages: []WorkPackage{
					{
						ID:                 "pkg-1",
						Title:              "Database Schema Migration",
						Goal:               "Create SQLite tables and migrate safely",
						Priority:           "CRITICAL",
						Role:               "implementer",
						AcceptanceCriteria: []string{"Tables created", "Indexes added"},
					},
				},
			},
		},
		StructuredFacts: map[string]string{
			"db_engine": "sqlite_pure_go",
		},
	}

	created, err := st.CreateWorkPlan(plan)
	if err != nil {
		t.Fatalf("create work plan: %v", err)
	}

	if created.CurrentRevision != 1 {
		t.Errorf("expected current revision 1, got %d", created.CurrentRevision)
	}

	// 2. Fetch WorkPlan
	fetched, err := st.GetWorkPlan(created.ID)
	if err != nil {
		t.Fatalf("get work plan: %v", err)
	}
	if len(fetched.Phases) != 1 || len(fetched.Phases[0].Packages) != 1 {
		t.Fatalf("unexpected phases structure: %+v", fetched.Phases)
	}

	// 3. Update WorkPlan -> creates Revision 2
	fetched.Title = "Sprint 1 Foundation (Revised)"
	updated, rev2, err := st.UpdateWorkPlan(*fetched, "Added second acceptance criteria")
	if err != nil {
		t.Fatalf("update work plan: %v", err)
	}
	if updated.CurrentRevision != 2 || rev2.Revision != 2 {
		t.Errorf("expected revision 2, got %d", updated.CurrentRevision)
	}

	// 4. List revisions
	revs, err := st.ListPlanRevisions(created.ID)
	if err != nil {
		t.Fatalf("list revisions: %v", err)
	}
	if len(revs) != 2 {
		t.Errorf("expected 2 revisions, got %d", len(revs))
	}

	// 5. Create Execution Snapshot
	snap, err := st.CreateExecutionSnapshot(created.ID, rev2.ID, `{"status":"RUNNING","progress":50}`)
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if snap.ID == "" {
		t.Errorf("expected snapshot ID, got empty")
	}

	// 6. Read immutable revision/snapshot back for Mission Runner restart.
	gotRev, err := st.GetPlanRevision(created.ID, 2)
	if err != nil {
		t.Fatalf("get plan revision: %v", err)
	}
	if gotRev.ID != rev2.ID || gotRev.SnapshotJSON == "" {
		t.Fatalf("unexpected revision: %+v", gotRev)
	}

	gotSnap, err := st.GetExecutionSnapshot(snap.ID)
	if err != nil {
		t.Fatalf("get execution snapshot: %v", err)
	}
	if gotSnap.RevisionID != rev2.ID || gotSnap.StateJSON != snap.StateJSON {
		t.Fatalf("unexpected execution snapshot: %+v", gotSnap)
	}
}

func TestUpdateWorkPlanRejectsStaleRevisionWithoutOverwritingCurrent(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "stale-plan.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	project, err := st.CreateProject(Project{CanonicalPath: t.TempDir(), Name: "Revision guard"})
	if err != nil {
		t.Fatal(err)
	}
	created, err := st.CreateWorkPlan(WorkPlan{ProjectID: project.ID, Title: "Original"})
	if err != nil {
		t.Fatal(err)
	}

	stale, err := st.GetWorkPlan(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	current, err := st.GetWorkPlan(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	current.Title = "Current edit"
	if _, _, err := st.UpdateWorkPlan(*current, "current edit"); err != nil {
		t.Fatalf("persist current revision: %v", err)
	}

	stale.Title = "Stale edit"
	if _, _, err := st.UpdateWorkPlan(*stale, "stale edit"); !errors.Is(err, ErrPlanRevisionConflict) {
		t.Fatalf("stale revision error = %v, want ErrPlanRevisionConflict", err)
	}

	latest, err := st.GetWorkPlan(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if latest.CurrentRevision != 2 || latest.Title != "Current edit" {
		t.Fatalf("stale update changed current plan: revision=%d title=%q", latest.CurrentRevision, latest.Title)
	}
	revisions, err := st.ListPlanRevisions(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(revisions) != 2 {
		t.Fatalf("stale update created a revision: got %d revisions, want 2", len(revisions))
	}
}

func TestMissionSchedule_AT_BecomesReady(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "schedule.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	proj, err := st.CreateProject(Project{CanonicalPath: t.TempDir(), Name: "Scheduled"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(WorkPlan{ProjectID: proj.ID, Title: "Ship", Phases: []PlanPhase{{ID: "p", Title: "P", Packages: []WorkPackage{{ID: "w", Title: "W", Goal: "G"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	when := time.Now().UTC().Add(-time.Second)
	created, err := st.CreateMissionSchedule(MissionSchedule{PlanID: plan.ID, ProjectID: proj.ID, Mode: "AT", ScheduledFor: &when})
	if err != nil {
		t.Fatal(err)
	}
	ready, err := st.ListReadyMissionSchedules(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if len(ready) != 1 || ready[0].ID != created.ID {
		t.Fatalf("schedule not ready: %+v", ready)
	}
	if ready[0].AttemptCount != 0 || ready[0].LastAttemptAt != nil || ready[0].LastError != "" {
		t.Fatalf("new schedule diagnostics must start empty: %+v", ready[0])
	}
	claimed, err := st.ClaimMissionSchedule(created.ID)
	if err != nil || !claimed {
		t.Fatalf("claim ready schedule: claimed=%v err=%v", claimed, err)
	}
	got, err := st.GetMissionSchedule(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.AttemptCount != 1 || got.LastAttemptAt == nil || got.LastError != "" {
		t.Fatalf("claim diagnostics not persisted: %+v", got)
	}
}

func TestMissionScheduleBindsRunID(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "schedule-run.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	proj, err := st.CreateProject(Project{CanonicalPath: t.TempDir(), Name: "Scheduled Run"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(WorkPlan{ProjectID: proj.ID, Title: "Ship", Phases: []PlanPhase{{ID: "p", Title: "P", Packages: []WorkPackage{{ID: "w", Title: "W", Goal: "G"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	when := time.Now().UTC().Add(-time.Second)
	item, err := st.CreateMissionSchedule(MissionSchedule{PlanID: plan.ID, ProjectID: proj.ID, Mode: "AT", ScheduledFor: &when})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.BindMissionScheduleRun(item.ID, "run-real"); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetMissionSchedule(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.RunID != "run-real" || got.Status != ScheduleRunning {
		t.Fatalf("run binding not persisted: %+v", got)
	}
}

func TestMissionScheduleClaimIsAtomicAcrossWorkers(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "schedule-claim.db")
	st1, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st1.Close()
	st2, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()

	project, err := st1.CreateProject(Project{CanonicalPath: t.TempDir(), Name: "Atomic schedule"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st1.CreateWorkPlan(WorkPlan{ProjectID: project.ID, Title: "Atomic plan", Phases: []PlanPhase{{ID: "p", Title: "P", Packages: []WorkPackage{{ID: "w", Title: "W", Goal: "G"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	item, err := st1.CreateMissionSchedule(MissionSchedule{ID: "schedule-atomic", PlanID: plan.ID, ProjectID: project.ID, Mode: "AT"})
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan bool, 2)
	for _, st := range []*Store{st1, st2} {
		go func(s *Store) {
			<-start
			claimed, claimErr := s.ClaimMissionSchedule(item.ID)
			if claimErr != nil {
				t.Errorf("claim schedule: %v", claimErr)
			}
			results <- claimed
		}(st)
	}
	close(start)
	claimed := 0
	for range 2 {
		if <-results {
			claimed++
		}
	}
	if claimed != 1 {
		t.Fatalf("expected one worker to claim schedule, got %d", claimed)
	}
}

func TestMissionRunSaveIsFencedByLeaseToken(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "fencing.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	proj, err := st.CreateProject(Project{CanonicalPath: t.TempDir(), Name: "Fence"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(WorkPlan{ProjectID: proj.ID, Title: "Plan", Phases: []PlanPhase{{ID: "p", Title: "P", Packages: []WorkPackage{{ID: "w", Title: "W", Goal: "G"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	rec := MissionRunRecord{ID: "run-fenced", PlanID: plan.ID, ProjectID: proj.ID, State: "EXECUTING", PayloadJSON: `{"id":"run-fenced"}`}
	if err := st.UpsertMissionRun(rec); err != nil {
		t.Fatal(err)
	}
	first, err := st.AcquireMissionLease(rec.ID, "worker-a", 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(15 * time.Millisecond)
	second, err := st.AcquireMissionLease(rec.ID, "worker-b", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	second.State = "PAUSED"
	second.PayloadJSON = `{"id":"run-fenced","state":"PAUSED"}`
	if err := st.UpsertMissionRun(*second); err != nil {
		t.Fatal(err)
	}
	first.State = "FAILED"
	first.PayloadJSON = `{"id":"run-fenced","state":"FAILED"}`
	if err := st.UpsertMissionRun(*first); err == nil {
		t.Fatal("expected stale lease token to be rejected")
	}
	got, err := st.GetMissionRun(rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "PAUSED" {
		t.Fatalf("stale worker overwrote state: %s", got.State)
	}
}
