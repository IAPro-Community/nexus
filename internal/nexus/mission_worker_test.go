package nexus

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

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
