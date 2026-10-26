package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const (
	missionCrashDBEnv     = "NEXUS_TEST_MISSION_CRASH_DB"
	missionCrashMarkerEnv = "NEXUS_TEST_MISSION_CRASH_MARKER"
	missionCrashWorkEnv   = "NEXUS_TEST_MISSION_CRASH_WORKSPACE"
)

type missionCrashMarker struct {
	RunID      string `json:"run_id"`
	DispatchID string `json:"dispatch_id"`
	State      string `json:"dispatch_state"`
}

type interruptedDispatchExecutor struct {
	markerPath string
}

func (*interruptedDispatchExecutor) Allocate(_ context.Context, _ *runner.MissionRun, _ *runner.PackageRun) (runner.AllocationResult, error) {
	return runner.AllocationResult{AgentID: "crash-test-agent", Workspace: "."}, nil
}

func (*interruptedDispatchExecutor) Compile(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.PromptArtifact, error) {
	return runner.PromptArtifact{VersionID: "crash-test-prompt", Content: "safe test fixture"}, nil
}

func (e *interruptedDispatchExecutor) Execute(_ context.Context, run *runner.MissionRun, pkg *runner.PackageRun, _ string) (runner.ExecutionResult, error) {
	if pkg.DispatchState != runner.DispatchIntent || pkg.DispatchID == "" {
		return runner.ExecutionResult{}, fmt.Errorf("executor entered without persisted dispatch intent: state=%q id=%q", pkg.DispatchState, pkg.DispatchID)
	}
	marker := missionCrashMarker{RunID: run.ID, DispatchID: pkg.DispatchID, State: string(pkg.DispatchState)}
	encoded, err := json.Marshal(marker)
	if err != nil {
		return runner.ExecutionResult{}, err
	}
	if err := os.WriteFile(e.markerPath, encoded, 0600); err != nil {
		return runner.ExecutionResult{}, err
	}
	select {}
}

func (*interruptedDispatchExecutor) Review(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.ReviewVerdict, error) {
	return runner.ReviewVerdict{Approved: true, ReviewerAgentID: "crash-test-reviewer", ReviewedAt: time.Now().UTC()}, nil
}

type recoveryCountingExecutor struct {
	executions atomic.Int32
}

func (*recoveryCountingExecutor) Allocate(_ context.Context, _ *runner.MissionRun, _ *runner.PackageRun) (runner.AllocationResult, error) {
	return runner.AllocationResult{AgentID: "recovery-agent", Workspace: "."}, nil
}

func (*recoveryCountingExecutor) Compile(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.PromptArtifact, error) {
	return runner.PromptArtifact{VersionID: "recovery-prompt", Content: "must not replay"}, nil
}

func (e *recoveryCountingExecutor) Execute(context.Context, *runner.MissionRun, *runner.PackageRun, string) (runner.ExecutionResult, error) {
	e.executions.Add(1)
	return runner.ExecutionResult{RuntimeID: "unexpected-replay"}, nil
}

func (*recoveryCountingExecutor) Review(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.ReviewVerdict, error) {
	return runner.ReviewVerdict{Approved: true, ReviewerAgentID: "recovery-reviewer", ReviewedAt: time.Now().UTC()}, nil
}

