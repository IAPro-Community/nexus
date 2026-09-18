package runner

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type interventionSideEffectExecutor struct {
	executed int
}

func (e *interventionSideEffectExecutor) Allocate(context.Context, *MissionRun, *PackageRun) (AllocationResult, error) {
	return AllocationResult{AgentID: "agent-1", Workspace: "."}, nil
}

func (e *interventionSideEffectExecutor) Compile(context.Context, *MissionRun, *PackageRun) (PromptArtifact, error) {
	return PromptArtifact{VersionID: "prompt-1", Content: "resume"}, nil
}

func (e *interventionSideEffectExecutor) Execute(context.Context, *MissionRun, *PackageRun, string) (ExecutionResult, error) {
	e.executed++
	return ExecutionResult{RuntimeID: fmt.Sprintf("runtime-%d", e.executed)}, nil
}

func (e *interventionSideEffectExecutor) Review(context.Context, *MissionRun, *PackageRun) (ReviewVerdict, error) {
	return ReviewVerdict{Approved: true, ReviewerAgentID: "reviewer-1", ReviewedAt: time.Now().UTC()}, nil
}

func TestResolveIntervention_DoesNotRedispatchUnknownExternalOutcome(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &interventionSideEffectExecutor{executed: 1}
	r := NewMissionRunner(repo, exec)
	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run-unknown-outcome", State: StateBlockedNeedsUser,
		Contract: AutonomyContract{MaxTotalIterations: 10, MaxRetries: 2, MaxNoProgress: 2},
		PackageRuns: []PackageRun{{
			ID: "pkg-a", PackageID: "pkg-a", State: StateExecuting,
			CompiledPrompt: "already compiled", DispatchID: "dispatch-a", DispatchState: DispatchIntent,
		}},
		NeedsHuman: &HumanIntervention{
			ID: "intervention-a", Version: 1, ReasonCode: "DISPATCH_OUTCOME_UNKNOWN",
			Summary: "provider outcome is unknown", Question: "Reconcile provider outcome?",
			MissionID: "run-unknown-outcome", TaskID: "pkg-a", Source: "runner", Timestamp: now,
			Options: []InterventionOption{{ID: "confirm-external-completion", Operation: InterventionConfirmExternalOutcome, PackageID: "pkg-a"}},
		},
		UpdatedAt: now,
	}
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}

	resolved, err := r.ResolveIntervention(context.Background(), run.ID, "intervention-a", 1, "confirm-external-completion", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		duplicate, duplicateErr := r.ResolveIntervention(context.Background(), run.ID, "intervention-a", 1, "confirm-external-completion", "retrying-client")
		if duplicateErr != nil || duplicate.NeedsHuman.Resolution.IdempotencyKey != resolved.NeedsHuman.Resolution.IdempotencyKey {
			t.Fatalf("identical decision attempt %d was not idempotent: run=%+v err=%v", attempt+2, duplicate, duplicateErr)
		}
	}
	if _, _, err := r.ExecuteNextStep(context.Background(), resolved.ID); err != nil {
		t.Fatal(err)
	}
	if exec.executed != 1 {
		t.Fatalf("external dispatch was repeated after human resolution: count=%d", exec.executed)
	}
}

func TestExecuteNextStep_DoesNotRedispatchCompletedDispatchCheckpoint(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &interventionSideEffectExecutor{}
	r := NewMissionRunner(repo, exec)
	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run-completed-dispatch", State: StateExecuting,
		Contract: AutonomyContract{MaxTotalIterations: 10},
		PackageRuns: []PackageRun{{
			ID: "pkg", PackageID: "pkg", State: StateExecuting,
			DispatchState: DispatchCompleted, DispatchID: "dispatch-known", AssignedRuntime: "runtime-known",
		}},
		UpdatedAt: now,
	}
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	updated, _, err := r.ExecuteNextStep(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.PackageRuns[0].State != StateTesting || exec.executed != 0 {
		t.Fatalf("completed dispatch was re-executed instead of resumed at testing: state=%s count=%d", updated.PackageRuns[0].State, exec.executed)
	}
}

