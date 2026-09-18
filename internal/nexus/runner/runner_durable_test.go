package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type fakeExecutor struct {
	mu        sync.Mutex
	allocated int
	compiled  int
	executed  int
	reviewed  int
	reviewOK  bool
}

func (f *fakeExecutor) Allocate(_ context.Context, run *MissionRun, _ *PackageRun) (AllocationResult, error) {
	f.mu.Lock()
	f.allocated++
	f.mu.Unlock()
	return AllocationResult{AgentID: "agt-real", Workspace: run.Workspace}, nil
}
func (f *fakeExecutor) Compile(context.Context, *MissionRun, *PackageRun) (PromptArtifact, error) {
	f.mu.Lock()
	f.compiled++
	f.mu.Unlock()
	return PromptArtifact{VersionID: "prompt-v1", Content: "implement package"}, nil
}
func (f *fakeExecutor) Execute(context.Context, *MissionRun, *PackageRun, string) (ExecutionResult, error) {
	f.mu.Lock()
	f.executed++
	f.mu.Unlock()
	return ExecutionResult{RuntimeID: "rt-real"}, nil
}
func (f *fakeExecutor) Review(context.Context, *MissionRun, *PackageRun) (ReviewVerdict, error) {
	f.mu.Lock()
	f.reviewed++
	f.mu.Unlock()
	return ReviewVerdict{Approved: f.reviewOK, ReviewerAgentID: "agt-reviewer", Findings: []string{"review evidence"}}, nil
}

func (f *fakeExecutor) executedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.executed
}

func durablePlan() PlanSpec {
	return PlanSpec{ID: "plan-1", ProjectID: "proj-1", Revision: 3, Packages: []PackageSpec{
		{ID: "pkg-a", PhaseID: "phase-1", Title: "A", Goal: "A", Priority: "HIGH", AcceptanceCriteria: []string{"A verified"}},
		{ID: "pkg-b", PhaseID: "phase-1", Title: "B", Goal: "B", Priority: "HIGH", Dependencies: []string{"pkg-a"}, AcceptanceCriteria: []string{"B verified"}},
	}}
}

func TestMissionRunnerRequiresRealExecutorAndRespectsDependencies(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: true}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{"true"}

	plan := durablePlan()
	plan.ExecutionSnapshotID = "snap_immutable"
	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	if run.ExecutionSnapshotID != "snap_immutable" {
		t.Fatalf("execution snapshot binding lost: %q", run.ExecutionSnapshotID)
	}
	if run.PackageRuns[0].State != StateReady {
		t.Fatalf("root package should be ready: %s", run.PackageRuns[0].State)
	}
	if run.PackageRuns[1].State != StatePending {
		t.Fatalf("dependent package must remain pending: %s", run.PackageRuns[1].State)
	}

	for i := 0; i < 20; i++ {
		updated, done, stepErr := r.ExecuteNextStep(context.Background(), run.ID)
		if stepErr != nil {
			t.Fatalf("step %d: %v", i, stepErr)
		}
		run = updated
		if done {
			break
		}
	}
	if run.State != StateCompletedVerified {
		t.Fatalf("expected completed verified, got %s", run.State)
	}
	if exec.allocated != 2 || exec.compiled != 2 || exec.executed != 2 || exec.reviewed != 2 {
		t.Fatalf("expected real executor lifecycle for both packages, got a=%d c=%d e=%d r=%d", exec.allocated, exec.compiled, exec.executed, exec.reviewed)
	}
	if run.PackageRuns[0].AssignedRuntime != "rt-real" || run.PackageRuns[0].PromptVersionID != "prompt-v1" {
		t.Fatalf("execution evidence missing: %+v", run.PackageRuns[0])
	}
}

