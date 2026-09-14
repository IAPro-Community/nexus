package nexus

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func canonicalFlowPlanFixture() store.WorkPlan {
	created := time.Unix(100, 0).UTC()
	updated := time.Unix(200, 0).UTC()
	return store.WorkPlan{
		ID: "plan-flow", ProjectID: "project-1", MissionID: "mission-legacy",
		Title: "Ship feature", Description: "Implement and verify", Status: "DRAFT", CurrentRevision: 4,
		StructuredFacts: map[string]string{"existing.fact": "keep", FlowPolicyFactKey: string(FlowAutonomous)},
		CreatedAt:       created, UpdatedAt: updated,
		Phases: []store.PlanPhase{
			{ID: "phase-build", Title: "Build", Description: "Implementation", Order: 1, Packages: []store.WorkPackage{
				{ID: "A", Title: "Plan", Goal: "Plan work", Priority: "NORMAL", Status: "READY", Role: "architect", AgentAllocation: "agent-a", AssignmentStrategy: string(FlowAssignmentExisting), ResourcePolicy: "BALANCED", Provider: "claude", Profile: "default", MaestroSkills: []string{"planning"}, RelevantPaths: []string{"internal/nexus"}, AcceptanceCriteria: []string{"plan exists"}, VerificationRequirements: []string{"test -f PLAN.md"}},
				{ID: "B", Title: "Backend", Goal: "Implement backend", Priority: "HIGH", Status: "READY", Dependencies: []string{"A"}, ParallelGroup: "impl", Role: "implementer", AssignmentStrategy: string(FlowAssignmentCreate), ResourcePolicy: "PRESERVE_QUOTA", MaestroGates: []string{"legacy-gate"}, MaestroSkills: []string{"backend"}, RelevantPaths: []string{"internal"}, AcceptanceCriteria: []string{"backend passes"}, VerificationRequirements: []string{"go test ./internal/..."}},
				{ID: "C", Title: "Frontend", Goal: "Implement frontend", Priority: "HIGH", Status: "READY", Dependencies: []string{"A"}, ParallelGroup: "impl", Role: "implementer", AssignmentStrategy: string(FlowAssignmentAuto), ResourcePolicy: "BALANCED", Provider: "codex", Profile: "fast", RelevantPaths: []string{"web/src"}, AcceptanceCriteria: []string{"frontend passes"}, VerificationRequirements: []string{"npm test"}},
			}},
			{ID: "phase-verify", Title: "Verify", Description: "Integration", Order: 2, Packages: []store.WorkPackage{
				{ID: "D", Title: "Verify", Goal: "Verify all", Priority: "CRITICAL", Status: "READY", Dependencies: []string{"B", "C"}, Role: "tester", AssignmentStrategy: string(FlowAssignmentAuto), MaestroSkills: []string{"verification"}, AcceptanceCriteria: []string{"all verified"}, VerificationRequirements: []string{"go test ./...", "npm test"}},
			}},
			{ID: "phase-empty", Title: "Reserved", Description: "Keep empty phase metadata", Order: 3},
		},
	}
}

func TestFlowFacadeRoundTripPreservesWorkPlanContracts(t *testing.T) {
	plan := canonicalFlowPlanFixture()
	flow := FlowFromWorkPlan(plan)
	if flow.Policy != FlowAutonomous {
		t.Fatalf("policy=%q", flow.Policy)
	}
	if len(flow.Steps) != 4 {
		t.Fatalf("steps=%d", len(flow.Steps))
	}
	if got := flow.Steps[1].AssignmentStrategy; got != FlowAssignmentCreate {
		t.Fatalf("B assignment=%q", got)
	}
	if got := flow.Steps[2].Provider; got != "codex" {
		t.Fatalf("C provider=%q", got)
	}
	if got := flow.Steps[3].VerificationRequirements; !reflect.DeepEqual(got, []string{"go test ./...", "npm test"}) {
		t.Fatalf("D verification=%v", got)
	}

	roundTrip := WorkPlanFromFlow(flow)
	if !reflect.DeepEqual(roundTrip, plan) {
		t.Fatalf("round trip changed plan\nwant=%#v\n got=%#v", plan, roundTrip)
	}
}

