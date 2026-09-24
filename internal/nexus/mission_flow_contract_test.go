package nexus

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestPlanToRunnerSpecCarriesFlowStepContracts(t *testing.T) {
	plan := &store.WorkPlan{ID: "plan", ProjectID: "project", CurrentRevision: 3, Phases: []store.PlanPhase{{ID: "phase", Packages: []store.WorkPackage{{
		ID: "step", Title: "Step", Goal: "Goal", Priority: "HIGH", Role: "tester", AssignmentStrategy: "AUTO", ResourcePolicy: "PRESERVE_QUOTA", Provider: "codex", Profile: "fast",
		SkillIDs: []string{"verification"}, MaestroSkills: []string{"verification"}, RelevantPaths: []string{"internal"}, AcceptanceCriteria: []string{"done"}, VerificationRequirements: []string{"go test ./..."},
	}}}}}
	spec, err := planToRunnerSpec(plan, "snapshot")
	if err != nil {
		t.Fatal(err)
	}
	got := spec.Packages[0]
	if got.AssignmentStrategy != "AUTO" || got.ResourcePolicy != "PRESERVE_QUOTA" || got.Provider != "codex" || got.Profile != "fast" {
		t.Fatalf("assignment/resource contract lost: %+v", got)
	}
	if !reflect.DeepEqual(got.SkillIDs, []string{"verification"}) || !reflect.DeepEqual(got.MaestroSkills, []string{"verification"}) || !reflect.DeepEqual(got.RelevantPaths, []string{"internal"}) || !reflect.DeepEqual(got.VerificationRequirements, []string{"go test ./..."}) {
		t.Fatalf("context/verification contract lost: %+v", got)
	}
}

func TestFreezePlanUsesGenericCatalogWithoutMaestro(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = func() MaestroStatus {
		return MaestroStatus{Available: false, Mode: MaestroOff, Error: "offline for test"}
	}
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Generic skills", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	plan := store.WorkPlan{ID: "plan", ProjectID: project.ID, CurrentRevision: 1, Phases: []store.PlanPhase{{ID: "phase", Packages: []store.WorkPackage{{
		ID: "step", Title: "Step", Goal: "Goal", Role: "tester", Status: "READY", AssignmentStrategy: "AUTO", SkillIDs: []string{"coding"}, AcceptanceCriteria: []string{"done"}, VerificationRequirements: []string{"true"},
	}}}}}
	frozen, _, err := freezePlanForExecution(n, plan)
	if err != nil {
		t.Fatalf("generic SkillIDs must freeze without Maestro: %v", err)
	}
	pkg := frozen.Phases[0].Packages[0]
	if !reflect.DeepEqual(pkg.SkillIDs, []string{"coding"}) || !reflect.DeepEqual(pkg.MaestroSkills, []string{"coding"}) {
		t.Fatalf("frozen canonical/compatibility skill contract lost: %#v", pkg)
	}
}

func TestFreezePlanMigratesLegacySkillFieldsThroughGenericCatalog(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = func() MaestroStatus { return MaestroStatus{Available: false, Mode: MaestroOff} }
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Legacy skills", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan := store.WorkPlan{ID: "legacy-plan", ProjectID: project.ID, CurrentRevision: 1, Phases: []store.PlanPhase{{ID: "phase", Packages: []store.WorkPackage{{
		ID: "step", Title: "Step", Goal: "Goal", Role: "tester", Status: "READY", AssignmentStrategy: "AUTO", MaestroSkills: []string{"testing"}, AcceptanceCriteria: []string{"done"}, VerificationRequirements: []string{"true"},
	}}}}}
	frozen, _, err := freezePlanForExecution(n, plan)
	if err != nil {
		t.Fatalf("legacy skill fields should use the generic catalog without Maestro: %v", err)
	}
	pkg := frozen.Phases[0].Packages[0]
	if !reflect.DeepEqual(pkg.SkillIDs, []string{"testing"}) {
		t.Fatalf("canonical SkillIDs not migrated: %#v", pkg.SkillIDs)
	}
}

func TestFreezePlanDoesNotTreatLegacyMaestroGatesAsSkills(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = func() MaestroStatus { return MaestroStatus{Available: false, Mode: MaestroOff} }
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Gate boundary", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan := store.WorkPlan{
		ID: "plan-gate-boundary", ProjectID: project.ID, CurrentRevision: 1,
		Phases: []store.PlanPhase{{ID: "phase-1", Packages: []store.WorkPackage{{
			ID: "step", Title: "Step", Goal: "Goal", Role: "tester", Status: "READY",
			MaestroGates: []string{"legacy-gate"}, MaestroSkills: []string{"testing"},
			AcceptanceCriteria: []string{"done"}, VerificationRequirements: []string{"true"},
		}}}},
	}
	frozen, _, err := freezePlanForExecution(n, plan)
	if err != nil {
		t.Fatalf("legacy gate should not be resolved as a Skill: %v", err)
	}
	pkg := frozen.Phases[0].Packages[0]
	if !reflect.DeepEqual(pkg.SkillIDs, []string{"testing"}) || !reflect.DeepEqual(pkg.MaestroSkills, []string{"testing"}) {
		t.Fatalf("unexpected canonical Skills: skill_ids=%v maestro_skills=%v", pkg.SkillIDs, pkg.MaestroSkills)
	}
	if !reflect.DeepEqual(pkg.MaestroGates, []string{"legacy-gate"}) {
		t.Fatalf("legacy gates should remain separate metadata: %v", pkg.MaestroGates)
	}
}