func TestExecuteNextStep_FailsClosedForUnknownDispatchWithoutIdentity(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &interventionSideEffectExecutor{}
	r := NewMissionRunner(repo, exec)
	run := &MissionRun{
		ID: "run-unknown-without-id", State: StateExecuting,
		Contract:    AutonomyContract{MaxTotalIterations: 10},
		PackageRuns: []PackageRun{{ID: "pkg", PackageID: "pkg", State: StateExecuting, DispatchState: DispatchUnknownExternalOutcome}},
		UpdatedAt:   time.Now().UTC(),
	}
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	updated, _, err := r.ExecuteNextStep(context.Background(), run.ID)
	if !errors.Is(err, ErrDispatchOutcomeUnknown) || updated.State != StateBlockedNeedsUser || updated.NeedsHuman == nil || exec.executed != 0 {
		t.Fatalf("unknown dispatch without identity was not fail-closed: state=%s intervention=%+v count=%d err=%v", updated.State, updated.NeedsHuman, exec.executed, err)
	}
}

func TestResolveIntervention_Success(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()

	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run_1", ProjectID: "proj_1", State: StateBlockedNeedsUser,
		PackageRuns: []PackageRun{{ID: "pkg_1", PackageID: "pkg_1", State: StateExecuting, DispatchState: DispatchUnknownExternalOutcome, DispatchID: "dispatch-1"}},
		NeedsHuman: &HumanIntervention{
			ID: "intv_1", ReasonCode: "DISPATCH_OUTCOME_UNKNOWN", Summary: "unknown",
			Question: "Inspect?", MissionID: "run_1", TaskID: "pkg_1",
			Source: "runner", Timestamp: now, Version: 1,
			Options: []InterventionOption{{ID: "confirm-external-completion", Operation: InterventionConfirmExternalOutcome, PackageID: "pkg_1"}},
		},
		UpdatedAt: now,
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}

	resolved, err := r.ResolveIntervention(ctx, "run_1", "intv_1", 1, "confirm-external-completion", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.State != StateExecuting {
		t.Fatalf("expected EXECUTING, got %s", resolved.State)
	}
	if !resolved.NeedsHuman.Resolved {
		t.Fatal("expected intervention to be resolved")
	}
	if resolved.NeedsHuman.ResolutionDecision != "confirm-external-completion" {
		t.Fatalf("expected confirm option, got %s", resolved.NeedsHuman.ResolutionDecision)
	}
	if resolved.NeedsHuman.ResolutionChosen != string(InterventionConfirmExternalOutcome) {
		t.Fatalf("expected confirm operation, got %s", resolved.NeedsHuman.ResolutionChosen)
	}
	if resolved.NeedsHuman.ResolvedAt == nil {
		t.Fatal("expected ResolvedAt to be set")
	}
}

func TestResolveIntervention_Idempotent(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()

	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run_2", State: StateBlockedNeedsUser,
		PackageRuns: []PackageRun{{ID: "pkg_2", PackageID: "pkg_2", State: StateFailed, DispatchState: DispatchFailedBeforeDispatch}},
		NeedsHuman: &HumanIntervention{
			ID: "intv_2", ReasonCode: "NO_PROGRESS", Summary: "failed",
			Question: "Choose strategy?", MissionID: "run_2", TaskID: "pkg_2",
			Source: "runner", Timestamp: now, Version: 1,
			Options: []InterventionOption{{ID: "replan-package", Operation: InterventionReplanPackage, PackageID: "pkg_2"}},
		},
		UpdatedAt: now,
	}
	repo.SaveRun(ctx, run)

	// First resolution
	resolved1, err := r.ResolveIntervention(ctx, "run_2", "intv_2", 1, "replan-package", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if resolved1.State != StateExecuting {
		t.Fatalf("first: expected EXECUTING, got %s", resolved1.State)
	}

	// Simulate that the run was saved and moved to executing
	repo.SaveRun(ctx, resolved1)

	// Load back - the intervention is now resolved
	saved, _ := repo.GetRun(ctx, "run_2")
	if !saved.NeedsHuman.Resolved {
		t.Fatal("intervention should be resolved")
	}
}

func TestResolveIntervention_ConflictingDecisionIsRejected(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run-conflict", State: StateBlockedNeedsUser,
		PackageRuns: []PackageRun{{ID: "pkg", PackageID: "pkg", State: StateFailed, DispatchState: DispatchFailedBeforeDispatch}},
		NeedsHuman: &HumanIntervention{
			ID: "intv-conflict", Version: 3, TaskID: "pkg", MissionID: "run-conflict", Timestamp: now,
			Options: []InterventionOption{
				{ID: "replan-package", Operation: InterventionReplanPackage, PackageID: "pkg"},
				{ID: "retry-safe-package", Operation: InterventionRetrySafePackage, PackageID: "pkg"},
			},
		},
	}
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	resolved, err := r.ResolveIntervention(context.Background(), run.ID, "intv-conflict", 3, "replan-package", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.ResolveIntervention(context.Background(), run.ID, "intv-conflict", 3, "retry-safe-package", "user-1")
	if !errors.Is(err, ErrInterventionAlreadyResolved) {
		t.Fatalf("expected semantic conflict, got %v", err)
	}
	saved, err := repo.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.NeedsHuman.Resolution.OptionID != resolved.NeedsHuman.Resolution.OptionID {
		t.Fatal("conflicting request replaced the canonical resolution")
	}
}

