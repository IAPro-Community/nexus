package nexus

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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

// challengeDispatchExecutor is a deterministic PackageExecutor: it writes the
// Agent marker into the workspace it was allocated and approves its own review.
// No provider is contacted; isolation comes from each run owning its own tree.
type challengeDispatchExecutor struct {
	mu         sync.Mutex
	workspaces map[string]string
}

func (*challengeDispatchExecutor) Allocate(_ context.Context, run *runner.MissionRun, _ *runner.PackageRun) (runner.AllocationResult, error) {
	return runner.AllocationResult{AgentID: run.LeadAgentID, Workspace: run.Workspace}, nil
}

func (*challengeDispatchExecutor) Compile(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.PromptArtifact, error) {
	return runner.PromptArtifact{VersionID: "challenge-dispatch-prompt-v1", Content: "implement the shared specification"}, nil
}

func (e *challengeDispatchExecutor) Execute(_ context.Context, run *runner.MissionRun, pkg *runner.PackageRun, _ string) (runner.ExecutionResult, error) {
	e.mu.Lock()
	if e.workspaces == nil {
		e.workspaces = map[string]string{}
	}
	e.workspaces[run.ID] = pkg.Workspace
	e.mu.Unlock()
	marker := filepath.Join(pkg.Workspace, "agent-marker.txt")
	if err := os.WriteFile(marker, []byte(run.LeadAgentID+"\n"), 0o600); err != nil {
		return runner.ExecutionResult{}, err
	}
	return runner.ExecutionResult{RuntimeID: "runtime-" + run.LeadAgentID}, nil
}

func (*challengeDispatchExecutor) Review(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.ReviewVerdict, error) {
	return runner.ReviewVerdict{Approved: true, ReviewerAgentID: "reviewer-dispatch"}, nil
}

func dispatchGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, strings.TrimSpace(string(out)))
	}
}

func initDispatchWorktree(t *testing.T, workspace string) {
	t.Helper()
	dispatchGit(t, workspace, "init", "-q")
	dispatchGit(t, workspace, "config", "user.email", "parallel-challenge@example.invalid")
	dispatchGit(t, workspace, "config", "user.name", "Parallel Challenge Fixture")
	if err := os.WriteFile(filepath.Join(workspace, "README.md"), []byte("# isolated worktree\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dispatchGit(t, workspace, "add", "README.md")
	dispatchGit(t, workspace, "commit", "-qm", "baseline")
}

// TestParallelChallengeDispatchesOneSpecificationToIsolatedAgents proves the
// dispatch half of the capability: the same specification is really started as
// two MissionRuns for two different Agents, each executing in its own isolated
// Git worktree, and only the completed runs are compared.
func TestParallelChallengeDispatchesOneSpecificationToIsolatedAgents(t *testing.T) {
	repo := runner.NewMemoryRunRepository()
	r := runner.NewMissionRunner(repo, &challengeDispatchExecutor{})
	contract := runner.DefaultAutonomyContract()
	contract.VerificationCommands = []string{"exit 0"}
	spec := runner.PlanSpec{ID: "plan-dispatch", ProjectID: "proj-dispatch", Revision: 1, Packages: []runner.PackageSpec{{
		ID: "pkg-shared", Title: "Implement the shared specification",
		Goal: "Implement the shared specification", Priority: "HIGH",
		AcceptanceCriteria: []string{"each Agent finishes in its own isolated worktree"},
	}}}
	workspaces := []string{t.TempDir(), t.TempDir()}
	agents := []string{"agent-alpha", "agent-beta"}
	runIDs := make([]string, 0, len(workspaces))
	for index, workspace := range workspaces {
		initDispatchWorktree(t, workspace)
		run, err := r.StartMissionRun(context.Background(), spec, workspace, contract, agents[index])
		if err != nil {
			t.Fatalf("dispatch %s: %v", agents[index], err)
		}
		if run.LeadAgentID != agents[index] || run.Workspace != workspace {
			t.Fatalf("run did not bind Agent %s to its isolated workspace: %+v", agents[index], run)
		}
		for step := 0; step < 50 && run.State != runner.StateCompletedVerified; step++ {
			run, _, err = r.ExecuteNextStep(context.Background(), run.ID)
			if err != nil {
				t.Fatalf("step %d for %s: %v", step, agents[index], err)
			}
		}
		if run.State != runner.StateCompletedVerified {
			t.Fatalf("%s ended in %s instead of COMPLETED_VERIFIED", agents[index], run.State)
		}
		if len(run.PackageRuns) != 1 || len(run.PackageRuns[0].Verifications) == 0 {
			t.Fatalf("%s completed without verification evidence: %+v", agents[index], run.PackageRuns)
		}
		if run.PackageRuns[0].Workspace != workspace {
			t.Fatalf("%s executed outside its isolated worktree: %q", agents[index], run.PackageRuns[0].Workspace)
		}
		marker, err := os.ReadFile(filepath.Join(workspace, "agent-marker.txt"))
		if err != nil || strings.TrimSpace(string(marker)) != agents[index] {
			t.Fatalf("%s did not write its marker into its own worktree: %q (%v)", agents[index], string(marker), err)
		}
		runIDs = append(runIDs, run.ID)
	}
	if runIDs[0] == runIDs[1] {
		t.Fatal("the specification must be dispatched to two distinct runs")
	}
	runs, err := r.ListRuns(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	dispatched := make([]runner.MissionRun, 0, len(runs))
	worktrees := map[string]bool{}
	for _, run := range runs {
		dispatched = append(dispatched, *run)
	}
	participants, err := MissionChallengeParticipants(dispatched)
	if err != nil {
		t.Fatalf("MissionChallengeParticipants over real runs: %v", err)
	}
	for _, participant := range participants {
		worktrees[participant.WorktreePath] = true
	}
	if len(worktrees) != 2 || !worktrees[workspaces[0]] || !worktrees[workspaces[1]] {
		t.Fatalf("participants must map to the two isolated real worktrees: %v", worktrees)
	}
	n := &Nexus{runner: r}
	result, err := n.EvaluatePlanChallenge(context.Background(), spec.ID, []AgentCompareCriterion{{Name: "tests", Kind: AgentCompareCriterionTests, Weight: 1}})
	if err != nil {
		t.Fatalf("EvaluatePlanChallenge over dispatched runs: %v", err)
	}
	if result.Participants != 2 || result.AutoMergeAllowed {
		t.Fatalf("participants=%d auto_merge=%v", result.Participants, result.AutoMergeAllowed)
	}
	if len(result.Ranking) != 2 || result.Ranking[0] != "agent-alpha" || result.Ranking[1] != "agent-beta" {
		t.Fatalf("ranking=%v want the deterministic tie-break agent-alpha, agent-beta", result.Ranking)
	}
}
