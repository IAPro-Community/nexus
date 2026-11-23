package runner

import (
	"context"
	"errors"
	"testing"
	"time"
)

func maestroOwnedRun(t *testing.T, repo RunRepository, id string) *MissionRun {
	t.Helper()
	now := time.Now().UTC()
	run := &MissionRun{
		ID: id, ProjectID: "proj-1", State: StateBlockedNeedsUser, ExecutionOwner: ExecutionOwnerMaestro,
		PackageRuns: []PackageRun{{ID: "pkg_1", PackageID: "pkg_1", State: StateFailed, DispatchState: DispatchFailedBeforeDispatch}},
		NeedsHuman: &HumanIntervention{
			ID: "intv_1", ReasonCode: "NO_PROGRESS", Summary: "failed",
			Question: "Choose strategy?", MissionID: id, TaskID: "pkg_1",
			Source: "runner", Timestamp: now, Version: 1,
			Options: []InterventionOption{{ID: "replan-package", Operation: InterventionReplanPackage, PackageID: "pkg_1"}},
		},
		UpdatedAt: now,
	}
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatalf("seed maestro-owned run: %v", err)
	}
	return run
}

// coordinationOwnedRun seeds a non-terminal run that already owns one PENDING
// coordination request and its source package, so every coordination entry
// point would have real work to do if the owner guard did not fire first.
func coordinationOwnedRun(t *testing.T, repo RunRepository, id, owner string) *MissionRun {
	t.Helper()
	now := time.Now().UTC()
	run := &MissionRun{
		ID: id, ProjectID: "proj-1", State: StateExecuting, ExecutionOwner: owner,
		PackageRuns: []PackageRun{{
			ID: "pkg_1", PackageID: "pkg_1", Title: "source", Goal: "source goal",
			State: StateExecuting, Attempt: 1, StartedAt: now,
		}},
		CoordinationRequests: []CoordinationRequest{{
			ID: "coordreq_1", RunID: id, SourcePackageID: "pkg_1", SourceAgentID: "agent-1",
			Operation: CoordinationDelegate, Subject: "Investigate the regression",
			Objective: "Explain the failing verification", Status: CoordinationPending, CreatedAt: now,
		}},
		StartedAt: now, UpdatedAt: now,
	}
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatalf("seed coordination run: %v", err)
	}
	return run
}