func TestResolveIntervention_StaleRequestHasNoBusinessMutation(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run-stale", State: StateBlockedNeedsUser,
		PackageRuns: []PackageRun{{ID: "pkg", PackageID: "pkg", State: StateFailed, DispatchState: DispatchFailedBeforeDispatch}},
		NeedsHuman: &HumanIntervention{
			ID: "intv-stale", Version: 2, TaskID: "pkg", MissionID: "run-stale", Timestamp: now,
			Options: []InterventionOption{{ID: "replan-package", Operation: InterventionReplanPackage, PackageID: "pkg"}},
		},
	}
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	_, err := r.ResolveIntervention(context.Background(), run.ID, "intv-stale", 1, "replan-package", "user-1")
	if !errors.Is(err, ErrStaleIntervention) {
		t.Fatalf("expected stale intervention, got %v", err)
	}
	saved, err := repo.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.State != StateBlockedNeedsUser || saved.NeedsHuman.Resolved || saved.PackageRuns[0].DispatchState != DispatchFailedBeforeDispatch {
		t.Fatalf("stale request changed business state: %+v", saved)
	}
}

func TestResolveIntervention_StaleVersionAfterResolutionIsRejected(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()
	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run-stale-resolved", State: StateBlockedNeedsUser,
		PackageRuns: []PackageRun{{ID: "pkg", PackageID: "pkg", State: StateFailed, DispatchState: DispatchFailedBeforeDispatch}},
		NeedsHuman: &HumanIntervention{
			ID: "intv", Version: 2, MissionID: "run-stale-resolved", TaskID: "pkg", Timestamp: now,
			Options: []InterventionOption{{ID: "replan-package", Operation: InterventionReplanPackage, PackageID: "pkg"}},
		},
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ResolveIntervention(ctx, run.ID, "intv", 2, "replan-package", "user"); err != nil {
		t.Fatal(err)
	}
	_, err := r.ResolveIntervention(ctx, run.ID, "intv", 1, "replan-package", "user")
	if !errors.Is(err, ErrStaleIntervention) {
		t.Fatalf("expected stale version after resolution, got %v", err)
	}
}

func TestResolveIntervention_RejectsOptionOutsideAutonomyContract(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()
	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run-policy", State: StateBlockedNeedsUser,
		Contract:    AutonomyContract{AllowedHumanOperations: []InterventionOperation{InterventionRetrySafePackage}},
		PackageRuns: []PackageRun{{ID: "pkg", PackageID: "pkg", State: StateExecuting, DispatchState: DispatchUnknownExternalOutcome, DispatchID: "dispatch-unknown"}},
		NeedsHuman: &HumanIntervention{
			ID: "intv-policy", Version: 1, MissionID: "run-policy", TaskID: "pkg", Timestamp: now,
			Options: []InterventionOption{{ID: "confirm-external-completion", Operation: InterventionConfirmExternalOutcome, PackageID: "pkg"}},
		},
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	_, err := r.ResolveIntervention(ctx, run.ID, "intv-policy", 1, "confirm-external-completion", "user")
	if !errors.Is(err, ErrInterventionPolicyDenied) {
		t.Fatalf("expected autonomy policy denial, got %v", err)
	}
	saved, err := repo.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.State != StateBlockedNeedsUser || saved.NeedsHuman.Resolved || saved.PackageRuns[0].DispatchID != "dispatch-unknown" {
		t.Fatalf("policy rejection mutated business state: %+v", saved)
	}
}