func TestMissionRunnerNeverAutoApprovesWithoutReviewerEvidence(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: false}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.MaxRetries = 1
	contract.VerificationCommands = []string{"true"}

	run, err := r.StartMissionRun(context.Background(), durablePlan(), t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 12; i++ {
		updated, _, stepErr := r.ExecuteNextStep(context.Background(), run.ID)
		if updated != nil {
			run = updated
		}
		if stepErr != nil {
			break
		}
	}
	if run.PackageRuns[0].State != StateFailed {
		t.Fatalf("review rejection must fail bounded package, got %s", run.PackageRuns[0].State)
	}
	if len(run.PackageRuns[0].Verdicts) == 0 || run.PackageRuns[0].Verdicts[0].ReviewerAgentID != "agt-reviewer" {
		t.Fatalf("expected concrete reviewer evidence, got %+v", run.PackageRuns[0].Verdicts)
	}
	for _, verdict := range run.PackageRuns[0].Verdicts {
		if verdict.ReviewerAgentID == "reviewer-auto" {
			t.Fatal("fabricated reviewer-auto is forbidden")
		}
	}
}

func TestMissionRunnerPersistsAcrossRunnerRecreation(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: true}
	r1 := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{"true"}
	run, err := r1.StartMissionRun(context.Background(), durablePlan(), t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := r1.ExecuteNextStep(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	}

	r2 := NewMissionRunner(repo, exec)
	restored, err := r2.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.ID != run.ID {
		t.Fatalf("restored wrong run: %s", restored.ID)
	}
	if restored.PackageRuns[0].State == StateReady {
		t.Fatal("restored state did not persist progress")
	}
}

func TestMissionRunnerEnforcesTotalIterationBudget(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: true}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.MaxTotalIterations = 1
	contract.VerificationCommands = []string{"true"}
	run, err := r.StartMissionRun(context.Background(), durablePlan(), t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.ExecuteNextStep(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	}
	updated, _, err := r.ExecuteNextStep(context.Background(), run.ID)
	if err == nil {
		t.Fatal("expected iteration budget error")
	}
	if updated.State != StateFailedBudgetExceeded {
		t.Fatalf("expected budget failure, got %s (%v)", updated.State, err)
	}
	if got := fmt.Sprint(err); got == "" {
		t.Fatal("expected explicit error")
	}
}

func TestMissionRunnerPauseResumeAndCancelAreDurable(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: true}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{"true"}
	run, err := r.StartMissionRun(context.Background(), durablePlan(), t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}

	paused, err := r.PauseRun(context.Background(), run.ID, "manual takeover")
	if err != nil {
		t.Fatal(err)
	}
	if paused.State != StatePaused || paused.PausedReason != "manual takeover" {
		t.Fatalf("pause not persisted: %+v", paused)
	}
	if _, _, err := r.ExecuteNextStep(context.Background(), run.ID); err == nil {
		t.Fatal("paused run must not execute")
	}

	resumed, err := r.ResumeRun(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.State != StateExecuting || resumed.PausedReason != "" {
		t.Fatalf("resume not persisted: %+v", resumed)
	}

	canceled, err := r.CancelRun(context.Background(), run.ID, "user requested")
	if err != nil {
		t.Fatal(err)
	}
	if canceled.State != StateCanceledByUser || canceled.PausedReason != "user requested" {
		t.Fatalf("cancel not persisted: %+v", canceled)
	}
}

func TestMissionRunnerRunToTerminalExecutesAutonomously(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: true}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.MaxTotalIterations = 50
	contract.VerificationCommands = []string{"true"}
	run, err := r.StartMissionRun(context.Background(), durablePlan(), t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}

	completed, err := r.RunToTerminal(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != StateCompletedVerified {
		t.Fatalf("expected autonomous completion, got %s", completed.State)
	}
}

type slowExecutor struct {
	fakeExecutor
	delay time.Duration
}

func (s *slowExecutor) Execute(ctx context.Context, run *MissionRun, pkg *PackageRun, prompt string) (ExecutionResult, error) {
	s.executed++
	select {
	case <-ctx.Done():
		return ExecutionResult{}, ctx.Err()
	case <-time.After(s.delay):
		return ExecutionResult{RuntimeID: "rt-slow"}, nil
	}
}

type countingRepo struct {
	*MemoryRunRepository
	renewals int
}