// The subprocess is killed while Execute is blocked after the SQLite dispatch
// intent is committed. A new Nexus process must recover that run without
// replaying a provider call whose outcome cannot be known.
func TestMissionProcessCrashDuringProviderDispatchFailsClosed(t *testing.T) {
	if dbPath := os.Getenv(missionCrashDBEnv); dbPath != "" {
		runMissionCrashChild(t, dbPath, os.Getenv(missionCrashMarkerEnv), os.Getenv(missionCrashWorkEnv))
		return
	}

	root := t.TempDir()
	dbPath := filepath.Join(root, "nexus.db")
	markerPath := filepath.Join(root, "dispatch.json")
	workspace := filepath.Join(root, "workspace")
	if err := os.Mkdir(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "child-test.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestMissionProcessCrashDuringProviderDispatchFailsClosed$")
	cmd.Env = withEnvironment(os.Environ(), map[string]string{
		missionCrashDBEnv:     dbPath,
		missionCrashMarkerEnv: markerPath,
		missionCrashWorkEnv:   workspace,
	})
	cmd.Stdout, cmd.Stderr = logFile, logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		t.Fatal(err)
	}
	_ = logFile.Close()
	childPID := cmd.Process.Pid
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()

	deadline := time.NewTimer(15 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(markerPath); err == nil {
			break
		}
		select {
		case childErr := <-wait:
			logBytes, _ := os.ReadFile(logPath)
			t.Fatalf("child exited before dispatch became active (pid=%d, err=%v): %s", childPID, childErr, strings.TrimSpace(string(logBytes)))
		case <-deadline.C:
			_ = cmd.Process.Kill()
			<-wait
			logBytes, _ := os.ReadFile(logPath)
			t.Fatalf("child did not reach provider dispatch within 15s (pid=%d): %s", childPID, strings.TrimSpace(string(logBytes)))
		case <-ticker.C:
		}
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill active child process %d: %v", childPID, err)
	}
	if err := <-wait; err == nil {
		t.Fatalf("child process %d exited successfully after forced termination", childPID)
	}

	encodedMarker, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatal(err)
	}
	var marker missionCrashMarker
	if err := json.Unmarshal(encodedMarker, &marker); err != nil {
		t.Fatalf("decode active dispatch marker: %v", err)
	}
	if marker.State != string(runner.DispatchIntent) || marker.RunID == "" || marker.DispatchID == "" {
		t.Fatalf("child did not crash inside a persisted dispatch: %+v", marker)
	}

	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen SQLite after process crash: %v", err)
	}
	defer st.Close()
	repo := newStoreRunRepository(st)
	crashedRun, err := repo.GetRun(context.Background(), marker.RunID)
	if err != nil {
		t.Fatalf("load mission after process crash: %v", err)
	}
	if crashedRun.State != runner.StateExecuting || crashedRun.PackageRuns[0].DispatchState != runner.DispatchIntent || crashedRun.PackageRuns[0].DispatchID != marker.DispatchID {
		t.Fatalf("crash evidence was not durable: run=%s package=%+v", crashedRun.State, crashedRun.PackageRuns[0])
	}

	recoveryExecutor := &recoveryCountingExecutor{}
	recoveredNexus := &Nexus{st: st, runner: runner.NewMissionRunner(repo, recoveryExecutor), workers: map[string]*missionWorker{}}
	workers, err := recoveredNexus.recoverMissionRuns(context.Background())
	if err != nil {
		t.Fatalf("start mission recovery after process crash: %v", err)
	}
	if len(workers) != 1 {
		t.Fatalf("recovery started %d workers, want 1", len(workers))
	}
	select {
	case <-workers[0].done:
	case <-time.After(40 * time.Second):
		t.Fatal("recovered worker did not settle within 40 seconds")
	}
	if !errors.Is(workers[0].err, runner.ErrDispatchOutcomeUnknown) {
		t.Fatalf("recovery error = %v, want fail-closed unknown dispatch", workers[0].err)
	}
	finalRun, err := repo.GetRun(context.Background(), marker.RunID)
	if err != nil {
		t.Fatalf("load recovered mission: %v", err)
	}
	if finalRun.State != runner.StateBlockedNeedsUser || finalRun.NeedsHuman == nil || finalRun.NeedsHuman.ReasonCode != "DISPATCH_OUTCOME_UNKNOWN" {
		t.Fatalf("unknown provider outcome was not escalated for reconciliation: %+v", finalRun)
	}
	if finalRun.PackageRuns[0].DispatchState != runner.DispatchUnknownExternalOutcome || finalRun.PackageRuns[0].DispatchID != marker.DispatchID {
		t.Fatalf("recovery changed dispatch identity or failed to mark it unknown: %+v", finalRun.PackageRuns[0])
	}
	if got := recoveryExecutor.executions.Load(); got != 0 {
		t.Fatalf("recovery replayed %d provider calls with an unknown prior outcome", got)
	}
}

func runMissionCrashChild(t *testing.T, dbPath, markerPath, workspace string) {
	t.Helper()
	if markerPath == "" || workspace == "" {
		t.Fatal("crash helper requires marker and workspace paths")
	}
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Crash recovery", CanonicalPath: workspace})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(store.WorkPlan{ProjectID: project.ID, Title: "Crash recovery"})
	if err != nil {
		t.Fatal(err)
	}
	contract := runner.DefaultAutonomyContract()
	contract.RequireVerification = false
	contract.MaxTotalIterations = 10
	missionRunner := runner.NewMissionRunner(newStoreRunRepository(st), &interruptedDispatchExecutor{markerPath: markerPath})
	run, err := missionRunner.StartMissionRun(context.Background(), runner.PlanSpec{
		ID: plan.ID, ProjectID: project.ID, Revision: plan.CurrentRevision,
		Packages: []runner.PackageSpec{{ID: "active-task", Title: "Active task", Goal: "Hold dispatch open for process crash test"}},
	}, workspace, contract, "")
	if err != nil {
		t.Fatal(err)
	}
	for range 10 {
		_, done, stepErr := missionRunner.ExecuteNextStep(context.Background(), run.ID)
		if stepErr != nil {
			t.Fatal(stepErr)
		}
		if done {
			t.Fatal("crash fixture completed before provider dispatch was interrupted")
		}
	}
	t.Fatal("crash fixture did not enter provider dispatch")
}

func withEnvironment(environment []string, overrides map[string]string) []string {
	for key, value := range overrides {
		prefix := key + "="
		filtered := environment[:0]
		for _, entry := range environment {
			if !strings.HasPrefix(entry, prefix) {
				filtered = append(filtered, entry)
			}
		}
		environment = append(filtered, prefix+value)
	}
	return environment
}
