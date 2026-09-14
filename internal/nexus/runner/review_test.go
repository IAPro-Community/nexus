package runner

import (
	"context"
	"testing"
)

func TestReviewFindingIdentityDeduplicatesAndResolves(t *testing.T) {
	one := NormalizeReviewFinding(ReviewFinding{Category: "test", File: "internal/x.go", Location: "10", Description: "missing assertion"})
	two := NormalizeReviewFinding(ReviewFinding{Category: "test", File: "internal/x.go", Location: "10", Description: "missing assertion"})
	if one.ID != two.ID {
		t.Fatalf("semantic identity changed: %s vs %s", one.ID, two.ID)
	}
	if got := DeduplicateReviewFindings([]ReviewFinding{one, two}); len(got) != 1 {
		t.Fatalf("duplicate finding was not collapsed: %+v", got)
	}
	history := MergeReviewFindings(nil, []ReviewFinding{one}, false, 1)
	if len(OpenReviewFindings(history)) != 1 || history[0].Status != FindingNew {
		t.Fatalf("initial finding ledger incorrect: %+v", history)
	}
	resolved := MergeReviewFindings(history, nil, true, 2)
	if len(resolved) != 1 || resolved[0].Status != FindingResolved || len(OpenReviewFindings(resolved)) != 0 {
		t.Fatalf("approved review did not resolve finding: %+v", resolved)
	}
	regressed := MergeReviewFindings(resolved, []ReviewFinding{one}, false, 3)
	if regressed[0].Status != FindingRegressed {
		t.Fatalf("reappearing finding should be REGRESSED: %+v", regressed)
	}
}

func TestMissionTeamPlannerBuilderReviewerVerifierPipeline(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: true}
	runner := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.RequireVerification = false
	contract.VerificationCommands = []string{"true"}
	contract.GlobalVerificationCommands = []string{"true"}
	run, err := runner.StartMissionRun(t.Context(), PlanSpec{
		ID: "mission-team", ProjectID: "project", Revision: 1, LeadAgentID: "lead",
		Packages: []PackageSpec{
			{ID: "planning", PhaseID: "planning", PhaseType: "PLANNING", Role: "architect", Goal: "plan", HandoffJSON: `{"mission_objective":"feature","next_expected_action":"build"}`},
			{ID: "implementation", PhaseID: "implementation", PhaseType: "IMPLEMENTATION", Role: "implementer", Goal: "build", Dependencies: []string{"planning"}, HandoffJSON: `{"mission_objective":"feature","next_expected_action":"review"}`},
			{ID: "review", PhaseID: "review", PhaseType: "REVIEW", Role: "reviewer", ReviewRequired: true, Goal: "review", Dependencies: []string{"implementation"}, HandoffJSON: `{"mission_objective":"feature","next_expected_action":"verify"}`},
			{ID: "verification", PhaseID: "verification", PhaseType: "VERIFICATION", Role: "qa-engineer", Goal: "verify", Dependencies: []string{"review"}, HandoffJSON: `{"mission_objective":"feature","next_expected_action":"done"}`},
		},
	}, t.TempDir(), contract, "lead")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40 && run.State != StateCompletedVerified; i++ {
		run, _, err = runner.ExecuteNextStep(t.Context(), run.ID)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	if run.State != StateCompletedVerified || run.LeadAgentID != "lead" {
		t.Fatalf("mission team did not complete with lead identity: state=%s lead=%s", run.State, run.LeadAgentID)
	}
	if len(run.PhaseRuns) != 4 || run.CurrentPhaseID != "verification" {
		t.Fatalf("phase lifecycle projection incorrect: %+v current=%s", run.PhaseRuns, run.CurrentPhaseID)
	}
	if len(run.PackageRuns[2].Verdicts) != 1 || run.PackageRuns[2].AssignedAgent == "" {
		t.Fatalf("review phase did not persist reviewer evidence: %+v", run.PackageRuns[2])
	}
	if run.PackageRuns[3].DispatchState != DispatchNone {
		t.Fatalf("verification phase must not dispatch a provider write turn: %+v", run.PackageRuns[3])
	}
	if exec.executed != 2 {
		t.Fatalf("planning/implementation should execute, review/verifier must not write: executed=%d", exec.executed)
	}
}

type remediationCycleExecutor struct {
	fakeExecutor
	reviews int
}

func (e *remediationCycleExecutor) Review(_ context.Context, _ *MissionRun, _ *PackageRun) (ReviewVerdict, error) {
	e.reviews++
	finding := func(file, description string) ReviewFinding {
		return ReviewFinding{Category: "quality", File: file, Location: "1", Description: description, RequiredAction: "fix"}
	}
	switch e.reviews {
	case 1:
		return ReviewVerdict{ReviewerAgentID: "reviewer", StructuredFindings: []ReviewFinding{finding("a.go", "first"), finding("b.go", "second"), finding("c.go", "third")}}, nil
	case 2:
		return ReviewVerdict{ReviewerAgentID: "reviewer", StructuredFindings: []ReviewFinding{finding("a.go", "first"), finding("d.go", "new")}}, nil
	default:
		return ReviewVerdict{Approved: true, ReviewerAgentID: "reviewer"}, nil
	}
}

func TestMissionTeamPlannerBuilderReviewerVerifierWithRemediationCycle(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &remediationCycleExecutor{fakeExecutor: fakeExecutor{reviewOK: true}}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.RequireVerification = false
	contract.VerificationCommands = []string{"true"}
	contract.GlobalVerificationCommands = []string{"true"}
	run, err := r.StartMissionRun(t.Context(), PlanSpec{
		ID: "mission-team-remediation", ProjectID: "project", Revision: 1, LeadAgentID: "lead",
		Packages: []PackageSpec{
			{ID: "planning", PhaseID: "planning", PhaseType: "PLANNING", Role: "architect", Goal: "plan", HandoffJSON: `{"mission_objective":"feature","next_expected_action":"build"}`},
			{ID: "implementation", PhaseID: "implementation", PhaseType: "IMPLEMENTATION", Role: "implementer", Goal: "build", Dependencies: []string{"planning"}, HandoffJSON: `{"mission_objective":"feature","next_expected_action":"review"}`},
			{ID: "review", PhaseID: "review", PhaseType: "REVIEW", Role: "reviewer", ReviewRequired: true, Goal: "review", Dependencies: []string{"implementation"}, HandoffJSON: `{"mission_objective":"feature","next_expected_action":"verify"}`},
			{ID: "verification", PhaseID: "verification", PhaseType: "VERIFICATION", Role: "qa-engineer", Goal: "verify", Dependencies: []string{"review"}, HandoffJSON: `{"mission_objective":"feature","next_expected_action":"done"}`},
		},
	}, t.TempDir(), contract, "lead")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 80 && run.State != StateCompletedVerified; i++ {
		run, _, err = r.ExecuteNextStep(t.Context(), run.ID)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	if run.State != StateCompletedVerified {
		t.Fatalf("remediation cycle did not complete: %s", run.State)
	}
	findings := run.PackageRuns[2].ReviewFindings
	if len(findings) != 4 || len(run.PackageRuns[2].Verdicts) != 3 {
		t.Fatalf("expected three review passes and four ledger entries, findings=%+v verdicts=%d", findings, len(run.PackageRuns[2].Verdicts))
	}
	if len(OpenReviewFindings(findings)) != 0 {
		t.Fatalf("approved review must resolve all findings: %+v", findings)
	}
	if run.PackageRuns[3].DispatchState != DispatchNone {
		t.Fatalf("verifier must remain read-only: %+v", run.PackageRuns[3])
	}
}