func assertCoordinationUntouched(t *testing.T, repo RunRepository, runID string) {
	t.Helper()
	stored, err := repo.GetRun(context.Background(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.LeaseOwner != "" || stored.LeaseToken != "" {
		t.Fatalf("refused coordination call took a lease: owner=%q token=%q", stored.LeaseOwner, stored.LeaseToken)
	}
	if len(stored.CoordinationRequests) != 1 {
		t.Fatalf("refused coordination call changed request count: %d", len(stored.CoordinationRequests))
	}
	req := stored.CoordinationRequests[0]
	if req.Status != CoordinationPending || req.ResolvedAt != nil || req.ResultReceiptID != "" ||
		req.ResolutionSummary != "" || req.ChildPackageID != "" || req.MaterializedAt != nil {
		t.Fatalf("refused coordination call mutated request: status=%s child=%q summary=%q",
			req.Status, req.ChildPackageID, req.ResolutionSummary)
	}
	if len(stored.PackageRuns) != 1 {
		t.Fatalf("refused coordination call materialized a child package: %d packages", len(stored.PackageRuns))
	}
}

func TestMissionRunnerRefusesMaestroOwnedRunForCoordination(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()
	run := coordinationOwnedRun(t, repo, "run_maestro_coord", ExecutionOwnerMaestro)

	submission := CoordinationRequest{SourcePackageID: "pkg_1", Operation: CoordinationNeedInfo, Subject: "Which module owns this?"}
	if _, err := r.SubmitCoordinationRequest(ctx, run.ID, submission); !errors.Is(err, ErrExecutionOwnerNotLocal) {
		t.Fatalf("submit coordination must refuse maestro-owned run: %v", err)
	}
	assertCoordinationUntouched(t, repo, run.ID)

	if _, err := r.ResolveCoordinationRequest(ctx, run.ID, "coordreq_1", CoordinationResolved, "done", "receipt_1"); !errors.Is(err, ErrExecutionOwnerNotLocal) {
		t.Fatalf("resolve coordination must refuse maestro-owned run: %v", err)
	}
	assertCoordinationUntouched(t, repo, run.ID)

	if _, err := r.MaterializeCoordinationWork(ctx, run.ID, "coordreq_1"); !errors.Is(err, ErrExecutionOwnerNotLocal) {
		t.Fatalf("materialize coordination must refuse maestro-owned run: %v", err)
	}
	assertCoordinationUntouched(t, repo, run.ID)
}

func TestCoordinationFailsClosedForUnknownExecutionOwner(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()
	run := coordinationOwnedRun(t, repo, "run_unknown_owner", "external-orchestrator")

	submission := CoordinationRequest{SourcePackageID: "pkg_1", Operation: CoordinationNeedInfo, Subject: "Status?"}
	if _, err := r.SubmitCoordinationRequest(ctx, run.ID, submission); !errors.Is(err, ErrInvalidExecutionOwner) {
		t.Fatalf("submit coordination must fail closed on unknown owner: %v", err)
	}
	if _, err := r.ResolveCoordinationRequest(ctx, run.ID, "coordreq_1", CoordinationResolved, "done", ""); !errors.Is(err, ErrInvalidExecutionOwner) {
		t.Fatalf("resolve coordination must fail closed on unknown owner: %v", err)
	}
	if _, err := r.MaterializeCoordinationWork(ctx, run.ID, "coordreq_1"); !errors.Is(err, ErrInvalidExecutionOwner) {
		t.Fatalf("materialize coordination must fail closed on unknown owner: %v", err)
	}
	assertCoordinationUntouched(t, repo, run.ID)
}

func TestCoordinationStillWorksForNexusOwnedRun(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()
	run := coordinationOwnedRun(t, repo, "run_nexus_coord", ExecutionOwnerNexus)

	submission := CoordinationRequest{SourcePackageID: "pkg_1", Operation: CoordinationNeedInfo, Subject: "What changed?"}
	if _, err := r.SubmitCoordinationRequest(ctx, run.ID, submission); err != nil {
		t.Fatalf("local runner must accept NEXUS-owned coordination: %v", err)
	}
	if _, err := r.MaterializeCoordinationWork(ctx, run.ID, "coordreq_1"); err != nil {
		t.Fatalf("local runner must materialize NEXUS-owned coordination: %v", err)
	}
	if _, err := r.ResolveCoordinationRequest(ctx, run.ID, "coordreq_1", CoordinationRejected, "not needed", ""); err != nil {
		t.Fatalf("local runner must resolve NEXUS-owned coordination: %v", err)
	}
}

func TestMissionRunnerRefusesMaestroOwnedRunForEveryLocalMutation(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()
	run := maestroOwnedRun(t, repo, "run_maestro")

	if _, _, err := r.ExecuteNextStep(ctx, run.ID); !errors.Is(err, ErrExecutionOwnerNotLocal) {
		t.Fatalf("execute step must refuse maestro-owned run: %v", err)
	}
	if _, err := r.PauseRun(ctx, run.ID, "stop"); !errors.Is(err, ErrExecutionOwnerNotLocal) {
		t.Fatalf("pause must refuse maestro-owned run: %v", err)
	}
	if _, err := r.ResumeRun(ctx, run.ID); !errors.Is(err, ErrExecutionOwnerNotLocal) {
		t.Fatalf("resume must refuse maestro-owned run: %v", err)
	}
	if _, err := r.CancelRun(ctx, run.ID, "stop"); !errors.Is(err, ErrExecutionOwnerNotLocal) {
		t.Fatalf("cancel must refuse maestro-owned run: %v", err)
	}
	if _, _, err := r.ResolveInterventionWithOutcome(ctx, run.ID, "intv_1", 1, "replan-package", "user-1"); !errors.Is(err, ErrExecutionOwnerNotLocal) {
		t.Fatalf("intervention resolution must refuse maestro-owned run: %v", err)
	}

	stored, err := repo.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != StateBlockedNeedsUser || stored.NeedsHuman.Resolved {
		t.Fatalf("refused local mutation changed maestro-owned run: state=%s resolved=%v", stored.State, stored.NeedsHuman.Resolved)
	}
}

func TestStartMissionRunRefusesMaestroOwnedPlan(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	plan := durablePlan()
	plan.ExecutionOwner = ExecutionOwnerMaestro

	if _, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), DefaultAutonomyContract(), ""); !errors.Is(err, ErrExecutionOwnerNotLocal) {
		t.Fatalf("start must refuse maestro-owned plan: %v", err)
	}
	if runs, err := repo.ListRuns(context.Background()); err != nil {
		t.Fatal(err)
	} else if len(runs) != 0 {
		t.Fatalf("refused start persisted %d runs", len(runs))
	}
}

func TestExecutionOwnerIsImmutableAndDefaultsToNexus(t *testing.T) {
	repo := NewMemoryRunRepository()
	ctx := context.Background()
	run := &MissionRun{ID: "run_owner", ProjectID: "proj-1", State: StateExecuting}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.GetRun(ctx, "run_owner")
	if err != nil {
		t.Fatal(err)
	}
	if stored.ExecutionOwner != ExecutionOwnerNexus {
		t.Fatalf("legacy run must default to NEXUS owner, got %q", stored.ExecutionOwner)
	}

	stored.ExecutionOwner = ExecutionOwnerMaestro
	if err := repo.SaveRun(ctx, stored); !errors.Is(err, ErrExecutionOwnerImmutable) {
		t.Fatalf("owner change must be refused: %v", err)
	}
}

func TestNormalizeExecutionOwnerKeepsUnknownValuesVisible(t *testing.T) {
	if got := NormalizeExecutionOwner("  nexus "); got != ExecutionOwnerNexus {
		t.Fatalf("expected canonical NEXUS, got %q", got)
	}
	if got := NormalizeExecutionOwner("maestro"); got != ExecutionOwnerMaestro {
		t.Fatalf("expected canonical MAESTRO, got %q", got)
	}
	if got := NormalizeExecutionOwner("unknown-owner"); got != "UNKNOWN-OWNER" {
		t.Fatalf("unknown owner must stay visible, got %q", got)
	}
}