func TestResolveIntervention_PreservesParallelAndCompletedPackageIdentity(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()
	now := time.Now().UTC()
	receipt := &WorkReceipt{ID: "receipt-c", RunID: "run-parallel", StepID: "pkg-c", Status: "VERIFIED", CompletedAt: now}
	run := &MissionRun{
		ID: "run-parallel", State: StateBlockedNeedsUser,
		PackageRuns: []PackageRun{
			{ID: "a", PackageID: "pkg-a", State: StateExecuting, DispatchState: DispatchUnknownExternalOutcome, DispatchID: "dispatch-a"},
			{ID: "b", PackageID: "pkg-b", State: StateExecuting, DispatchState: DispatchUnknownExternalOutcome, DispatchID: "dispatch-b"},
			{ID: "c", PackageID: "pkg-c", State: StateCompletedVerified, DispatchState: DispatchCompleted, DispatchID: "dispatch-c", WorkReceipt: receipt},
			{ID: "d", PackageID: "pkg-d", State: StateReady},
		},
		NeedsHuman: &HumanIntervention{
			ID: "intv-parallel", Version: 1, MissionID: "run-parallel", TaskID: "pkg-a", Timestamp: now,
			Options: []InterventionOption{{ID: "confirm-external-completion", Operation: InterventionConfirmExternalOutcome, PackageID: "pkg-a"}},
		},
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	resolved, err := r.ResolveIntervention(ctx, run.ID, "intv-parallel", 1, "confirm-external-completion", "user")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.PackageRuns[0].DispatchState != DispatchUnknownExternalOutcome || resolved.PackageRuns[0].DispatchID != "dispatch-a" {
		t.Fatalf("target package dispatch identity changed: %+v", resolved.PackageRuns[0])
	}
	if resolved.PackageRuns[1].DispatchState != DispatchUnknownExternalOutcome || resolved.PackageRuns[1].DispatchID != "dispatch-b" {
		t.Fatalf("unrelated executing package was reset: %+v", resolved.PackageRuns[1])
	}
	if resolved.PackageRuns[2].State != StateCompletedVerified || resolved.PackageRuns[2].DispatchID != "dispatch-c" || resolved.PackageRuns[2].WorkReceipt == nil || resolved.PackageRuns[2].WorkReceipt.ID != receipt.ID {
		t.Fatalf("completed package evidence was not preserved: %+v", resolved.PackageRuns[2])
	}
	if resolved.PackageRuns[3].State != StateReady {
		t.Fatalf("unrelated waiting package changed: %+v", resolved.PackageRuns[3])
	}
}

func TestResolveIntervention_AllowsRetryOnlyAfterProvenPreDispatchFailure(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &interventionSideEffectExecutor{}
	r := NewMissionRunner(repo, exec)
	ctx := context.Background()
	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run-safe-retry", State: StateBlockedNeedsUser,
		Contract:    DefaultAutonomyContract(),
		PackageRuns: []PackageRun{{ID: "pkg", PackageID: "pkg", State: StateFailed, DispatchState: DispatchFailedBeforeDispatch}},
		NeedsHuman: &HumanIntervention{
			ID: "intv-safe-retry", Version: 1, MissionID: "run-safe-retry", TaskID: "pkg", Timestamp: now,
			Options: []InterventionOption{{ID: "retry-safe-step", Operation: InterventionRetrySafePackage, PackageID: "pkg"}},
		},
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	resolved, err := r.ResolveIntervention(ctx, run.ID, "intv-safe-retry", 1, "retry-safe-step", "user")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.PackageRuns[0].DispatchState != DispatchNone || resolved.PackageRuns[0].State != StateExecuting {
		t.Fatalf("safe retry did not reset only proven pre-dispatch failure: %+v", resolved.PackageRuns[0])
	}
	if _, _, err := r.ExecuteNextStep(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	if exec.executed != 1 {
		t.Fatalf("expected exactly one safe retry side effect, got %d", exec.executed)
	}
}

func TestExecuteNextStepUsesFailedNoProgressWithoutHumanBlocker(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()
	run := &MissionRun{ID: "run-no-progress", State: StateExecuting, Contract: AutonomyContract{MaxTotalIterations: 10}, UpdatedAt: time.Now().UTC()}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	updated, _, err := r.ExecuteNextStep(ctx, run.ID)
	if err == nil {
		t.Fatal("expected no-progress error")
	}
	if updated.State != StateFailedNoProgress || updated.NeedsHuman != nil {
		t.Fatalf("no-progress failure became an actionable human blocker: %+v", updated)
	}
}

func TestResolveIntervention_StaleVersionRejected(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()

	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run_3", State: StateBlockedNeedsUser,
		NeedsHuman: &HumanIntervention{
			ID: "intv_3", ReasonCode: "SECURITY", Summary: "security",
			Question: "Approve?", MissionID: "run_3", TaskID: "pkg_3",
			Source: "runner", Timestamp: now, Version: 2,
		},
		UpdatedAt: now,
	}
	repo.SaveRun(ctx, run)

	// Try with stale version
	_, err := r.ResolveIntervention(ctx, "run_3", "intv_3", 1, "approve", "")
	if err == nil {
		t.Fatal("expected error for stale version")
	}
}

func TestResolveIntervention_StaleInterventionIDRejected(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()

	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run_4", State: StateBlockedNeedsUser,
		NeedsHuman: &HumanIntervention{
			ID: "intv_4", ReasonCode: "SECURITY", Summary: "security",
			Question: "Approve?", MissionID: "run_4", TaskID: "pkg_4",
			Source: "runner", Timestamp: now, Version: 1,
		},
		UpdatedAt: now,
	}
	repo.SaveRun(ctx, run)

	// Try with wrong intervention ID
	_, err := r.ResolveIntervention(ctx, "run_4", "intv_wrong", 1, "approve", "")
	if err == nil {
		t.Fatal("expected error for wrong intervention ID")
	}
}