func TestValidateFlowExecutionContractRejectsContradictoryAssignment(t *testing.T) {
	plan := store.WorkPlan{ID: "plan", ProjectID: "project", Phases: []store.PlanPhase{{ID: "p", Packages: []store.WorkPackage{{ID: "A", AssignmentStrategy: "CREATE", AgentAllocation: "agent-1"}}}}}
	if err := validateFlowExecutionContract(plan); err == nil {
		t.Fatal("CREATE + existing Agent must be rejected")
	}
	plan.Phases[0].Packages[0].AssignmentStrategy = "EXISTING"
	plan.Phases[0].Packages[0].AgentAllocation = ""
	if err := validateFlowExecutionContract(plan); err == nil {
		t.Fatal("EXISTING without Agent must be rejected")
	}
	plan.Phases[0].Packages[0].AssignmentStrategy = "AUTO"
	plan.Phases[0].Packages[0].ResourcePolicy = "MANUAL"
	if err := validateFlowExecutionContract(plan); err == nil {
		t.Fatal("MANUAL resource policy without provider/profile must be rejected")
	}
}

func TestStartMissionRunAutonomousRejectsPinnedAgentWithoutWorktreeIsolation(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Strict admission", CanonicalPath: t.TempDir(), DefaultIsolation: "worktree"})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Unsafe pinned", Role: "implementer"})
	if err != nil {
		t.Fatal(err)
	}
	revision, err := st.AddRevision(agent.ID, (AgentConfig{Isolation: "project"}).ConfigJSON())
	if err != nil {
		t.Fatal(err)
	}
	agent.CurrentRevisionID = revision.ID
	if err := st.UpdateAgent(agent); err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Plan", "", []store.PlanPhase{{
		Title: "Phase", Packages: []store.WorkPackage{{
			Title: "Required worktree", Goal: "Run safely", Status: "READY", Role: "implementer",
			AssignmentStrategy: string(FlowAssignmentExisting), AgentAllocation: agent.ID,
			AcceptanceCriteria: []string{"done"},
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = n.StartMissionRun(context.Background(), plan.ID, "", runner.DefaultAutonomyContract(), true)
	if err == nil {
		t.Fatal("expected autonomous mission admission to reject pinned project checkout")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "worktree") {
		t.Fatalf("expected actionable worktree admission error, got %v", err)
	}
}

func TestStrictAdmissionAllowsAutoAllocationWithoutProjectWideWorktreeDefault(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Per-Agent isolation", CanonicalPath: t.TempDir(), DefaultIsolation: "project"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Auto isolated", "", []store.PlanPhase{{
		Title: "Phase", Packages: []store.WorkPackage{{
			Title: "Auto work", Goal: "Run safely", Status: "READY", Role: "implementer",
			AssignmentStrategy: string(FlowAssignmentAuto), AcceptanceCriteria: []string{"done"},
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := n.PreflightFlowStrict(context.Background(), plan.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	var isolation *FlowPreflightCheck
	for i := range report.Checks {
		if report.Checks[i].Key == "worktree_isolation" {
			isolation = &report.Checks[i]
			break
		}
	}
	if isolation == nil || isolation.Status != "PASS" {
		t.Fatalf("AUTO allocation should be worktree-safe without project-wide default: %+v", report.Checks)
	}
}

func TestNorthStar_PromptToFlowToPreflightCycle(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "NorthStarE2E", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	// 1. Decompose prompt into flow proposal
	proposal, err := n.DecomposePromptIntoFlowProposal(context.Background(), FlowDecompositionRequest{
		ProjectID:    project.ID,
		Goal:         "Implement responsive layout and run automated regression tests",
		SourcePrompt: "Ensure container queries adapt smoothly and unit tests pass",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(proposal.Flow.Steps) < 2 {
		t.Fatalf("expected structured multi-step flow, got %d steps", len(proposal.Flow.Steps))
	}

	// 2. Materialize Flow into durable WorkPlan
	plan := WorkPlanFromFlow(proposal.Flow)
	createdPlan, err := st.CreateWorkPlan(plan)
	if err != nil {
		t.Fatal(err)
	}

	// 3. Run Preflight check before execution
	report, err := n.PreflightFlow(context.Background(), createdPlan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Ready {
		t.Fatalf("expected preflight to pass for valid generated flow, got %+v", report)
	}
	if len(report.Checks) == 0 {
		t.Fatal("expected preflight checks in report")
	}
}