func TestFlowFacadeLegacyAllocationMapsToExisting(t *testing.T) {
	plan := store.WorkPlan{ID: "legacy", ProjectID: "p", CurrentRevision: 1, Phases: []store.PlanPhase{{ID: "phase", Packages: []store.WorkPackage{{ID: "A", Title: "A", AgentAllocation: "agent-1"}}}}}
	flow := FlowFromWorkPlan(plan)
	if flow.Policy != FlowGuided {
		t.Fatalf("default policy=%q", flow.Policy)
	}
	if flow.Steps[0].AssignmentStrategy != FlowAssignmentExisting {
		t.Fatalf("legacy allocation should map EXISTING: %#v", flow.Steps[0])
	}
	if flow.Steps[0].AgentID != "agent-1" {
		t.Fatalf("agent lost: %#v", flow.Steps[0])
	}
}

func TestFlowFacadePreservesCanonicalSkillIDs(t *testing.T) {
	plan := store.WorkPlan{ID: "canonical-skills", ProjectID: "p", CurrentRevision: 1, Phases: []store.PlanPhase{{ID: "phase", Packages: []store.WorkPackage{{ID: "A", Title: "A", SkillIDs: []string{"coding", "testing"}}}}}}
	flow := FlowFromWorkPlan(plan)
	if !reflect.DeepEqual(flow.Steps[0].SkillIDs, []string{"coding", "testing"}) {
		t.Fatalf("canonical SkillIDs lost in Flow facade: %#v", flow.Steps[0])
	}
	roundTrip := WorkPlanFromFlow(flow)
	if !reflect.DeepEqual(roundTrip.Phases[0].Packages[0].SkillIDs, plan.Phases[0].Packages[0].SkillIDs) {
		t.Fatalf("canonical SkillIDs lost in WorkPlan round trip: %#v", roundTrip.Phases[0].Packages[0])
	}
}

func TestFlowDAGDeterministicOrderWavesAndDependents(t *testing.T) {
	flow := FlowFromWorkPlan(canonicalFlowPlanFixture())
	if err := ValidateFlowDAG(flow); err != nil {
		t.Fatal(err)
	}
	order, err := TopologicalOrder(flow)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(order, []string{"A", "B", "C", "D"}) {
		t.Fatalf("order=%v", order)
	}
	waves, err := ExecutionWaves(flow)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(waves, [][]string{{"A"}, {"B", "C"}, {"D"}}) {
		t.Fatalf("waves=%v", waves)
	}
	if got := DependentsOf(flow, "A"); !reflect.DeepEqual(got, []string{"B", "C"}) {
		t.Fatalf("dependents=%v", got)
	}
}

func TestFlowDAGRejectsUnknownDependencyDuplicateAndCycle(t *testing.T) {
	base := FlowFromWorkPlan(canonicalFlowPlanFixture())

	unknown := base
	unknown.Steps = append([]FlowStep(nil), base.Steps...)
	unknown.Steps[1].Dependencies = []string{"missing"}
	if err := ValidateFlowDAG(unknown); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("unknown dependency err=%v", err)
	}

	duplicate := base
	duplicate.Steps = append([]FlowStep(nil), base.Steps...)
	duplicate.Steps[1].ID = "A"
	if err := ValidateFlowDAG(duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate err=%v", err)
	}

	cycle := base
	cycle.Steps = append([]FlowStep(nil), base.Steps...)
	cycle.Steps[0].Dependencies = []string{"D"}
	if err := ValidateFlowDAG(cycle); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("cycle err=%v", err)
	}
}

func TestFlowDAGOrderIsStableAcrossRuns(t *testing.T) {
	flow := FlowFromWorkPlan(canonicalFlowPlanFixture())
	first, err := TopologicalOrder(flow)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30; i++ {
		got, err := TopologicalOrder(flow)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, first) {
			t.Fatalf("nondeterministic: %v vs %v", first, got)
		}
	}
}

func TestFlowDAGRejectsStepWithUnknownPhase(t *testing.T) {
	flow := FlowFromWorkPlan(canonicalFlowPlanFixture())
	flow.Steps[1].PhaseID = "missing-phase"
	if err := ValidateFlowDAG(flow); err == nil || !strings.Contains(err.Error(), "unknown phase") {
		t.Fatalf("unknown phase err=%v", err)
	}
}
