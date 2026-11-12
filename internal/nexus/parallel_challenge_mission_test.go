package nexus

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

type challengeRepo struct {
	runs []*runner.MissionRun
}

func (r *challengeRepo) SaveRun(_ context.Context, run *runner.MissionRun) error {
	r.runs = append(r.runs, run)
	return nil
}

func (r *challengeRepo) GetRun(_ context.Context, id string) (*runner.MissionRun, error) {
	for _, run := range r.runs {
		if run.ID == id {
			return run, nil
		}
	}
	return nil, runner.ErrRunNotFound
}

func (r *challengeRepo) ListRuns(_ context.Context) ([]*runner.MissionRun, error) {
	return r.runs, nil
}

func (r *challengeRepo) AcquireLease(_ context.Context, _, _ string, _ time.Duration) (*runner.MissionRun, error) {
	return nil, nil
}

func (r *challengeRepo) RenewLease(_ context.Context, _, _, _ string, _ time.Duration) error {
	return nil
}

func (r *challengeRepo) ReleaseLease(_ context.Context, _, _, _ string) error {
	return nil
}

func challengeRun(id, planID, lead, workspace string, state runner.State, passed, failed int) runner.MissionRun {
	receipts := make([]*runner.WorkReceipt, 0, passed+failed)
	for i := 0; i < passed; i++ {
		receipts = append(receipts, &runner.WorkReceipt{ID: id + "-t-pass", RunID: id, StepID: "s", Tests: []runner.VerificationResult{{Passed: true}}})
	}
	for i := 0; i < failed; i++ {
		receipts = append(receipts, &runner.WorkReceipt{ID: id + "-t-fail", RunID: id, StepID: "s", Tests: []runner.VerificationResult{{Passed: false}}})
	}
	run := runner.MissionRun{
		ID:          id,
		PlanID:      planID,
		LeadAgentID: lead,
		Workspace:   workspace,
		State:       state,
	}
	for _, receipt := range receipts {
		run.PackageRuns = append(run.PackageRuns, runner.PackageRun{ID: receipt.ID, WorkReceipt: receipt})
	}
	return run
}

func TestMissionChallengeParticipantsMapsCompletedRuns(t *testing.T) {
	runs := []runner.MissionRun{
		challengeRun("run-a", "plan-1", "lead-a", "/wt/a", runner.StateCompleted, 3, 1),
		challengeRun("run-b", "plan-1", "lead-b", "/wt/b", runner.StateCompleted, 2, 0),
	}
	participants, err := MissionChallengeParticipants(runs)
	if err != nil {
		t.Fatalf("MissionChallengeParticipants: %v", err)
	}
	if len(participants) != 2 {
		t.Fatalf("participants=%d want 2", len(participants))
	}
	byLabel := map[string]ChallengeParticipant{}
	for _, p := range participants {
		byLabel[p.Label] = p
	}
	a := byLabel["lead-a"]
	if a.TestsPassed != 3 || a.TestsFailed != 1 {
		t.Fatalf("lead-a evidence = %+v", a)
	}
	b := byLabel["lead-b"]
	if b.TestsPassed != 2 || b.TestsFailed != 0 {
		t.Fatalf("lead-b evidence = %+v", b)
	}
	// Cost evidence does not exist on runs: UNKNOWN must stay UNKNOWN.
	for label, p := range byLabel {
		if p.CostKnown {
			t.Fatalf("%s must not fabricate cost evidence", label)
		}
	}
}

func TestMissionChallengeParticipantsRefusesIncompleteRuns(t *testing.T) {
	runs := []runner.MissionRun{
		challengeRun("run-a", "plan-1", "lead-a", "/wt/a", runner.StateCompleted, 1, 0),
		challengeRun("run-b", "plan-1", "lead-b", "/wt/b", runner.StateExecuting, 1, 0),
	}
	if _, err := MissionChallengeParticipants(runs); err == nil {
		t.Fatal("running runs must be refused: no decisive evidence")
	}
}

