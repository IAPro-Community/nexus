package nexus

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestTakeControlMissionRunRefusesNonLocalOwnersWithoutMutation(t *testing.T) {
	t.Setenv("NEXUS_FEATURES_ENABLED", "true")
	t.Setenv("NEXUS_FEATURE_MISSIONS", "true")
	for _, tc := range []struct {
		name      string
		owner     string
		wantError error
	}{
		{name: "maestro", owner: runner.ExecutionOwnerMaestro, wantError: runner.ErrExecutionOwnerNotLocal},
		{name: "unknown", owner: "unknown-owner", wantError: runner.ErrInvalidExecutionOwner},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := openTestNexus(t)
			st, err := n.OpenProject()
			if err != nil {
				t.Fatal(err)
			}
			project, err := st.CreateProject(store.Project{Name: "Manual takeover owner", CanonicalPath: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Implementer", Status: store.AgentStopped})
			if err != nil {
				t.Fatal(err)
			}
			revision, err := st.AddRevision(agent.ID, `{"provider":"fake","profile":"default"}`)
			if err != nil {
				t.Fatal(err)
			}
			agent.CurrentRevisionID = revision.ID
			if err := st.UpdateAgent(agent); err != nil {
				t.Fatal(err)
			}

			repo := runner.NewMemoryRunRepository()
			run := &runner.MissionRun{
				ID: "run-takeover-" + tc.name, State: runner.StateExecuting, ExecutionOwner: tc.owner,
				PackageRuns: []runner.PackageRun{{PackageID: "pkg", AssignedAgent: agent.ID, State: runner.StateExecuting}},
			}
			if err := repo.SaveRun(context.Background(), run); err != nil {
				t.Fatal(err)
			}
			n.runner = runner.NewMissionRunner(repo, nil)
			workerCanceled := false
			workerDone := make(chan struct{})
			close(workerDone)
			n.workers = map[string]*missionWorker{run.ID: {
				id:     "worker-" + tc.name,
				cancel: func() { workerCanceled = true },
				done:   workerDone,
			}}

			if _, err := n.TakeControlMissionRun(context.Background(), run.ID, "manual takeover"); !errors.Is(err, tc.wantError) {
				t.Fatalf("takeover error=%v, want %v", err, tc.wantError)
			}
			if workerCanceled {
				t.Fatal("refused takeover stopped the autonomous worker")
			}
			assertManualControlRunUnchanged(t, repo, run.ID, runner.StateExecuting)
			assertManualControlAgentUnchanged(t, st, agent.ID, store.AgentStopped)
		})
	}
}

func TestReturnMissionRunRefusesNonLocalOwnersWithoutMutation(t *testing.T) {
	t.Setenv("NEXUS_FEATURES_ENABLED", "true")
	t.Setenv("NEXUS_FEATURE_MISSIONS", "true")
	for _, tc := range []struct {
		name      string
		owner     string
		wantError error
	}{
		{name: "maestro", owner: runner.ExecutionOwnerMaestro, wantError: runner.ErrExecutionOwnerNotLocal},
		{name: "unknown", owner: "unknown-owner", wantError: runner.ErrInvalidExecutionOwner},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := openTestNexus(t)
			st, err := n.OpenProject()
			if err != nil {
				t.Fatal(err)
			}
			project, err := st.CreateProject(store.Project{Name: "Manual return owner", CanonicalPath: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Implementer", Status: store.AgentWorking})
			if err != nil {
				t.Fatal(err)
			}

			repo := runner.NewMemoryRunRepository()
			run := &runner.MissionRun{
				ID: "run-return-" + tc.name, State: runner.StatePaused, ExecutionOwner: tc.owner,
				PackageRuns: []runner.PackageRun{{PackageID: "pkg", AssignedAgent: agent.ID, State: runner.StateExecuting}},
			}
			if err := repo.SaveRun(context.Background(), run); err != nil {
				t.Fatal(err)
			}
			n.runner = runner.NewMissionRunner(repo, nil)

			if _, err := n.ReturnMissionRun(context.Background(), run.ID); !errors.Is(err, tc.wantError) {
				t.Fatalf("return error=%v, want %v", err, tc.wantError)
			}
			assertManualControlRunUnchanged(t, repo, run.ID, runner.StatePaused)
			assertManualControlAgentUnchanged(t, st, agent.ID, store.AgentWorking)
			n.workersMu.Lock()
			workerCount := len(n.workers)
			n.workersMu.Unlock()
			if workerCount != 0 {
				t.Fatalf("refused return started %d autonomous workers", workerCount)
			}
		})
	}
}

