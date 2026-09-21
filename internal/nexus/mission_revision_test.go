package nexus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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

	frozen, _, err := loadWorkPlanRevision(st, updated.ID, payload.PlanRevision)
	if err != nil {
		t.Fatal(err)
	}
	if frozen.CurrentRevision != approved {
		t.Fatalf("frozen revision=%d want=%d", frozen.CurrentRevision, approved)
	}
	if got := frozen.Phases[0].Packages[0].Title; got != "Original" {
		t.Fatalf("scheduled revision used mutable plan, package=%q", got)
	}
	if _, err := n.ScheduleMission(context.Background(), updated.ID, approved, ScheduleWhenResources, nil, "", "", runner.DefaultAutonomyContract()); err != nil {
		t.Fatalf("historical approved revision must remain schedulable after later edits: %v", err)
	}
}

func TestScheduleMissionRejectsUnknownPlanRevision(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	project, err := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Plan", "", []store.PlanPhase{{
		Title: "Phase",
		Packages: []store.WorkPackage{{
			Title: "A", Goal: "A", Status: "READY", Role: "implementer", AcceptanceCriteria: []string{"done"},
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := n.ScheduleMission(context.Background(), plan.ID, plan.CurrentRevision+100, ScheduleWhenResources, nil, "", "", runner.DefaultAutonomyContract()); err == nil {
		t.Fatal("schedule must reject a plan revision that has no immutable snapshot")
	}
}

func TestCostBudgetAdmissionRequiresFreshExplicitConfirmation(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Budgeted", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Plan", "", []store.PlanPhase{{
		Title: "Phase",
		Packages: []store.WorkPackage{{
			Title: "A", Goal: "A", Status: "READY", Role: "implementer", AcceptanceCriteria: []string{"done"},
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	prefs := store.DefaultCostPreferences()
	prefs.HardLimitMicroUSD = 1_000_000
	if err := st.PutCostPreferences(prefs); err != nil {
		t.Fatal(err)
	}

	if err := validateCostBudgetAdmission(context.Background(), st, plan, ""); err == nil {
		t.Fatal("hard budget with unknown headroom must require explicit confirmation")
	}

	summary, err := NewCostApplicationService(st).Summary(context.Background(), project.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	token := costBudgetConfirmationToken(plan.ID, plan.CurrentRevision, summary)
	if token == "" {
		t.Fatal("expected non-empty confirmation token")
	}
	if err := validateCostBudgetAdmission(context.Background(), st, plan, token); err != nil {
		t.Fatalf("fresh confirmation rejected: %v", err)
	}

	knownCost := int64(250_000)
	if _, err := st.RecordCostObservation(store.CostObservation{
		ProjectID: project.ID, Provider: "codex", Profile: "work",
		CostMicroUSD: &knownCost, CostKind: store.CostKindProviderReported,
		Source: "provider_response", Status: "COMPLETED",
		ObservedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := validateCostBudgetAdmission(context.Background(), st, plan, token); err == nil {
		t.Fatal("stale confirmation token must be rejected after cost evidence changes")
	}
}