func TestMissionChallengeParticipantsRefusesCrossPlanAndDuplicates(t *testing.T) {
	if _, err := MissionChallengeParticipants([]runner.MissionRun{
		challengeRun("run-a", "plan-1", "lead-a", "/wt/a", runner.StateCompleted, 1, 0),
		challengeRun("run-b", "plan-2", "lead-b", "/wt/b", runner.StateCompleted, 1, 0),
	}); err == nil {
		t.Fatal("cross-plan runs must be refused")
	}
	// Same workspace twice: isolation cannot be proven.
	if _, err := MissionChallengeParticipants([]runner.MissionRun{
		challengeRun("run-a", "plan-1", "lead-a", "/wt/shared", runner.StateCompleted, 1, 0),
		challengeRun("run-b", "plan-1", "lead-b", "/wt/shared", runner.StateCompleted, 1, 0),
	}); err == nil {
		t.Fatal("shared worktree must be refused")
	}
}

func TestMissionChallengeParticipantsUniquesDuplicateLeadLabels(t *testing.T) {
	runs := []runner.MissionRun{
		challengeRun("run-a", "plan-1", "lead-a", "/wt/a", runner.StateCompleted, 1, 0),
		challengeRun("run-b", "plan-1", "lead-a", "/wt/b", runner.StateCompleted, 1, 0),
	}
	participants, err := MissionChallengeParticipants(runs)
	if err != nil {
		t.Fatalf("MissionChallengeParticipants: %v", err)
	}
	seen := map[string]bool{}
	for _, p := range participants {
		if seen[p.Label] {
			t.Fatalf("duplicate label %q", p.Label)
		}
		seen[p.Label] = true
	}
	if !seen["lead-a@run-b"] {
		t.Fatalf("expected suffixed label for the second run of the same lead; got %v", seen)
	}
}

func TestEvaluatePlanChallengeUsesRunnerRuns(t *testing.T) {
	repo := &challengeRepo{runs: []*runner.MissionRun{}}
	a := challengeRun("run-a", "plan-9", "lead-a", "/wt/a", runner.StateCompleted, 4, 0)
	b := challengeRun("run-b", "plan-9", "lead-b", "/wt/b", runner.StateCompleted, 2, 2)
	repo.runs = append(repo.runs, &a, &b)
	n := &Nexus{runner: runner.NewMissionRunner(repo, nil)}

	result, err := n.EvaluatePlanChallenge(context.Background(), "plan-9", []AgentCompareCriterion{{Name: "tests", Kind: "tests", Weight: 1}})
	if err != nil {
		t.Fatalf("EvaluatePlanChallenge: %v", err)
	}
	if result.Participants != 2 {
		t.Fatalf("participants=%d want 2", result.Participants)
	}
	if result.AutoMergeAllowed {
		t.Fatal("challenge must never auto-merge")
	}
	if len(result.Ranking) == 0 || result.Ranking[0] != "lead-a" {
		t.Fatalf("ranking=%v want lead-a first (4 passed vs 2/2)", result.Ranking)
	}
	if !strings.HasPrefix(result.Specification, "plan:") {
		t.Fatalf("specification=%q", result.Specification)
	}
}

func TestEvaluatePlanChallengeRequiresEnoughRuns(t *testing.T) {
	repo := &challengeRepo{}
	one := challengeRun("run-a", "plan-9", "lead-a", "/wt/a", runner.StateCompleted, 1, 0)
	repo.runs = append(repo.runs, &one)
	n := &Nexus{runner: runner.NewMissionRunner(repo, nil)}
	if _, err := n.EvaluatePlanChallenge(context.Background(), "plan-9", []AgentCompareCriterion{{Name: "tests", Kind: "tests", Weight: 1}}); err == nil {
		t.Fatal("single completed run must be refused")
	}
}