func assertManualControlRunUnchanged(t *testing.T, repo runner.RunRepository, runID string, wantState runner.State) {
	t.Helper()
	stored, err := repo.GetRun(context.Background(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != wantState {
		t.Fatalf("refused manual control mutated run state: got %s want %s", stored.State, wantState)
	}
}

func assertManualControlAgentUnchanged(t *testing.T, st *store.Store, agentID, wantStatus string) {
	t.Helper()
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		t.Fatal(err)
	}
	if agent.Status != wantStatus {
		t.Fatalf("refused manual control mutated Agent status: got %s want %s", agent.Status, wantStatus)
	}
	generations, err := st.ListGenerations(agentID)
	if err != nil {
		t.Fatal(err)
	}
	if len(generations) != 0 {
		t.Fatalf("refused manual control started %d Agent runtimes", len(generations))
	}
}

func TestReconcileEphemeralMissionAgentsPreservesActiveIdentity(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Ephemeral reconciliation", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	createWorker := func(name, status, kind string) store.Agent {
		t.Helper()
		agent, createErr := st.CreateAgent(store.Agent{
			ProjectID: project.ID, Name: name, Status: status, IdentityKind: kind,
		})
		if createErr != nil {
			t.Fatal(createErr)
		}
		return agent
	}
	active := createWorker("Active mission", store.AgentStopped, string(runner.ExecutionIdentityEphemeralWorker))
	terminal := createWorker("Terminal mission", store.AgentStopped, string(runner.ExecutionIdentityEphemeralWorker))
	orphan := createWorker("Orphan", store.AgentFailed, string(runner.ExecutionIdentityEphemeralWorker))
	working := createWorker("Still working", store.AgentWorking, string(runner.ExecutionIdentityEphemeralWorker))
	persistent := createWorker("Persistent", store.AgentStopped, "PERSISTENT")

	removed, err := n.reconcileEphemeralMissionAgents([]*runner.MissionRun{{
		ID: "active-run", ProjectID: project.ID, State: runner.StateExecuting,
		PackageRuns: []runner.PackageRun{{AssignedAgent: active.ID}},
	}, {
		ID: "terminal-run", ProjectID: project.ID, State: runner.StateCompletedVerified,
		PackageRuns: []runner.PackageRun{{AssignedAgent: terminal.ID}},
	}}, "")
	if err != nil {
		t.Fatal(err)
	}
	if removed != 2 {
		t.Fatalf("removed %d ephemeral Agents, want terminal plus orphan, active, working and persistent retained", removed)
	}
	for _, retained := range []store.Agent{active, working, persistent} {
		if _, err := st.GetAgent(retained.ID, project.ID); err != nil {
			t.Errorf("Agent %s should be retained: %v", retained.Name, err)
		}
	}
	for _, deleted := range []store.Agent{terminal, orphan} {
		if _, err := st.GetAgent(deleted.ID, project.ID); err == nil {
			t.Errorf("terminal ephemeral Agent %s should be deleted", deleted.Name)
		}
	}
}

func TestRunApplicationServiceReconcilesEphemeralWorkersAtStartup(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Startup reconciliation", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Finished plan", "", []store.PlanPhase{{
		ID: "phase", Title: "Phase", Packages: []store.WorkPackage{{ID: "step", Title: "Step", Goal: "Done", Status: "READY"}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{
		ProjectID: project.ID, Name: "Finished specialist", Status: store.AgentStopped,
		IdentityKind: string(runner.ExecutionIdentityEphemeralWorker),
	})
	if err != nil {
		t.Fatal(err)
	}
	run := &runner.MissionRun{
		ID: "finished-run", PlanID: plan.ID, ProjectID: project.ID, State: runner.StateCompletedVerified,
		PackageRuns: []runner.PackageRun{{AssignedAgent: agent.ID}},
	}
	if err := newStoreRunRepository(st).SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}

	_ = NewRunApplicationService(n)
	if _, err := st.GetAgent(agent.ID, project.ID); err == nil {
		t.Fatal("startup reconciliation left a stopped Agent from a terminal Mission")
	}
}

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
	t.Setenv("NEXUS_FEATURES_ENABLED", "true")
	t.Setenv("NEXUS_FEATURE_MISSIONS", "true")
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
