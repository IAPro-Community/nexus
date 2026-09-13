package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type globalRepairExecutor struct {
	fakeExecutor
	executions int
}

func (e *globalRepairExecutor) Execute(ctx context.Context, run *MissionRun, pkg *PackageRun, prompt string) (ExecutionResult, error) {
	e.executions++
	if e.executions > 1 {
		if err := os.WriteFile(filepath.Join(run.Workspace, "global-ready"), []byte("verified\n"), 0o600); err != nil {
			return ExecutionResult{}, err
		}
	}
	return e.fakeExecutor.Execute(ctx, run, pkg, prompt)
}

func TestMissionRunnerGlobalDefinitionOfDoneReopensAfterFailure(t *testing.T) {
	workspace := t.TempDir()
	exec := &globalRepairExecutor{fakeExecutor: fakeExecutor{reviewOK: true}}
	r := NewMissionRunner(NewMemoryRunRepository(), exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{"true"}
	contract.GlobalVerificationCommands = []string{"test -f global-ready"}
	contract.MaxNoProgress = 2
	run, err := r.StartMissionRun(context.Background(), PlanSpec{ID: "global", ProjectID: "p", Revision: 1, Packages: []PackageSpec{{ID: "step", Title: "Step", Goal: "Goal"}}}, workspace, contract, "")
	if err != nil {
		t.Fatal(err)
	}
	var globalFailure bool
	for i := 0; i < 30 && run.State != StateCompletedVerified; i++ {
		updated, _, stepErr := r.ExecuteNextStep(context.Background(), run.ID)
		if updated != nil {
			run = updated
		}
		if stepErr != nil && len(run.GlobalVerifications) > 0 {
			globalFailure = true
		}
	}
	if !globalFailure {
		t.Fatal("expected global verification failure evidence")
	}
	if run.State != StateCompletedVerified {
		t.Fatalf("expected convergence after global repair, got %s", run.State)
	}
	if len(run.GlobalVerifications) < 2 {
		t.Fatalf("expected failed and passing global verification evidence, got %+v", run.GlobalVerifications)
	}
	if run.GlobalVerifications[0].Passed || !run.GlobalVerifications[len(run.GlobalVerifications)-1].Passed {
		t.Fatalf("expected global verification to fail then pass: %+v", run.GlobalVerifications)
	}
}

func TestAutonomousMissionCannotCompleteWithoutGlobalDefinitionOfDone(t *testing.T) {
	r := NewMissionRunner(NewMemoryRunRepository(), &fakeExecutor{reviewOK: true})
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{"true"}
	run, err := r.StartMissionRun(context.Background(), PlanSpec{
		ID:         "autonomous-no-global-dod",
		ProjectID:  "p",
		Revision:   1,
		Autonomous: true,
		Packages:   []PackageSpec{{ID: "step", Title: "Step", Goal: "Goal"}},
	}, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20 && !IsTerminalState(run.State); i++ {
		updated, _, _ := r.ExecuteNextStep(context.Background(), run.ID)
		if updated != nil {
			run = updated
		}
	}
	if run.State != StateFailedVerification {
		t.Fatalf("autonomous run state=%s, want FAILED_VERIFICATION", run.State)
	}
}

func TestResolveIntegratedVerificationWorkspaceUsesAgentWorktree(t *testing.T) {
	canonical := t.TempDir()
	worktree := t.TempDir()
	run := &MissionRun{
		Workspace: canonical,
		PackageRuns: []PackageRun{{
			PackageID: "step",
			State:     StateVerified,
			Workspace: worktree,
		}},
	}
	got, err := resolveIntegratedVerificationWorkspace(run)
	if err != nil {
		t.Fatal(err)
	}
	if got != worktree {
		t.Fatalf("workspace=%q, want agent worktree %q", got, worktree)
	}
}

func TestResolveIntegratedVerificationWorkspaceFailsWithoutMerge(t *testing.T) {
	canonical := t.TempDir()
	run := &MissionRun{
		Workspace: canonical,
		PackageRuns: []PackageRun{
			{PackageID: "a", State: StateVerified, Workspace: t.TempDir()},
			{PackageID: "b", State: StateVerified, Workspace: t.TempDir()},
		},
	}
	_, err := resolveIntegratedVerificationWorkspace(run)
	if err == nil || !strings.Contains(err.Error(), "integrated tree") {
		t.Fatalf("error=%v, want integrated tree failure", err)
	}
}

func TestMissionRunnerGlobalDefinitionVerifiesWorktreeNotCanonicalBaseline(t *testing.T) {
	canonical := t.TempDir()
	worktree := t.TempDir()
	if err := os.WriteFile(filepath.Join(worktree, "global-ready"), []byte("verified\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	exec := &worktreeAllocatorExecutor{
		fakeExecutor: fakeExecutor{reviewOK: true},
		workspace:    worktree,
	}
	r := NewMissionRunner(NewMemoryRunRepository(), exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{"true"}
	contract.GlobalVerificationCommands = []string{"test -f global-ready"}
	run, err := r.StartMissionRun(context.Background(), PlanSpec{ID: "wt", ProjectID: "p", Revision: 1, Packages: []PackageSpec{{ID: "step", Title: "Step", Goal: "Goal"}}}, canonical, contract, "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20 && run.State != StateCompletedVerified && !IsTerminalState(run.State); i++ {
		updated, _, _ := r.ExecuteNextStep(context.Background(), run.ID)
		if updated != nil {
			run = updated
		}
	}
	if run.State != StateCompletedVerified {
		t.Fatalf("expected COMPLETED_VERIFIED from worktree DoD, got %s (%s)", run.State, run.PausedReason)
	}
	if _, err := os.Stat(filepath.Join(canonical, "global-ready")); !os.IsNotExist(err) {
		t.Fatalf("canonical baseline unexpectedly contains global-ready: %v", err)
	}
}

type worktreeAllocatorExecutor struct {
	fakeExecutor
	workspace string
}

func (e *worktreeAllocatorExecutor) Allocate(ctx context.Context, run *MissionRun, pkg *PackageRun) (AllocationResult, error) {
	return AllocationResult{AgentID: "agt-wt", Workspace: e.workspace}, nil
}
