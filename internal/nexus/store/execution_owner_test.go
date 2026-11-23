package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestWorkPlanExecutionOwnerDefaultsPersistsAndIsImmutable(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "owner.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	proj, err := st.CreateProject(Project{CanonicalPath: t.TempDir(), Name: "Owner"})
	if err != nil {
		t.Fatal(err)
	}

	plan, err := st.CreateWorkPlan(WorkPlan{ProjectID: proj.ID, Title: "Legacy plan"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ExecutionOwner != "NEXUS" {
		t.Fatalf("legacy plan must default to NEXUS owner, got %q", plan.ExecutionOwner)
	}

	maestroPlan, err := st.CreateWorkPlan(WorkPlan{ProjectID: proj.ID, Title: "Maestro plan", ExecutionOwner: "MAESTRO"})
	if err != nil {
		t.Fatal(err)
	}
	if maestroPlan.ExecutionOwner != "MAESTRO" {
		t.Fatalf("expected MAESTRO owner, got %q", maestroPlan.ExecutionOwner)
	}

	if _, err := st.CreateWorkPlan(WorkPlan{ProjectID: proj.ID, Title: "Bad owner", ExecutionOwner: "OTHER"}); err == nil {
		t.Fatal("non-canonical execution owner must be rejected")
	}

	fetched, err := st.GetWorkPlan(maestroPlan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.ExecutionOwner != "MAESTRO" {
		t.Fatalf("owner must persist, got %q", fetched.ExecutionOwner)
	}

	fetched.Title = "Maestro plan v2"
	if _, _, err := st.UpdateWorkPlan(*fetched, "rename"); err != nil {
		t.Fatalf("same-owner update must succeed: %v", err)
	}

	fetched.Title = "Hijacked"
	fetched.ExecutionOwner = "NEXUS"
	if _, _, err := st.UpdateWorkPlan(*fetched, "owner swap"); !errors.Is(err, ErrPlanExecutionOwnerImmutable) {
		t.Fatalf("owner swap must be refused: %v", err)
	}

	after, err := st.GetWorkPlan(maestroPlan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.ExecutionOwner != "MAESTRO" || after.Title != "Maestro plan v2" {
		t.Fatalf("refused owner swap mutated the plan: owner=%q title=%q", after.ExecutionOwner, after.Title)
	}
}

func TestMissionRunExecutionOwnerPersistsAndIsImmutable(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "owner.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	proj, err := st.CreateProject(Project{CanonicalPath: t.TempDir(), Name: "Owner"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(WorkPlan{ProjectID: proj.ID, Title: "Plan"})
	if err != nil {
		t.Fatal(err)
	}

	rec := MissionRunRecord{ID: "run-owner", PlanID: plan.ID, ProjectID: proj.ID, State: "EXECUTING", PayloadJSON: `{"id":"run-owner"}`}
	if err := st.UpsertMissionRun(rec); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetMissionRun(rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ExecutionOwner != "NEXUS" {
		t.Fatalf("legacy run must default to NEXUS owner, got %q", got.ExecutionOwner)
	}

	maestro := MissionRunRecord{ID: "run-maestro", PlanID: plan.ID, ProjectID: proj.ID, State: "EXECUTING", PayloadJSON: `{"id":"run-maestro"}`, ExecutionOwner: "MAESTRO"}
	if err := st.UpsertMissionRun(maestro); err != nil {
		t.Fatal(err)
	}
	got, err = st.GetMissionRun(maestro.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ExecutionOwner != "MAESTRO" {
		t.Fatalf("owner must persist, got %q", got.ExecutionOwner)
	}

	maestro.State = "PAUSED"
	maestro.PayloadJSON = `{"id":"run-maestro","state":"PAUSED"}`
	if err := st.UpsertMissionRun(maestro); err != nil {
		t.Fatalf("same-owner update must succeed: %v", err)
	}

	hijack := maestro
	hijack.ExecutionOwner = "NEXUS"
	hijack.PayloadJSON = `{"id":"run-maestro","state":"FAILED"}`
	if err := st.UpsertMissionRun(hijack); !errors.Is(err, ErrExecutionOwnerImmutable) {
		t.Fatalf("owner swap must be refused: %v", err)
	}

	after, err := st.GetMissionRun(maestro.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.ExecutionOwner != "MAESTRO" || after.State != "PAUSED" {
		t.Fatalf("refused owner swap mutated the run: owner=%q state=%q", after.ExecutionOwner, after.State)
	}

	if err := st.UpsertMissionRun(MissionRunRecord{ID: "run-bad", PlanID: plan.ID, ProjectID: proj.ID, State: "EXECUTING", PayloadJSON: `{}`, ExecutionOwner: "OTHER"}); err == nil {
		t.Fatal("non-canonical execution owner must be rejected")
	}
}
