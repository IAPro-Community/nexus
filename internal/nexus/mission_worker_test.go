package nexus

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

type delayedHandoffExecutor struct {
	allocations   int
	firstStarted  chan struct{}
	firstRelease  chan struct{}
	secondStart   chan struct{}
	secondRelease chan struct{}
}

func (e *delayedHandoffExecutor) Allocate(_ context.Context, _ *runner.MissionRun, _ *runner.PackageRun) (runner.AllocationResult, error) {
	e.allocations++
	switch e.allocations {
	case 1:
		close(e.firstStarted)
		<-e.firstRelease // deliberately ignore cancellation to force handoff timeout
		return runner.AllocationResult{}, errors.New("old allocation exited after handoff timeout")
	case 2:
		close(e.secondStart)
		<-e.secondRelease
	}
	return runner.AllocationResult{AgentID: "agent", Workspace: "."}, nil
}

func (*delayedHandoffExecutor) Compile(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.PromptArtifact, error) {
	return runner.PromptArtifact{VersionID: "prompt", Content: "resume"}, nil
}

func (*delayedHandoffExecutor) Execute(context.Context, *runner.MissionRun, *runner.PackageRun, string) (runner.ExecutionResult, error) {
	return runner.ExecutionResult{RuntimeID: "runtime"}, nil
}

func (*delayedHandoffExecutor) Review(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.ReviewVerdict, error) {
	return runner.ReviewVerdict{Approved: true, ReviewerAgentID: "reviewer", ReviewedAt: time.Now().UTC()}, nil
}

func TestReleaseMissionWorkerCannotDeleteNewGeneration(t *testing.T) {
	n := &Nexus{workers: map[string]*missionWorker{}}
	old := &missionWorker{done: make(chan struct{})}
	newGeneration := &missionWorker{done: make(chan struct{})}
	n.workers["run-1"] = newGeneration

	n.releaseMissionWorker("run-1", old)
	n.workersMu.Lock()
	current := n.workers["run-1"]
	n.workersMu.Unlock()
	if current != newGeneration {
		t.Fatal("old worker removed or replaced the newer worker generation")
	}

	n.releaseMissionWorker("run-1", newGeneration)
	n.workersMu.Lock()
	_, exists := n.workers["run-1"]
	n.workersMu.Unlock()
	if exists {
		t.Fatal("current worker generation was not released")
	}
}

func TestMissionWorkerHandoffRetriesAfterOldWorkerExceedsTimeout(t *testing.T) {
	ctx := context.Background()
	repo := runner.NewMemoryRunRepository()
	exec := &delayedHandoffExecutor{
		firstStarted: make(chan struct{}), firstRelease: make(chan struct{}),
		secondStart: make(chan struct{}), secondRelease: make(chan struct{}),
	}
	missionRunner := runner.NewMissionRunner(repo, exec)
	contract := runner.DefaultAutonomyContract()
	contract.RequireVerification = false
	contract.MaxTotalIterations = 20
	run := &runner.MissionRun{
		ID: "run-worker-handoff", State: runner.StateExecuting, Contract: contract,
		PackageRuns: []runner.PackageRun{{ID: "pkg", PackageID: "pkg", State: runner.StateAllocating}},
		UpdatedAt:   time.Now().UTC(),
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	n := &Nexus{runner: missionRunner, workers: map[string]*missionWorker{}}
	old := n.startMissionWorker(run.ID, false)
	select {
	case <-exec.firstStarted:
	case <-time.After(time.Second):
		t.Fatal("old worker did not reach the blocking allocator")
	}

	returned := n.startMissionWorkerWithHandoffTimeout(run.ID, true, time.Millisecond)
	if returned != old {
		t.Fatalf("timed-out handoff should retain the old generation until it exits: got=%p want=%p", returned, old)
	}
	close(exec.firstRelease)
	select {
	case <-exec.secondStart:
	case <-time.After(time.Second):
		t.Fatal("pending resume was not retried after the old worker exited")
	}
	n.workersMu.Lock()
	current := n.workers[run.ID]
	n.workersMu.Unlock()
	if current == nil || current == old {
		t.Fatal("new worker generation was not registered after handoff")
	}
	close(exec.secondRelease)
	select {
	case <-current.done:
	case <-time.After(time.Second):
		t.Fatal("replacement worker did not finish")
	}
	final, err := repo.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if final.State != runner.StateCompletedVerified || exec.allocations != 2 {
		t.Fatalf("replacement did not recover to terminal state: state=%s allocations=%d", final.State, exec.allocations)
	}
}

func TestWaitMissionWorkerReturnsDurableTerminalRun(t *testing.T) {
	repo := runner.NewMemoryRunRepository()
	missionRunner := runner.NewMissionRunner(repo, nil)
	run := &runner.MissionRun{ID: "run-wait-complete", State: runner.StateCompletedVerified}
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	worker := &missionWorker{done: make(chan struct{})}
	close(worker.done)
	n := &Nexus{
		runner:  missionRunner,
		workers: map[string]*missionWorker{run.ID: worker},
	}

	got, err := n.WaitMissionWorker(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.State != runner.StateCompletedVerified {
		t.Fatalf("wait returned unexpected run: %+v", got)
	}
}

func TestWaitMissionWorkerReturnsWorkerFailureWithDurableState(t *testing.T) {
	repo := runner.NewMemoryRunRepository()
	missionRunner := runner.NewMissionRunner(repo, nil)
	run := &runner.MissionRun{ID: "run-wait-failed", State: runner.StateFailed}
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("provider failed")
	worker := &missionWorker{done: make(chan struct{}), err: wantErr}
	close(worker.done)
	n := &Nexus{
		runner:  missionRunner,
		workers: map[string]*missionWorker{run.ID: worker},
	}

	got, err := n.WaitMissionWorker(context.Background(), run.ID)
	if !errors.Is(err, wantErr) {
		t.Fatalf("wait error=%v, want %v", err, wantErr)
	}
	if got == nil || got.State != runner.StateFailed {
		t.Fatalf("wait lost durable failed state: %+v", got)
	}
}

func TestWaitMissionWorkerReturnsFailureWhenWorkerAlreadyReleased(t *testing.T) {
	repo := runner.NewMemoryRunRepository()
	missionRunner := runner.NewMissionRunner(repo, nil)
	run := &runner.MissionRun{ID: "run-fast-failed", State: runner.StateFailedVerification, PausedReason: "global verification failed"}
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	n := &Nexus{runner: missionRunner, workers: map[string]*missionWorker{}}

	got, err := n.WaitMissionWorker(context.Background(), run.ID)
	if err == nil || !strings.Contains(err.Error(), "FAILED_VERIFICATION") {
		t.Fatalf("fast failed Mission must retain non-zero CLI semantics: run=%+v err=%v", got, err)
	}
}

func TestWaitMissionWorkerRejectsOrphanedActiveRun(t *testing.T) {
	repo := runner.NewMemoryRunRepository()
	missionRunner := runner.NewMissionRunner(repo, nil)
	run := &runner.MissionRun{ID: "run-orphaned", State: runner.StateExecuting}
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	n := &Nexus{runner: missionRunner, workers: map[string]*missionWorker{}}

	got, err := n.WaitMissionWorker(context.Background(), run.ID)
	if err == nil || !strings.Contains(err.Error(), "no active worker") {
		t.Fatalf("orphaned active run must fail closed: run=%+v err=%v", got, err)
	}
}
