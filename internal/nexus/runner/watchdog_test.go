package runner

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestProgressWatchdogIgnoresFreshExecutingWork(t *testing.T) {
	now := time.Now().UTC()
	run := &MissionRun{
		ID:             "run_fresh",
		State:          StateExecuting,
		StartedAt:      now,
		LastProgressAt: now,
		Contract:       AutonomyContract{StallTimeoutSeconds: 15},
		PackageRuns:    []PackageRun{{PackageID: "pkg", State: StateExecuting}},
	}
	if ApplyProgressWatchdog(run, now) {
		t.Fatal("fresh executing work must not be treated as stalled")
	}
	if run.State != StateExecuting {
		t.Fatalf("state mutated: %s", run.State)
	}
}

func TestProgressWatchdogFailsClosedOnStallWithoutNeedsUser(t *testing.T) {
	now := time.Now().UTC()
	run := &MissionRun{
		ID:             "run_stall",
		State:          StateExecuting,
		StartedAt:      now.Add(-time.Hour),
		LastProgressAt: now.Add(-20 * time.Minute),
		UpdatedAt:      now, // lease/heartbeat activity is not progress
		Contract:       AutonomyContract{StallTimeoutSeconds: 900, EscalateOnFailure: true},
		PackageRuns:    []PackageRun{{PackageID: "pkg", State: StateExecuting}},
	}
	if !ApplyProgressWatchdog(run, now) {
		t.Fatal("stalled EXECUTING mission must fail closed")
	}
	if run.State != StateFailedNoProgress {
		t.Fatalf("expected FAILED_NO_PROGRESS, got %s", run.State)
	}
	if run.NeedsHuman != nil {
		t.Fatalf("stall must not become NEEDS_YOU: %+v", run.NeedsHuman)
	}
	if ClassifyAttention(run) == AttentionRequireUser {
		t.Fatal("FAILED_NO_PROGRESS must not classify as REQUIRE_USER")
	}
}

func TestProgressWatchdogDoesNotTouchTerminalStates(t *testing.T) {
	now := time.Now().UTC()
	run := &MissionRun{
		ID:             "run_done",
		State:          StateCompletedVerified,
		LastProgressAt: now.Add(-time.Hour),
		Contract:       AutonomyContract{StallTimeoutSeconds: 1},
	}
	if ApplyProgressWatchdog(run, now) {
		t.Fatal("terminal runs must not be stalled")
	}
}

func TestArmStallWatchdogCancelsBlockedOperationContext(t *testing.T) {
	now := time.Now().UTC()
	run := &MissionRun{
		ID:             "run_watch_cancel",
		State:          StateExecuting,
		StartedAt:      now.Add(-time.Minute),
		LastProgressAt: now.Add(-2 * time.Second),
		Contract:       AutonomyContract{StallTimeoutSeconds: 1},
		PackageRuns:    []PackageRun{{PackageID: "pkg", State: StateExecuting}},
	}
	parent := context.Background()
	opCtx, cancel := armStallWatchdog(parent, run)
	defer cancel()
	select {
	case <-opCtx.Done():
		if !errors.Is(opCtx.Err(), context.Canceled) {
			t.Fatalf("expected canceled context, got %v", opCtx.Err())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("stall watchdog did not cancel blocked operation context")
	}
}

func TestFinalizeStallIfNeededPromotesCanceledToFailedNoProgress(t *testing.T) {
	now := time.Now().UTC()
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{reviewOK: true})
	run := &MissionRun{
		ID:             "run_finalize_stall",
		PlanID:         "plan",
		ProjectID:      "project",
		State:          StateExecuting,
		StartedAt:      now.Add(-time.Hour),
		LastProgressAt: now.Add(-time.Hour),
		UpdatedAt:      now.Add(-time.Hour),
		Contract:       AutonomyContract{StallTimeoutSeconds: 1, MaxRetries: 1, MaxTotalIterations: 10},
		PackageRuns:    []PackageRun{{PackageID: "pkg", State: StateExecuting, StartedAt: now.Add(-time.Hour)}},
	}
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	stalled, err := r.finalizeStallIfNeeded(context.Background(), run, context.Canceled)
	if !stalled {
		t.Fatal("expected stall finalization")
	}
	if err == nil || !strings.Contains(err.Error(), "stalled") {
		t.Fatalf("error=%v", err)
	}
	if run.State != StateFailedNoProgress {
		t.Fatalf("state=%s want FAILED_NO_PROGRESS", run.State)
	}
}
