package nexus

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestMissionAdmissionBlockersReportsExactFailures(t *testing.T) {
	report := &FlowPreflightReport{
		Ready: false,
		Checks: []FlowPreflightCheck{
			{Key: "dag_validity", Status: "PASS", Summary: "ok"},
			{Key: "worktree_isolation", Status: "FAIL", Summary: "step build pins Agent unsafe without effective Isolation=worktree"},
			{Key: "resources", Status: "FAIL", Summary: "nenhum recurso de provedor disponível para admissão strict"},
		},
	}
	got := missionAdmissionBlockers(report)
	for _, want := range []string{
		"worktree_isolation: step build pins Agent unsafe without effective Isolation=worktree",
		"resources: nenhum recurso de provedor disponível para admissão strict",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("blockers=%q missing %q", got, want)
		}
	}
	if strings.Contains(got, "dag_validity") {
		t.Fatalf("PASS checks must not leak into blockers: %q", got)
	}
}

func TestMissionAdmissionBlockersFailsClosedWithoutSpecificCheck(t *testing.T) {
	got := missionAdmissionBlockers(&FlowPreflightReport{Ready: false})
	if !strings.Contains(got, "not ready") {
		t.Fatalf("unexpected fallback blocker: %q", got)
	}
}

func TestStartMissionRunTypesMalformedGoalContractAsPreparationFailure(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Malformed goal", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Malformed goal", "", []store.PlanPhase{{
		Title: "Build",
		Packages: []store.WorkPackage{{
			Title: "Step", Goal: "Step", Status: "READY", Role: "implementer",
			AcceptanceCriteria: []string{"done"}, VerificationRequirements: []string{"true"},
		}},
	}}, map[string]string{GoalContractFactKey: "{"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = n.StartMissionRun(context.Background(), plan.ID, "", runner.DefaultAutonomyContract(), false)
	if err == nil {
		t.Fatal("malformed goal contract must fail before MissionRunner ownership")
	}
	var preparationErr *MissionPreparationError
	if !errors.As(err, &preparationErr) {
		t.Fatalf("expected MissionPreparationError, got %T: %v", err, err)
	}
	if preparationErr.Stage != "goal_contract" {
		t.Fatalf("preparation stage=%q want goal_contract", preparationErr.Stage)
	}
}

func TestMissionPreparationErrorPreservesCause(t *testing.T) {
	cause := errors.New("disk unavailable")
	err := missionPreparationError("snapshot", cause)
	if !errors.Is(err, cause) {
		t.Fatalf("preparation error must preserve cause: %v", err)
	}
	var preparationErr *MissionPreparationError
	if !errors.As(err, &preparationErr) || preparationErr.Stage != "snapshot" {
		t.Fatalf("unexpected typed preparation error: %#v", preparationErr)
	}
}
