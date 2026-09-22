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
		ID:           "run_coord",
		PlanID:       "plan_coord",
		PlanRevision: 1,
		ProjectID:    "project_coord",
		Workspace:    "/tmp/project",
		State:        StateExecuting,
		Contract:     DefaultAutonomyContract(),
		PackageRuns:  []PackageRun{{ID: "pkgrun_1", PackageID: "step_1", State: StateExecuting}},
		StartedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatalf("save run: %v", err)
	}

	req := CoordinationRequest{
		ID:              "coord_1",
		SourcePackageID: "step_1",
		Operation:       CoordinationNeedInfo,
		Subject:         "authentication persistence",
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
		ID:           "run_coord_invalid",
		PlanID:       "plan_coord_invalid",
		PlanRevision: 1,
		ProjectID:    "project_coord",
		Workspace:    "/tmp/project",
		State:        StateExecuting,
		Contract:     DefaultAutonomyContract(),
		PackageRuns:  []PackageRun{{ID: "pkgrun_1", PackageID: "step_1", State: StateExecuting}},
		StartedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatalf("save run: %v", err)
	}
	_, err := r.SubmitCoordinationRequest(ctx, run.ID, CoordinationRequest{
		ID:              "coord_foreign",
		SourcePackageID: "step_missing",
		Operation:       CoordinationNeedInfo,
		Subject:         "missing package",
	})
	if err == nil {
		t.Fatal("foreign source package must be rejected")
	}
}

func TestMissionRunnerMaterializesCoordinationAsDynamicWork(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, coordinationTestExecutor{})
	ctx := context.Background()
	now := time.Now().UTC()
	run := &MissionRun{
		ID:           "run_dynamic",
		PlanID:       "plan_dynamic",
		PlanRevision: 1,
		ProjectID:    "project_dynamic",
		Workspace:    "/tmp/project",
		State:        StateExecuting,
		Contract:     DefaultAutonomyContract(),
		PackageRuns: []PackageRun{
			{ID: "pkgrun_dep", PackageID: "dep", State: StateVerified, StartedAt: now},
			{ID: "pkgrun_parent", PackageID: "parent", State: StateExecuting, Dependencies: []string{"dep"}, Priority: "HIGH", RelevantPaths: []string{"internal/auth"}, StartedAt: now},
		},
		StartedAt: now,
		UpdatedAt: now,
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatalf("save run: %v", err)
	}
	req, err := r.SubmitCoordinationRequest(ctx, run.ID, CoordinationRequest{
		ID:                "coord_delegate",
		SourcePackageID:   "parent",
		SourceAgentID:     "agent_lead",
		Operation:         CoordinationDelegate,
		Subject:           "inspect token persistence",
		Objective:         "independently inspect token persistence and report evidence",
		PreferredRole:     "backend-reviewer",
		PreferredProvider: "codex",
	})
	if err != nil {
		t.Fatalf("submit coordination: %v", err)
	}
	child, err := r.MaterializeCoordinationWork(ctx, run.ID, req.ID)
	if err != nil {
		t.Fatalf("materialize coordination: %v", err)
	}
	if !child.DynamicWork || child.ParentPackageID != "parent" || child.CoordinationRequestID != req.ID {
		t.Fatalf("missing dynamic provenance: %#v", child)
	}
	if child.State != StateReady {
		t.Fatalf("verified inherited dependencies should make child READY, got %s", child.State)
	}
	if len(child.Dependencies) != 1 || child.Dependencies[0] != "dep" {
		t.Fatalf("dynamic work must inherit safe prerequisites without depending on executing parent: %#v", child.Dependencies)
	}
	if child.DesiredProvider != "codex" || child.Role != "backend-reviewer" {
		t.Fatalf("coordination preferences were not projected: %#v", child)
	}
	again, err := r.MaterializeCoordinationWork(ctx, run.ID, req.ID)
	if err != nil {
		t.Fatalf("idempotent materialization: %v", err)
	}
	if again.PackageID != child.PackageID {
		t.Fatalf("materialization must be idempotent: %s != %s", again.PackageID, child.PackageID)
	}
	stored, err := repo.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if len(stored.PackageRuns) != 3 {
		t.Fatalf("expected one dynamic child, got %d packages", len(stored.PackageRuns))
	}
	if stored.CoordinationRequests[0].ChildPackageID != child.PackageID || stored.CoordinationRequests[0].MaterializedAt == nil {
		t.Fatalf("coordination request must retain child lineage: %#v", stored.CoordinationRequests[0])
	}
}

func TestMissionRunnerCoordinationResolvesFromChildReceipt(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, coordinationTestExecutor{})
	ctx := context.Background()
	now := time.Now().UTC()
	run := &MissionRun{
		ID:           "run_receipt",
		PlanID:       "plan_receipt",
		PlanRevision: 1,
		ProjectID:    "project_receipt",
		Workspace:    "/tmp/project",
		State:        StateExecuting,
		Contract:     DefaultAutonomyContract(),
		PackageRuns:  []PackageRun{{ID: "pkgrun_parent", PackageID: "parent", State: StateExecuting, StartedAt: now}},
		StartedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatalf("save run: %v", err)
	}
	req, err := r.SubmitCoordinationRequest(ctx, run.ID, CoordinationRequest{
		ID:              "coord_review",
		SourcePackageID: "parent",
		Operation:       CoordinationReviewRequest,
		Subject:         "security review",
		Objective:       "review authentication changes independently",
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	child, err := r.MaterializeCoordinationWork(ctx, run.ID, req.ID)
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	stored, err := repo.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	dynamic := packageByID(stored, child.PackageID)
	if dynamic == nil {
		t.Fatal("dynamic package missing")
	}
	finished := time.Now().UTC()
	dynamic.State = StateVerified
	dynamic.FinishedAt = &finished
	dynamic.WorkReceipt = &WorkReceipt{
		ID:          "receipt_dynamic",
		RunID:       run.ID,
		StepID:      dynamic.PackageID,
		Status:      "VERIFIED",
		Summary:     "independent security review completed",
		StartedAt:   dynamic.StartedAt,
		CompletedAt: finished,
	}
	if err := r.saveRun(ctx, stored); err != nil {
		t.Fatalf("save completed dynamic work: %v", err)
	}
	reloaded, err := repo.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("reload resolved run: %v", err)
	}
	got := coordinationRequestByID(reloaded, req.ID)
	if got == nil || got.Status != CoordinationResolved || got.ResultReceiptID != "receipt_dynamic" || got.ResolvedAt == nil {
		t.Fatalf("coordination must resolve from verified receipt: %#v", got)
	}
}
