package nexus

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestStartMissionRunApprovedRejectsStalePlanRevision(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	project, err := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Plan", "", []store.PlanPhase{{Title: "Phase", Packages: []store.WorkPackage{{Title: "A", Goal: "A", Status: "READY", Role: "implementer", AcceptanceCriteria: []string{"done"}}}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	approved := plan.CurrentRevision
	plan.Title = "Changed"
	if _, _, err := n.UpdateWorkPlan(context.Background(), *plan, "change after UI rendered"); err != nil {
		t.Fatal(err)
	}
	if _, err := n.StartMissionRunApproved(context.Background(), plan.ID, approved, "", runner.DefaultAutonomyContract(), false); err == nil {
		t.Fatal("expected stale approved revision to be rejected")
	}
}

func TestScheduleMissionFreezesApprovedPlanRevision(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	project, err := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Plan", "", []store.PlanPhase{{Title: "Phase", Packages: []store.WorkPackage{{Title: "Original", Goal: "A", Status: "READY", Role: "implementer", AcceptanceCriteria: []string{"done"}}}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	approved := plan.CurrentRevision
	schedule, err := n.ScheduleMission(context.Background(), plan.ID, approved, ScheduleWhenResources, nil, "", "", runner.DefaultAutonomyContract())
	if err != nil {
		t.Fatal(err)
	}
	var payload missionSchedulePayload
	if err := json.Unmarshal([]byte(schedule.ContractJSON), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.PlanRevision != approved {
		t.Fatalf("got revision %d want %d", payload.PlanRevision, approved)
	}

	plan.Phases[0].Packages[0].Title = "Changed later"
	updated, _, err := n.UpdateWorkPlan(context.Background(), *plan, "edit after schedule")
	if err != nil {
		t.Fatal(err)
	}
	if updated.CurrentRevision == approved {
		t.Fatal("expected a new current revision")
	}

	run, err := n.startMissionRunAtRevision(context.Background(), updated, payload.PlanRevision, payload.AgentID, payload.Contract, false)
	if err != nil {
		t.Fatal(err)
	}
	if run.PlanRevision != approved {
		t.Fatalf("run revision=%d want=%d", run.PlanRevision, approved)
	}
	if got := run.PackageRuns[0].Title; got != "Original" {
		t.Fatalf("scheduled run used mutable plan, package=%q", got)
	}
}
