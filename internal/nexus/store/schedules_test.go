package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestMissionScheduleAttemptDiagnosticsPersist(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "schedule.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	project, err := st.CreateProject(Project{Name: "P", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(WorkPlan{ProjectID: project.ID, Title: "Plan"})
	if err != nil {
		t.Fatal(err)
	}
	when := time.Now().UTC()
	item, err := st.CreateMissionSchedule(MissionSchedule{
		PlanID: plan.ID, ProjectID: project.ID, Mode: "AT", ScheduledFor: &when, ContractJSON: "{}",
	})
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := st.ClaimMissionSchedule(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !claimed {
		t.Fatal("expected first schedule claim to succeed")
	}
	if err := st.FailMissionSchedule(item.ID, "budget confirmation expired"); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetMissionSchedule(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ScheduleFailed || got.AttemptCount != 1 || got.LastAttemptAt == nil {
		t.Fatalf("attempt diagnostics not persisted: %+v", got)
	}
	if got.LastError != "budget confirmation expired" {
		t.Fatalf("last_error=%q", got.LastError)
	}
}
