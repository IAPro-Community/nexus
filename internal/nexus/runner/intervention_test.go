package runner

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type interventionSideEffectExecutor struct {
	executed                 int
	allocated                int
	compiled                 int
	coordinationRequests     []CoordinationRequest
	coordinationReplaySafe   bool
	coordinationUnsafeReason string
}

func (e *interventionSideEffectExecutor) Allocate(context.Context, *MissionRun, *PackageRun) (AllocationResult, error) {
	e.allocated++
	return AllocationResult{AgentID: "agent-1", Workspace: "."}, nil
}

func (e *interventionSideEffectExecutor) Compile(context.Context, *MissionRun, *PackageRun) (PromptArtifact, error) {
	e.compiled++
	return PromptArtifact{VersionID: "prompt-1", Content: "resume"}, nil
}

func (e *interventionSideEffectExecutor) Execute(context.Context, *MissionRun, *PackageRun, string) (ExecutionResult, error) {
	e.executed++
	return ExecutionResult{
		RuntimeID:                fmt.Sprintf("runtime-%d", e.executed),
		CoordinationRequests:     e.coordinationRequests,
		CoordinationReplaySafe:   e.coordinationReplaySafe,
		CoordinationUnsafeReason: e.coordinationUnsafeReason,
	}, nil
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
	if got := resolved.PackageRuns[0]; got.State != StateTesting || got.DispatchState != DispatchCompleted || got.WorkReceipt != nil {
		t.Fatalf("human reconciliation must resume verification without fabricating a verified receipt: %+v", got)
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

func TestResolveInterventionDoesNotReplayCompletedUnsafeCoordination(t *testing.T) {
	tests := []struct {
		name                   string
		coordinationRequests   []CoordinationRequest
		coordinationUnsafe     string
		wantInterventionReason string
	}{
		{
			name:                   "coordination replay was not proven safe",
			coordinationRequests:   []CoordinationRequest{{Operation: CoordinationNeedInfo, Subject: "architecture.auth_endpoint", Objective: "confirm endpoint behavior"}},
			wantInterventionReason: "COORDINATION_REPLAY_UNPROVEN",
		},
		{
			name:                   "coordination followed an unsafe side effect",
			coordinationUnsafe:     "external mutation occurred before coordination was requested",
			wantInterventionReason: "UNSAFE_COORDINATION_AFTER_SIDE_EFFECT",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			repo := NewMemoryRunRepository()
			exec := &interventionSideEffectExecutor{
				coordinationRequests:     test.coordinationRequests,
				coordinationUnsafeReason: test.coordinationUnsafe,
			}
			r := NewMissionRunner(repo, exec)
			now := time.Now().UTC()
			run := &MissionRun{
				ID: "run-unsafe-coordination", State: StateExecuting,
				Contract: AutonomyContract{MaxTotalIterations: 10, RequireVerification: false},
				PackageRuns: []PackageRun{{
					ID: "pkg", PackageID: "pkg", State: StateExecuting,
					AssignedAgent: "agent-1", Workspace: t.TempDir(), CompiledPrompt: "perform one side effect",
				}},
				StartedAt: now, UpdatedAt: now,
			}
			if err := repo.SaveRun(ctx, run); err != nil {
				t.Fatal(err)
			}

			blocked, _, err := r.ExecuteNextStep(ctx, run.ID)
			if err == nil {
				t.Fatal("unsafe coordination must create a human intervention")
			}
			if blocked.NeedsHuman == nil || blocked.NeedsHuman.ReasonCode != test.wantInterventionReason {
				t.Fatalf("unexpected intervention: %+v", blocked.NeedsHuman)
			}
			if len(blocked.NeedsHuman.Options) != 1 || blocked.NeedsHuman.Options[0].ID != "verify-completed-without-replay" {
				t.Fatalf("completed unsafe dispatch must offer verification without replay: %+v", blocked.NeedsHuman.Options)
			}
			if got := blocked.PackageRuns[0]; got.DispatchState != DispatchCompleted || got.DispatchID == "" || !got.ReplayProhibited {
				t.Fatalf("completed side effect identity was not preserved: %+v", got)
			}
			if exec.executed != 1 {
				t.Fatalf("initial side effect count=%d, want 1", exec.executed)
			}

			_, err = r.ResolveIntervention(ctx, run.ID, blocked.NeedsHuman.ID, blocked.NeedsHuman.Version, "replan-package", "user-1")
			if !errors.Is(err, ErrInvalidInterventionOption) {
				t.Fatalf("replanning must not be offered for a completed dispatch, got %v", err)
			}
			if err := applyInterventionOption(&blocked.PackageRuns[0], InterventionOption{ID: "replan-package", Operation: InterventionReplanPackage, PackageID: "pkg"}); !errors.Is(err, ErrInterventionPolicyDenied) {
				t.Fatalf("forged replan of a completed dispatch must fail closed, got %v", err)
			}
			saved, err := repo.GetRun(ctx, run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if saved.State != StateBlockedNeedsUser || saved.NeedsHuman.Resolved || saved.PackageRuns[0].DispatchState != DispatchCompleted || saved.PackageRuns[0].DispatchID != blocked.PackageRuns[0].DispatchID {
				t.Fatalf("denied replay changed durable state: %+v", saved)
			}
			if exec.executed != 1 {
				t.Fatalf("denied replay repeated the side effect: count=%d", exec.executed)
			}

			resolved, err := r.ResolveIntervention(ctx, run.ID, blocked.NeedsHuman.ID, blocked.NeedsHuman.Version, "verify-completed-without-replay", "user-1")
			if err != nil {
				t.Fatalf("verify completed work: %v", err)
			}
			if got := resolved.PackageRuns[0]; got.State != StateTesting || got.DispatchState != DispatchCompleted || got.DispatchID != blocked.PackageRuns[0].DispatchID {
				t.Fatalf("verification must retain completed dispatch identity: %+v", got)
			}
			if _, _, err := r.ExecuteNextStep(ctx, run.ID); err != nil {
				t.Fatalf("run verification after explicit confirmation: %v", err)
			}
			if exec.executed != 1 {
				t.Fatalf("verification re-executed provider side effect: count=%d", exec.executed)
			}
		})
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
	if resolved.PackageRuns[0].State != StateTesting || resolved.PackageRuns[0].DispatchState != DispatchCompleted {
		t.Fatalf("confirmed external completion must continue through verification: %+v", resolved.PackageRuns[0])
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
	if resolved1.State != StateExecuting || resolved1.PackageRuns[0].State != StateAllocating {
		t.Fatalf("replan must resume the run at validated allocation, got run=%s package=%s", resolved1.State, resolved1.PackageRuns[0].State)
	}

	// Simulate that the run was saved and moved to executing
	repo.SaveRun(ctx, resolved1)

	// Load back - the intervention is now resolved
	saved, _ := repo.GetRun(ctx, "run_2")
	if !saved.NeedsHuman.Resolved {
		t.Fatal("intervention should be resolved")
	}
}

func TestResolveIntervention_ReplanReallocatesAndRecompilesBeforeRetry(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRunRepository()
	exec := &interventionSideEffectExecutor{}
	r := NewMissionRunner(repo, exec)
	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run-replan-pipeline", State: StateBlockedNeedsUser,
		Contract: AutonomyContract{MaxTotalIterations: 20, RequireVerification: false},
		PackageRuns: []PackageRun{{
			ID: "pkg", PackageID: "pkg", State: StateFailed, DispatchState: DispatchFailedBeforeDispatch,
			AssignedAgent: "stale-agent", AssignedRuntime: "stale-runtime", Workspace: "stale-workspace",
			PromptVersionID: "stale-prompt", CompiledPrompt: "stale prompt",
		}},
		NeedsHuman: &HumanIntervention{
			ID: "intv-replan-pipeline", Version: 1, MissionID: "run-replan-pipeline", TaskID: "pkg", Timestamp: now,
			Options: []InterventionOption{{ID: "replan-package", Operation: InterventionReplanPackage, PackageID: "pkg"}},
		},
		UpdatedAt: now,
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	resolved, err := r.ResolveIntervention(ctx, run.ID, "intv-replan-pipeline", 1, "replan-package", "user")
	if err != nil {
		t.Fatal(err)
	}
	if got := resolved.PackageRuns[0]; got.State != StateAllocating || got.AssignedAgent != "stale-agent" || got.AssignedRuntime != "" || got.Workspace != "stale-workspace" || got.CompiledPrompt != "" || got.PromptVersionID != "" {
		t.Fatalf("replan did not retain the working copy while clearing stale execution state: %+v", got)
	}
	for i := 0; i < 3; i++ {
		if _, _, err := r.ExecuteNextStep(ctx, run.ID); err != nil {
			t.Fatalf("pipeline step %d: %v", i+1, err)
		}
	}
	if exec.allocated != 1 || exec.compiled != 1 || exec.executed != 1 {
		t.Fatalf("replan must allocate, compile, and only then execute: allocate=%d compile=%d execute=%d", exec.allocated, exec.compiled, exec.executed)
	}
}

func TestResolveIntervention_DefersPackageUntilDependenciesAreVerified(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &interventionSideEffectExecutor{})
	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run-replan-dependency", State: StateBlockedNeedsUser,
		Contract: AutonomyContract{MaxTotalIterations: 20, RequireVerification: false},
		PackageRuns: []PackageRun{
			{ID: "target", PackageID: "target", State: StateFailed, DispatchState: DispatchFailedBeforeDispatch, Dependencies: []string{"dependency"}},
			{ID: "dependency", PackageID: "dependency", State: StateReady},
		},
		NeedsHuman: &HumanIntervention{
			ID: "intv-replan-dependency", Version: 1, MissionID: "run-replan-dependency", TaskID: "target", Timestamp: now,
			Options: []InterventionOption{{ID: "replan-package", Operation: InterventionReplanPackage, PackageID: "target"}},
		},
		UpdatedAt: now,
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	resolved, err := r.ResolveIntervention(ctx, run.ID, "intv-replan-dependency", 1, "replan-package", "user")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.PackageRuns[0].State != StatePending {
		t.Fatalf("replanned package ran before dependencies were verified: %+v", resolved.PackageRuns[0])
	}
	if _, _, err := r.ExecuteNextStep(ctx, run.ID); err != nil {
		t.Fatalf("dependency should continue first: %v", err)
	}
	progress, err := repo.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if progress.PackageRuns[0].State != StatePending || progress.PackageRuns[1].State != StateAllocating {
		t.Fatalf("dependency ordering was not preserved: target=%s dependency=%s", progress.PackageRuns[0].State, progress.PackageRuns[1].State)
	}
	progress.PackageRuns[1].State = StateVerified
	if err := repo.SaveRun(ctx, progress); err != nil {
		t.Fatal(err)
	}
	resumed, _, err := r.ExecuteNextStep(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.PackageRuns[0].State != StateAllocating {
		t.Fatalf("package did not resume allocation after dependency verification: %+v", resumed.PackageRuns[0])
	}
}

func TestCompletedDispatchVerificationFailureBlocksWithoutReplay(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRunRepository()
	exec := &interventionSideEffectExecutor{}
	r := NewMissionRunner(repo, exec)
	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run-completed-verification-failure", State: StateBlockedNeedsUser,
		Contract: AutonomyContract{MaxTotalIterations: 10, RequireVerification: true, VerificationCommands: []string{"false"}},
		PackageRuns: []PackageRun{{
			ID: "pkg", PackageID: "pkg", State: StateTesting, DispatchState: DispatchCompleted,
			DispatchID: "dispatch-completed", AssignedRuntime: "runtime-completed", ReplayProhibited: true, Workspace: t.TempDir(),
		}},
		NeedsHuman: &HumanIntervention{
			ID: "intv-completed-verification-failure", Version: 1, MissionID: "run-completed-verification-failure", TaskID: "pkg", Timestamp: now,
			Options: []InterventionOption{{ID: "verify-completed-without-replay", Operation: InterventionVerifyCompletedWithoutReplay, PackageID: "pkg"}},
		},
		UpdatedAt: now,
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	resolved, err := r.ResolveIntervention(ctx, run.ID, "intv-completed-verification-failure", 1, "verify-completed-without-replay", "user")
	if err != nil {
		t.Fatal(err)
	}
	updated, _, err := r.ExecuteNextStep(ctx, resolved.ID)
	if err != nil {
		t.Fatalf("verification failure should become an actionable intervention: %v", err)
	}
	if updated.State != StateBlockedNeedsUser || updated.NeedsHuman == nil || updated.PackageRuns[0].DispatchState != DispatchCompleted || updated.PackageRuns[0].DispatchID != "dispatch-completed" || !updated.PackageRuns[0].ReplayProhibited {
		t.Fatalf("failed verification after completed dispatch must block and preserve dispatch identity: %+v", updated)
	}
	if len(updated.NeedsHuman.Options) != 1 || updated.NeedsHuman.Options[0].Operation != InterventionVerifyCompletedWithoutReplay {
		t.Fatalf("failed verification must offer only safe verification retry: %+v", updated.NeedsHuman.Options)
	}
	if exec.executed != 0 {
		t.Fatalf("verification failure re-executed provider side effect: count=%d", exec.executed)
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
	if resolved.PackageRuns[0].DispatchState != DispatchCompleted || resolved.PackageRuns[0].DispatchID != "dispatch-a" || resolved.PackageRuns[0].State != StateTesting {
		t.Fatalf("target package was not reconciled into verification: %+v", resolved.PackageRuns[0])
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
	if resolved.PackageRuns[0].DispatchState != DispatchNone || resolved.PackageRuns[0].State != StateAllocating {
		t.Fatalf("safe retry did not reset only proven pre-dispatch failure: %+v", resolved.PackageRuns[0])
	}
	for i := 0; i < 3; i++ {
		if _, _, err := r.ExecuteNextStep(ctx, run.ID); err != nil {
			t.Fatalf("safe retry stage %d: %v", i+1, err)
		}
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

	if resolved.PackageRuns[0].DispatchState != DispatchCompleted || resolved.PackageRuns[0].DispatchID != "dispatch_old" || resolved.PackageRuns[0].State != StateTesting {
		t.Fatalf("resolution must reconcile the dispatch without skipping verification, got dispatch=%s state=%s id=%s", resolved.PackageRuns[0].DispatchState, resolved.PackageRuns[0].State, resolved.PackageRuns[0].DispatchID)
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
