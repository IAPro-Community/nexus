package runner

import (
	"context"
	"testing"
	"time"
)

type coordinationTestExecutor struct{}

func (coordinationTestExecutor) Allocate(context.Context, *MissionRun, *PackageRun) (AllocationResult, error) {
	return AllocationResult{}, nil
}
func (coordinationTestExecutor) Compile(context.Context, *MissionRun, *PackageRun) (PromptArtifact, error) {
	return PromptArtifact{}, nil
}
func (coordinationTestExecutor) Execute(context.Context, *MissionRun, *PackageRun, string) (ExecutionResult, error) {
	return ExecutionResult{}, nil
}
func (coordinationTestExecutor) Review(context.Context, *MissionRun, *PackageRun) (ReviewVerdict, error) {
	return ReviewVerdict{}, nil
}

func TestValidateCoordinationRequest(t *testing.T) {
	base := CoordinationRequest{
		ID:              "coord_1",
		RunID:           "run_1",
		SourcePackageID: "step_1",
		Operation:       CoordinationNeedInfo,
		Subject:         "refresh token persistence",
		Status:          CoordinationPending,
	}
	if err := ValidateCoordinationRequest(base); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	delegated := base
	delegated.Operation = CoordinationDelegate
	delegated.Objective = "inspect authentication persistence"
	delegated.PreferredRole = "backend-engineer"
	delegated.PreferredProvider = "codex"
	if err := ValidateCoordinationRequest(delegated); err != nil {
		t.Fatalf("valid delegation rejected: %v", err)
	}

	missingObjective := delegated
	missingObjective.Objective = ""
	if err := ValidateCoordinationRequest(missingObjective); err == nil {
		t.Fatal("DELEGATE without objective must fail")
	}

	unknown := base
	unknown.Operation = CoordinationOperation("CHAT")
	if err := ValidateCoordinationRequest(unknown); err == nil {
		t.Fatal("free-form/unknown coordination operation must fail closed")
	}
}


func TestMissionRunnerCoordinationLifecycleIsDurableAndIdempotent(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, coordinationTestExecutor{})
	ctx := context.Background()

	run := &MissionRun{
		ID: "run_coord",
		PlanID: "plan_coord",
		PlanRevision: 1,
		ProjectID: "project_coord",
		Workspace: "/tmp/project",
		State: StateExecuting,
		Contract: DefaultAutonomyContract(),
		PackageRuns: []PackageRun{{ID: "pkgrun_1", PackageID: "step_1", State: StateExecuting}},
		StartedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatalf("save run: %v", err)
	}

	req := CoordinationRequest{
		ID: "coord_1",
		SourcePackageID: "step_1",
		Operation: CoordinationNeedInfo,
		Subject: "authentication persistence",
	}
	first, err := r.SubmitCoordinationRequest(ctx, run.ID, req)
	if err != nil {
		t.Fatalf("submit coordination: %v", err)
	}
	if first.Status != CoordinationPending || first.RunID != run.ID {
		t.Fatalf("unexpected stored request: %#v", first)
	}
	second, err := r.SubmitCoordinationRequest(ctx, run.ID, req)
	if err != nil {
		t.Fatalf("idempotent resubmit: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected same request id, got %q != %q", second.ID, first.ID)
	}

	stored, err := repo.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if len(stored.CoordinationRequests) != 1 {
		t.Fatalf("expected one durable request, got %d", len(stored.CoordinationRequests))
	}

	resolved, err := r.ResolveCoordinationRequest(ctx, run.ID, req.ID, CoordinationResolved, "answered from durable evidence", "receipt_1")
	if err != nil {
		t.Fatalf("resolve coordination: %v", err)
	}
	if resolved.Status != CoordinationResolved || resolved.ResultReceiptID != "receipt_1" || resolved.ResolvedAt == nil {
		t.Fatalf("unexpected resolution: %#v", resolved)
	}
	if _, err := r.ResolveCoordinationRequest(ctx, run.ID, req.ID, CoordinationResolved, "answered from durable evidence", "receipt_1"); err != nil {
		t.Fatalf("idempotent resolution: %v", err)
	}
}

func TestMissionRunnerCoordinationRejectsForeignSourcePackage(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, coordinationTestExecutor{})
	ctx := context.Background()
	run := &MissionRun{
		ID: "run_coord_invalid",
		PlanID: "plan_coord_invalid",
		PlanRevision: 1,
		ProjectID: "project_coord",
		Workspace: "/tmp/project",
		State: StateExecuting,
		Contract: DefaultAutonomyContract(),
		PackageRuns: []PackageRun{{ID: "pkgrun_1", PackageID: "step_1", State: StateExecuting}},
		StartedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatalf("save run: %v", err)
	}
	_, err := r.SubmitCoordinationRequest(ctx, run.ID, CoordinationRequest{
		ID: "coord_foreign",
		SourcePackageID: "step_missing",
		Operation: CoordinationNeedInfo,
		Subject: "missing package",
	})
	if err == nil {
		t.Fatal("foreign source package must be rejected")
	}
}