func TestResolveIntervention_NotBlockedRejected(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()

	run := &MissionRun{ID: "run_5", State: StateExecuting, UpdatedAt: time.Now().UTC()}
	repo.SaveRun(ctx, run)

	_, err := r.ResolveIntervention(ctx, "run_5", "intv_5", 1, "approve", "")
	if err == nil {
		t.Fatal("expected error for non-blocked mission")
	}
}

func TestResolveIntervention_NilNeedsHumanRejected(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()

	run := &MissionRun{ID: "run_6", State: StateBlockedNeedsUser, UpdatedAt: time.Now().UTC()}
	repo.SaveRun(ctx, run)

	_, err := r.ResolveIntervention(ctx, "run_6", "intv_6", 1, "approve", "")
	if err == nil {
		t.Fatal("expected error for nil NeedsHuman")
	}
}

func TestResolveIntervention_ResetsStaleDispatchIntent(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()

	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run_dispatch", State: StateBlockedNeedsUser,
		Contract: AutonomyContract{MaxRetries: 3, MaxTotalIterations: 120, MaxNoProgress: 2},
		PackageRuns: []PackageRun{
			{
				ID: "pkg_1", PackageID: "pkg_1", State: StateExecuting,
				AssignedAgent: "agent_1", DispatchState: DispatchIntent, DispatchID: "dispatch_old",
			},
		},
		NeedsHuman: &HumanIntervention{
			ID: "intv_dispatch", ReasonCode: "DISPATCH_OUTCOME_UNKNOWN", Summary: "unknown",
			Question: "Inspect?", MissionID: "run_dispatch", TaskID: "pkg_1",
			Source: "runner", Timestamp: now, Version: 1,
			Options: []InterventionOption{{ID: "confirm-external-completion", Operation: InterventionConfirmExternalOutcome, PackageID: "pkg_1"}},
		},
		UpdatedAt: now,
	}
	repo.SaveRun(ctx, run)

	resolved, err := r.ResolveIntervention(ctx, "run_dispatch", "intv_dispatch", 1, "confirm-external-completion", "user-1")
	if err != nil {
		t.Fatal(err)
	}

	if resolved.PackageRuns[0].DispatchState != DispatchUnknownExternalOutcome || resolved.PackageRuns[0].DispatchID != "dispatch_old" {
		t.Fatalf("resolution must preserve unknown dispatch identity, got state=%s id=%s", resolved.PackageRuns[0].DispatchState, resolved.PackageRuns[0].DispatchID)
	}
}

func TestBlockNeedsHuman_GeneratesIDAndVersion(t *testing.T) {
	r := NewMissionRunner(NewMemoryRunRepository(), &fakeExecutor{})
	run := &MissionRun{ID: "run_7"}
	pkg := &PackageRun{PackageID: "pkg_7"}

	r.blockNeedsHuman(run, pkg, "TEST_CODE", "test summary", "test question?", []string{"action1"})

	if run.NeedsHuman.ID == "" {
		t.Fatal("expected intervention ID to be generated")
	}
	if run.NeedsHuman.Version != 1 {
		t.Fatalf("expected version 1, got %d", run.NeedsHuman.Version)
	}
	if run.NeedsHuman.Resolved {
		t.Fatal("expected intervention to not be resolved")
	}
}

func TestSaveRunRejectsBlockedMissionWithoutActionableIntervention(t *testing.T) {
	r := NewMissionRunner(NewMemoryRunRepository(), &fakeExecutor{})
	err := r.saveRun(context.Background(), &MissionRun{ID: "blocked-without-question", State: StateBlockedNeedsUser})
	if err == nil {
		t.Fatal("expected blocked mission without intervention to be rejected")
	}
}