func (c *countingRepo) RenewLease(ctx context.Context, id, owner, token string, ttl time.Duration) error {
	c.renewals++
	return c.MemoryRunRepository.RenewLease(ctx, id, owner, token, ttl)
}

func TestMissionRunnerRenewsLeaseDuringLongProviderOperation(t *testing.T) {
	repo := &countingRepo{MemoryRunRepository: NewMemoryRunRepository()}
	exec := &slowExecutor{fakeExecutor: fakeExecutor{reviewOK: true}, delay: 80 * time.Millisecond}
	r := NewMissionRunner(repo, exec)
	r.leaseTTL = 30 * time.Millisecond
	contract := DefaultAutonomyContract()
	contract.MaxTotalIterations = 20
	contract.VerificationCommands = []string{"true"}
	plan := PlanSpec{ID: "plan-heartbeat", ProjectID: "proj", Revision: 1, Packages: []PackageSpec{{ID: "pkg", Title: "P", Goal: "G"}}}
	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	// READY -> ALLOCATING -> COMPILING -> EXECUTING
	for i := 0; i < 3; i++ {
		if _, _, err := r.ExecuteNextStep(context.Background(), run.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := r.ExecuteNextStep(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	}
	if repo.renewals < 2 {
		t.Fatalf("expected heartbeat renewals during slow execution, got %d", repo.renewals)
	}
}

func TestMemoryRepositoryRejectsStaleFencedSave(t *testing.T) {
	repo := NewMemoryRunRepository()
	run := &MissionRun{ID: "run-fence", PlanID: "p", ProjectID: "proj", State: StateExecuting, StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	first, err := repo.AcquireLease(context.Background(), run.ID, "worker-a", 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(15 * time.Millisecond)
	second, err := repo.AcquireLease(context.Background(), run.ID, "worker-b", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	second.State = StatePaused
	if err := repo.SaveRun(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	first.State = StateFailed
	if err := repo.SaveRun(context.Background(), first); !errors.Is(err, ErrLeaseHeld) {
		t.Fatalf("expected stale fenced save rejection, got %v", err)
	}
	got, err := repo.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StatePaused {
		t.Fatalf("stale writer overwrote state: %s", got.State)
	}
}

type allocateFailExecutor struct{ fakeExecutor }

func (a *allocateFailExecutor) Allocate(context.Context, *MissionRun, *PackageRun) (AllocationResult, error) {
	a.allocated++
	if a.allocated == 1 {
		return AllocationResult{}, fmt.Errorf("no resource")
	}
	return AllocationResult{AgentID: "agt", Workspace: "."}, nil
}

func TestMissionRunnerRetriesFromFailedStage(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &allocateFailExecutor{fakeExecutor: fakeExecutor{reviewOK: true}}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.MaxRetries = 3
	contract.MaxTotalIterations = 20
	contract.VerificationCommands = []string{"true"}
	plan := PlanSpec{ID: "plan-retry-stage", ProjectID: "proj", Revision: 1, Packages: []PackageSpec{{ID: "pkg", Title: "P", Goal: "G"}}}
	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.ExecuteNextStep(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	} // ready->allocating
	updated, _, err := r.ExecuteNextStep(context.Background(), run.ID)
	if err == nil {
		t.Fatal("expected allocation failure")
	}
	if updated.PackageRuns[0].RetryFrom != StateAllocating {
		t.Fatalf("retry stage = %s", updated.PackageRuns[0].RetryFrom)
	}
	if _, _, err := r.ExecuteNextStep(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	} // remediation->allocating
	updated, _, err = r.ExecuteNextStep(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.PackageRuns[0].State != StateCompiling {
		t.Fatalf("expected allocation retry, got %s", updated.PackageRuns[0].State)
	}
}

func TestMissionRunnerStopsRepeatedIdenticalFailuresAsNoProgress(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: false}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.MaxRetries = 10
	contract.MaxNoProgress = 2
	contract.MaxTotalIterations = 50
	contract.VerificationCommands = []string{"true"}
	plan := PlanSpec{ID: "plan-no-progress", ProjectID: "proj", Revision: 1, Packages: []PackageSpec{{ID: "pkg", Title: "P", Goal: "G"}}}
	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30; i++ {
		updated, _, stepErr := r.ExecuteNextStep(context.Background(), run.ID)
		if updated != nil {
			run = updated
		}
		if run.State == StateFailedNoProgress {
			break
		}
		_ = stepErr
	}
	if run.State != StateFailedNoProgress {
		t.Fatalf("expected repeated identical failure to stop as no progress, got %s", run.State)
	}
}

type remediationCompileExecutor struct {
	fakeExecutor
	compileVersion int
	reviews        int
}

func (e *remediationCompileExecutor) Compile(context.Context, *MissionRun, *PackageRun) (PromptArtifact, error) {
	e.compiled++
	e.compileVersion++
	return PromptArtifact{VersionID: fmt.Sprintf("prompt-v%d", e.compileVersion), Content: fmt.Sprintf("attempt-%d", e.compileVersion)}, nil
}
func (e *remediationCompileExecutor) Review(context.Context, *MissionRun, *PackageRun) (ReviewVerdict, error) {
	e.reviewed++
	e.reviews++
	return ReviewVerdict{Approved: e.reviews >= 2, ReviewerAgentID: "reviewer", Findings: []string{"same finding"}}, nil
}

func TestMissionRunnerRecompilesImmutablePromptForRemediationAttempt(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &remediationCompileExecutor{}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.MaxRetries = 4
	contract.MaxNoProgress = 3
	contract.MaxTotalIterations = 30
	contract.VerificationCommands = []string{"true"}
	plan := PlanSpec{ID: "plan-remediation-prompt", ProjectID: "proj", Revision: 1, Packages: []PackageSpec{{ID: "pkg", Title: "P", Goal: "G"}}}
	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	completed, err := r.RunToTerminal(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != StateCompletedVerified {
		t.Fatalf("state=%s", completed.State)
	}
	if exec.compiled < 2 {
		t.Fatalf("remediation must compile a new immutable prompt, compiled=%d", exec.compiled)
	}
	pkg := completed.PackageRuns[0]
	if pkg.PromptVersionID != "prompt-v2" {
		t.Fatalf("expected latest remediation prompt version, got %s", pkg.PromptVersionID)
	}
}

type verificationEscalationExecutor struct {
	fakeExecutor
	workspace   string
	allocations []int
}

func (e *verificationEscalationExecutor) Allocate(ctx context.Context, run *MissionRun, pkg *PackageRun) (AllocationResult, error) {
	e.allocations = append(e.allocations, pkg.Attempt)
	return e.fakeExecutor.Allocate(ctx, run, pkg)
}

func (e *verificationEscalationExecutor) Execute(ctx context.Context, run *MissionRun, pkg *PackageRun, prompt string) (ExecutionResult, error) {
	if pkg.Attempt > 1 {
		if err := os.WriteFile(filepath.Join(run.Workspace, "verified.marker"), []byte("verified"), 0600); err != nil {
			return ExecutionResult{}, err
		}
	}
	return e.fakeExecutor.Execute(ctx, run, pkg, prompt)
}

func TestVerificationFailureReturnsToAllocationForRuntimeEscalation(t *testing.T) {
	workspace := t.TempDir()
	exec := &verificationEscalationExecutor{fakeExecutor: fakeExecutor{reviewOK: true}, workspace: workspace}
	r := NewMissionRunner(NewMemoryRunRepository(), exec)
	contract := DefaultAutonomyContract()
	contract.MaxRetries = 2
	contract.VerificationCommands = []string{"test -f verified.marker"}
	run, err := r.StartMissionRun(context.Background(), PlanSpec{ID: "model-escalation", ProjectID: "proj", Revision: 1, Packages: []PackageSpec{{ID: "pkg", Title: "Model escalation"}}}, workspace, contract, "")
	if err != nil {
		t.Fatal(err)
	}
	completed, err := r.RunToTerminal(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != StateCompletedVerified || len(exec.allocations) != 2 || exec.allocations[1] != 2 {
		t.Fatalf("verification retry did not re-enter allocation: state=%s allocations=%v", completed.State, exec.allocations)
	}
}

func TestManualControlAgentPrefersCurrentRunnablePackage(t *testing.T) {
	run := &MissionRun{CurrentPkgIndex: 1, PackageRuns: []PackageRun{
		{PackageID: "a", AssignedAgent: "agent-a", State: StateVerified},
		{PackageID: "b", AssignedAgent: "agent-b", State: StateTesting},
	}}
	if got := ManualControlAgentID(run); got != "agent-b" {
		t.Fatalf("expected agent-b, got %q", got)
	}
}

func TestManualControlAgentFallsBackToActiveAssignedPackage(t *testing.T) {
	run := &MissionRun{CurrentPkgIndex: 0, PackageRuns: []PackageRun{
		{PackageID: "a", State: StatePending},
		{PackageID: "b", AssignedAgent: "agent-b", State: StateCompiling},
	}}
	if got := ManualControlAgentID(run); got != "agent-b" {
		t.Fatalf("expected agent-b, got %q", got)
	}
}

type concurrencyExecutor struct {
	fakeExecutor
	mu        sync.Mutex
	active    int
	maxActive int
	delay     time.Duration
}

func (c *concurrencyExecutor) Execute(ctx context.Context, run *MissionRun, pkg *PackageRun, prompt string) (ExecutionResult, error) {
	c.mu.Lock()
	c.active++
	if c.active > c.maxActive {
		c.maxActive = c.active
	}
	c.mu.Unlock()
	select {
	case <-ctx.Done():
		c.mu.Lock()
		c.active--
		c.mu.Unlock()
		return ExecutionResult{}, ctx.Err()
	case <-time.After(c.delay):
	}
	c.mu.Lock()
	c.active--
	c.executed++
	c.mu.Unlock()
	return ExecutionResult{RuntimeID: "rt-" + pkg.PackageID}, nil
}

func TestRunToTerminalExecutesParallelGroupProviderTurnsConcurrently(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &concurrencyExecutor{fakeExecutor: fakeExecutor{reviewOK: true}, delay: 40 * time.Millisecond}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.MaxTotalIterations = 100
	contract.VerificationCommands = []string{"true"}
	plan := PlanSpec{ID: "parallel", ProjectID: "proj", Revision: 1, Packages: []PackageSpec{
		{ID: "web", Title: "Web", ParallelGroup: "build"},
		{ID: "api", Title: "API", ParallelGroup: "build"},
	}}
	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	completed, err := r.RunToTerminal(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != StateCompletedVerified {
		t.Fatalf("state=%s", completed.State)
	}
	if exec.maxActive < 2 {
		t.Fatalf("expected concurrent provider execution, maxActive=%d", exec.maxActive)
	}
}

func TestParallelPackagesDoNotShareDefaultAgentImplicitly(t *testing.T) {
	r := NewMissionRunner(NewMemoryRunRepository(), &fakeExecutor{reviewOK: true})
	contract := DefaultAutonomyContract()
	plan := PlanSpec{ID: "parallel", ProjectID: "proj", Revision: 1, Packages: []PackageSpec{
		{ID: "web", ParallelGroup: "build"}, {ID: "api", ParallelGroup: "build"},
	}}
	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "agent-default")
	if err != nil {
		t.Fatal(err)
	}
	if run.PackageRuns[0].AssignedAgent != "" || run.PackageRuns[1].AssignedAgent != "" {
		t.Fatalf("parallel packages must allocate isolated Agents, got %q / %q", run.PackageRuns[0].AssignedAgent, run.PackageRuns[1].AssignedAgent)
	}
}

type alwaysFailExecutor struct{ fakeExecutor }

func (f *alwaysFailExecutor) Execute(context.Context, *MissionRun, *PackageRun, string) (ExecutionResult, error) {
	f.executed++
	return ExecutionResult{}, MarkDispatchNotSent(errors.New("synthetic provider failure"))
}

func TestMissionRunnerHonorsAutoRemediateDisabled(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &alwaysFailExecutor{fakeExecutor: fakeExecutor{reviewOK: true}}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.AutoRemediate = false
	contract.EscalateOnFailure = true
	contract.VerificationCommands = []string{"true"}
	plan := PlanSpec{ID: "plan-no-remediation", ProjectID: "proj-1", Revision: 1, Packages: []PackageSpec{{ID: "pkg", Title: "pkg"}}}
	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10 && run.State != StateBlockedNeedsUser; i++ {
		updated, _, _ := r.ExecuteNextStep(context.Background(), run.ID)
		if updated != nil {
			run = updated
		}
	}
	if run.State != StateBlockedNeedsUser {
		t.Fatalf("AutoRemediate=false must block after first execution failure, got state=%s package=%s", run.State, run.PackageRuns[0].State)
	}
	if exec.executed != 1 {
		t.Fatalf("AutoRemediate=false must not retry provider execution, executions=%d", exec.executed)
	}
}

type overnightAcceptanceExecutor struct {
	globalRepairExecutor
	failedOnce bool
}

func (e *overnightAcceptanceExecutor) Execute(ctx context.Context, run *MissionRun, pkg *PackageRun, prompt string) (ExecutionResult, error) {
	if !e.failedOnce {
		e.failedOnce = true
		return ExecutionResult{}, MarkDispatchNotSent(errors.New("injected overnight provider failure"))
	}
	return e.globalRepairExecutor.Execute(ctx, run, pkg, prompt)
}

// TestOvernightAcceptanceSandbox exercises the unattended control loop with a
// parallel plan, a dependency handoff, an injected provider failure, bounded
// remediation and a final global Definition-of-Done check. It intentionally
// uses the deterministic executor seam; provider/native acceptance remains a
// separate external gate.
func TestOvernightAcceptanceSandbox(t *testing.T) {
	workspace := t.TempDir()
	exec := &overnightAcceptanceExecutor{globalRepairExecutor: globalRepairExecutor{fakeExecutor: fakeExecutor{reviewOK: true}}}
	r := NewMissionRunner(NewMemoryRunRepository(), exec)
	contract := DefaultAutonomyContract()
	contract.MaxRetries = 3
	contract.MaxNoProgress = 2
	contract.VerificationCommands = []string{"true"}
	contract.GlobalVerificationCommands = []string{"test -f global-ready"}
	plan := PlanSpec{ID: "overnight-sandbox", ProjectID: "nexus", Revision: 1, Packages: []PackageSpec{
		{ID: "api", Title: "API contract", Goal: "Produce API contract", ParallelGroup: "foundation"},
		{ID: "web", Title: "Web integration", Goal: "Consume API contract", Dependencies: []string{"api"}},
	}}
	run, err := r.StartMissionRun(context.Background(), plan, workspace, contract, "")
	if err != nil {
		t.Fatal(err)
	}
	completed, err := r.RunToTerminal(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != StateCompletedVerified {
		t.Fatalf("overnight sandbox did not converge: %s", completed.State)
	}
	if completed.PackageRuns[0].Attempt == 0 || completed.PackageRuns[0].StrategyVariant == "" {
		t.Fatalf("injected failure did not produce bounded remediation evidence: %+v", completed.PackageRuns[0])
	}
	if len(completed.PackageRuns[1].ContextCapsule.DependencyReceipts) != 1 {
		t.Fatalf("dependency artifact handoff missing: %+v", completed.PackageRuns[1].ContextCapsule)
	}
	if len(completed.GlobalVerifications) == 0 || !completed.GlobalVerifications[len(completed.GlobalVerifications)-1].Passed {
		t.Fatalf("global verification evidence incomplete: %+v", completed.GlobalVerifications)
	}
}

type dispatchObservingExecutor struct {
	fakeExecutor
	repo              RunRepository
	mu                sync.Mutex
	intentPersisted   bool
	allGroupPersisted bool
}

func (e *dispatchObservingExecutor) Execute(ctx context.Context, run *MissionRun, pkg *PackageRun, prompt string) (ExecutionResult, error) {
	persisted, err := e.repo.GetRun(ctx, run.ID)
	if err == nil {
		intentPersisted := false
		allGroupPersisted := false
		for i := range persisted.PackageRuns {
			candidate := &persisted.PackageRuns[i]
			if candidate.PackageID == pkg.PackageID && candidate.DispatchID != "" && candidate.DispatchState == DispatchIntent {
				intentPersisted = true
			}
		}
		all := true
		for i := range persisted.PackageRuns {
			candidate := &persisted.PackageRuns[i]
			if candidate.ParallelGroup == pkg.ParallelGroup && candidate.State == StateExecuting {
				if candidate.DispatchID == "" || candidate.DispatchState != DispatchIntent {
					all = false
				}
			}
		}
		if all {
			allGroupPersisted = true
		}
		e.mu.Lock()
		if intentPersisted {
			e.intentPersisted = true
		}
		if allGroupPersisted {
			e.allGroupPersisted = true
		}
		e.mu.Unlock()
	}
	return e.fakeExecutor.Execute(ctx, run, pkg, prompt)
}

func TestMissionRunnerPersistsDispatchIntentBeforeProviderExecution(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &dispatchObservingExecutor{fakeExecutor: fakeExecutor{reviewOK: true}, repo: repo}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{"true"}
	plan := PlanSpec{ID: "dispatch-intent", ProjectID: "proj", Revision: 1, Packages: []PackageSpec{{ID: "pkg", Title: "P", Goal: "G"}}}
	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		updated, _, stepErr := r.ExecuteNextStep(context.Background(), run.ID)
		if stepErr != nil {
			t.Fatal(stepErr)
		}
		run = updated
	}
	if !exec.intentPersisted {
		t.Fatal("dispatch intent must be durably persisted before provider Execute")
	}
	if run.PackageRuns[0].DispatchState != DispatchCompleted {
		t.Fatalf("successful provider turn must finish dispatch evidence, got %q", run.PackageRuns[0].DispatchState)
	}
}

func TestMissionRunnerRestartRefusesUnknownInFlightDispatch(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: true}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{"true"}
	plan := PlanSpec{ID: "dispatch-restart", ProjectID: "proj", Revision: 1, Packages: []PackageSpec{{ID: "pkg", Title: "P", Goal: "G"}}}
	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	run.PackageRuns[0].State = StateExecuting
	run.PackageRuns[0].DispatchID = "dispatch-crash-window"
	run.PackageRuns[0].DispatchState = DispatchIntent
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}

	restarted := NewMissionRunner(repo, exec)
	updated, _, err := restarted.ExecuteNextStep(context.Background(), run.ID)
	if err == nil {
		t.Fatal("unknown in-flight dispatch must fail closed instead of re-executing")
	}
	if exec.executed != 0 {
		t.Fatalf("provider was dispatched twice after restart: executions=%d", exec.executed)
	}
	if updated.State != StateBlockedNeedsUser {
		t.Fatalf("unknown dispatch outcome must block for user resolution, got %s", updated.State)
	}
}

func TestMissionRunnerParallelGroupPersistsAllDispatchIntentsBeforeLaunch(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &dispatchObservingExecutor{fakeExecutor: fakeExecutor{reviewOK: true}, repo: repo}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{"true"}
	contract.MaxTotalIterations = 40
	plan := PlanSpec{ID: "dispatch-parallel", ProjectID: "proj", Revision: 1, Packages: []PackageSpec{
		{ID: "web", Title: "Web", ParallelGroup: "build"},
		{ID: "api", Title: "API", ParallelGroup: "build"},
	}}
	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 7; i++ {
		updated, _, stepErr := r.ExecuteNextStep(context.Background(), run.ID)
		if stepErr != nil {
			t.Fatal(stepErr)
		}
		run = updated
		if exec.executedCount() > 0 {
			break
		}
	}
	if !exec.allGroupPersisted {
		t.Fatal("all parallel dispatch intents must be persisted before any provider goroutine launches")
	}
}
