package nexus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/quota"
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

func TestScheduleMissionWithOptionsPersistsBudgetConfirmation(t *testing.T) {
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
	item, err := n.ScheduleMissionWithOptions(
		context.Background(),
		plan.ID,
		plan.CurrentRevision,
		ScheduleWhenResources,
		nil,
		"",
		"",
		runner.DefaultAutonomyContract(),
		MissionAdmissionOptions{CostConfirmationToken: "budget-token"},
	)
	if err != nil {
		t.Fatal(err)
	}
	var payload missionSchedulePayload
	if err := json.Unmarshal([]byte(item.ContractJSON), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.CostConfirmationToken != "budget-token" {
		t.Fatalf("cost confirmation token=%q, want persisted token", payload.CostConfirmationToken)
	}
}

func TestScheduleAdmissionFailureIsDurableAndDoesNotRetrySilently(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	project, err := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Plan", "", []store.PlanPhase{{
		Title: "Phase",
		Packages: []store.WorkPackage{{
			Title: "A", Goal: "A", Status: "READY", Role: "implementer",
			AcceptanceCriteria: []string{"done"}, VerificationRequirements: []string{"true"},
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	when := time.Now().UTC().Add(-time.Second)
	item, err := n.ScheduleMission(context.Background(), plan.ID, plan.CurrentRevision, ScheduleAt, &when, "", "", runner.DefaultAutonomyContract())
	if err != nil {
		t.Fatal(err)
	}
	if err := n.processMissionSchedules(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetMissionSchedule(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != store.ScheduleFailed || got.AttemptCount != 1 || got.LastAttemptAt == nil || got.LastError == "" {
		t.Fatalf("failed schedule diagnostics missing: %+v", got)
	}
}

func TestWhenResourcesDoesNotClaimWithoutEligibleResource(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	project, err := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Plan", "", []store.PlanPhase{{
		Title: "Phase",
		Packages: []store.WorkPackage{{
			Title: "A", Goal: "A", Status: "READY", Role: "implementer",
			AcceptanceCriteria: []string{"done"}, VerificationRequirements: []string{"true"},
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	item, err := n.ScheduleMission(context.Background(), plan.ID, plan.CurrentRevision, ScheduleWhenResources, nil, "", "", runner.DefaultAutonomyContract())
	if err != nil {
		t.Fatal(err)
	}
	if err := n.processMissionSchedules(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetMissionSchedule(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != store.SchedulePending || got.AttemptCount != 0 || got.LastAttemptAt != nil {
		t.Fatalf("WHEN_RESOURCES should wait without spinning when no resource is eligible: %+v", got)
	}
}

func TestAutonomousResourceReadinessHonorsIsolationContract(t *testing.T) {
	live := &quota.QuotaView{Status: "LIVE"}
	base := ProviderAccount{
		Profile: "default", Authenticated: true, Available: true, Health: "healthy",
		QuotaView: live, QuotaRemaining: 0.8, QuotaTotal: 1,
		Capabilities: map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED"},
	}

	gemini := base
	gemini.ID = "gemini-default"
	gemini.Provider = "gemini"
	if hasAutonomousResourceForContract([]ProviderAccount{gemini}, runner.DefaultAutonomyContract()) {
		t.Fatal("WHEN_RESOURCES must not claim for an unverified isolation provider")
	}

	codex := base
	codex.ID = "codex-default"
	codex.Provider = "codex"
	if !hasAutonomousResourceForContract([]ProviderAccount{codex}, runner.DefaultAutonomyContract()) {
		t.Fatal("verified Codex isolation should satisfy the default autonomous contract")
	}

	unrestricted := runner.DefaultAutonomyContract()
	unrestricted.DisallowDestructiveGit = false
	unrestricted.AllowGitPush = true
	unrestricted.AllowDeploy = true
	unrestricted.AllowExternalNetwork = true
	unrestricted.AllowSecretAccess = true
	unrestricted.AllowPaidServices = true
	if !hasAutonomousResourceForContract([]ProviderAccount{gemini}, unrestricted) {
		t.Fatal("explicit unrestricted contract should not require verified containment")
	}
}

func TestStartMissionRunFreezesGoalContractInExecutionSnapshot(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Goal snapshot", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	facts, err := PersistGoalContractFact(nil, GoalContract{
		GoalDescription:      "corrigir roteamento sem perder o prompt",
		AutoPromoteToMission: true,
		LeadID:               "lead-1",
		ProjectID:            project.ID,
		GoalID:               "goal-snapshot",
		CreatedAt:            time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Goal snapshot", "", []store.PlanPhase{{
		Title: "Build",
		Packages: []store.WorkPackage{{
			Title: "Fix routing", Goal: "Fix routing", Status: "READY", Role: "implementer",
			AcceptanceCriteria: []string{"routing fixed"}, VerificationRequirements: []string{"true"},
		}},
	}}, facts)
	if err != nil {
		t.Fatal(err)
	}

	run, err := n.StartMissionRun(context.Background(), plan.ID, "lead-1", runner.DefaultAutonomyContract(), false)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := n.loadMissionSnapshot(run)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Goal == nil {
		t.Fatal("execution snapshot lost GoalContract")
	}
	if snapshot.Goal.GoalID != "goal-snapshot" ||
		snapshot.Goal.ProjectID != project.ID ||
		snapshot.Goal.LeadID != "lead-1" ||
		snapshot.Goal.GoalDescription != "corrigir roteamento sem perder o prompt" ||
		!snapshot.Goal.AutoPromoteToMission {
		t.Fatalf("unexpected frozen GoalContract: %+v", snapshot.Goal)
	}
}
