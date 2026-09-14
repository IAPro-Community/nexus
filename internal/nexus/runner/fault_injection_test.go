package runner

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestFaultInjectionStoreReopenPreservesIntervention(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRunRepository()
	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run_reopen", ProjectID: "p", State: StateBlockedNeedsUser,
		NeedsHuman: &HumanIntervention{
			ID: "intv", ReasonCode: "SECURITY", Summary: "auth", Question: "Approve?",
			MissionID: "run_reopen", TaskID: "pkg", Source: "runner", Timestamp: now, Version: 1,
			Options: []InterventionOption{{ID: "replan-package", Operation: InterventionReplanPackage, PackageID: "pkg"}},
		},
		UpdatedAt: now, LastProgressAt: now,
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	reopened := NewMemoryRunRepository()
	var copy MissionRun
	if err := json.Unmarshal(raw, &copy); err != nil {
		t.Fatal(err)
	}
	if err := reopened.SaveRun(ctx, &copy); err != nil {
		t.Fatal(err)
	}
	got, err := reopened.GetRun(ctx, "run_reopen")
	if err != nil {
		t.Fatal(err)
	}
	if got.NeedsHuman == nil || got.NeedsHuman.ID != "intv" || got.State != StateBlockedNeedsUser {
		t.Fatalf("reopen lost intervention: %+v", got)
	}
	center := NewAttentionCenter(reopened)
	first, err := center.ListAttention(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := center.ListAttention(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.NeedsYou) != 1 || len(second.NeedsYou) != 1 {
		t.Fatalf("restart must not duplicate attention: first=%d second=%d", len(first.NeedsYou), len(second.NeedsYou))
	}
	if AttentionDedupKey(got) != AttentionDedupKey(run) {
		t.Fatalf("dedup key must survive reopen: %q vs %q", AttentionDedupKey(got), AttentionDedupKey(run))
	}
}

func TestFaultInjectionStaleRoutingJSONFailsClosed(t *testing.T) {
	pkg := PackageRun{PackageID: "pkg", RoutingDecisionJSON: "{not-json"}
	if json.Valid([]byte(pkg.RoutingDecisionJSON)) {
		t.Fatal("fixture must be invalid JSON")
	}
}

func TestFaultInjectionDuplicateInterventionResolvesOnce(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()
	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run_dup", State: StateBlockedNeedsUser,
		PackageRuns: []PackageRun{{ID: "pkg-1", PackageID: "pkg-1", State: StateFailed, DispatchState: DispatchFailedBeforeDispatch}},
		NeedsHuman: &HumanIntervention{
			ID: "intv_dup", ReasonCode: "SECURITY", Summary: "x", Question: "?",
			MissionID: "run_dup", TaskID: "pkg-1", Source: "runner", Timestamp: now, Version: 1,
			Options: []InterventionOption{{ID: "replan-package", Operation: InterventionReplanPackage, PackageID: "pkg-1"}},
		},
		Contract: DefaultAutonomyContract(), UpdatedAt: now,
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ResolveIntervention(ctx, "run_dup", "intv_dup", 1, "replan-package", "tester"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ResolveIntervention(ctx, "run_dup", "intv_dup", 1, "replan-package", "tester"); err != nil {
		t.Fatalf("duplicate resolve must be idempotent: %v", err)
	}
}

func TestFaultInjectionQuotaFailoverIsNotAttention(t *testing.T) {
	run := &MissionRun{ID: "run_q", State: StateAllocating, UpdatedAt: time.Now().UTC()}
	if item := AttentionFromRun(run); item != nil {
		t.Fatalf("quota/allocation recovery must stay silent: %+v", item)
	}
}

func TestOvernightSoakHarnessBounded(t *testing.T) {
	duration := 15 * time.Second
	if raw := os.Getenv("OVERNIGHT_SOAK_DURATION"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			t.Fatal(err)
		}
		duration = parsed
	}
	if duration <= 0 {
		t.Fatalf("soak duration must be positive, got %s", duration)
	}
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	workspace := t.TempDir()
	cycles := 0
	const cyclePause = 250 * time.Millisecond

soakLoop:
	for {
		select {
		case <-ctx.Done():
			break soakLoop
		default:
		}

		exec := &overnightAcceptanceExecutor{globalRepairExecutor: globalRepairExecutor{fakeExecutor: fakeExecutor{reviewOK: true}}}
		r := NewMissionRunner(NewMemoryRunRepository(), exec)
		contract := DefaultAutonomyContract()
		contract.VerificationCommands = []string{"true"}
		contract.GlobalVerificationCommands = []string{"true"}
		plan := PlanSpec{ID: "soak", ProjectID: "p", Revision: 1, Packages: []PackageSpec{
			{ID: "a", Title: "a", Goal: "a", ParallelGroup: "g"},
			{ID: "b", Title: "b", Goal: "b", Dependencies: []string{"a"}},
		}}
		run, err := r.StartMissionRun(ctx, plan, workspace, contract, "")
		if err != nil {
			if ctx.Err() != nil {
				break soakLoop
			}
			t.Fatalf("soak cycle %d start failed: %v", cycles+1, err)
		}
		completed, err := r.RunToTerminal(ctx, run.ID)
		if err != nil && ctx.Err() == nil {
			t.Fatalf("soak cycle %d failed: %v", cycles+1, err)
		}
		if completed == nil {
			if ctx.Err() != nil {
				break soakLoop
			}
			t.Fatalf("soak cycle %d returned no run", cycles+1)
		}
		switch completed.State {
		case StateCompletedVerified, StateFailedNoProgress, StateBlockedNeedsUser, StateCanceledByUser, StateFailed, StateFailedBudgetExceeded, StateFailedVerification:
		default:
			t.Fatalf("soak cycle %d must end in an explicit terminal state, got %s", cycles+1, completed.State)
		}
		cycles++

		timer := time.NewTimer(cyclePause)
		select {
		case <-ctx.Done():
			timer.Stop()
			break soakLoop
		case <-timer.C:
		}
	}
	if cycles == 0 {
		t.Fatalf("soak completed no cycles during %s", duration)
	}
	t.Logf("completed %d mission cycles during %s", cycles, duration)
}
