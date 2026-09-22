package runner

import (
	"context"
	"fmt"
	"strings"
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

func TestMissionRunnerCoordinationRejectsSemanticDriftForSameID(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, coordinationTestExecutor{})
	ctx := context.Background()
	now := time.Now().UTC()
	run := &MissionRun{
		ID:           "run_coord_drift",
		PlanID:       "plan_coord_drift",
		PlanRevision: 1,
		ProjectID:    "project_coord",
		Workspace:    "/tmp/project",
		State:        StateExecuting,
		Contract:     DefaultAutonomyContract(),
		PackageRuns:  []PackageRun{{ID: "pkgrun_1", PackageID: "step_1", State: StateExecuting}},
		StartedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatalf("save run: %v", err)
	}
	base := CoordinationRequest{
		ID:                   "coord_same",
		SourcePackageID:      "step_1",
		Operation:            CoordinationDelegate,
		Subject:              "review auth",
		Objective:            "review auth independently",
		PreferredProvider:    "codex",
		RequiredCapabilities: []string{"code_review"},
	}
	if _, err := r.SubmitCoordinationRequest(ctx, run.ID, base); err != nil {
		t.Fatalf("initial submit: %v", err)
	}
	changed := base
	changed.PreferredProvider = "claude"
	if _, err := r.SubmitCoordinationRequest(ctx, run.ID, changed); err == nil {
		t.Fatal("same coordination id with changed semantics must fail closed")
	}
}

func TestMissionRunnerCoordinationResolutionNormalizesWhitespaceForIdempotency(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, coordinationTestExecutor{})
	ctx := context.Background()
	now := time.Now().UTC()
	run := &MissionRun{
		ID:           "run_coord_resolution",
		PlanID:       "plan_coord_resolution",
		PlanRevision: 1,
		ProjectID:    "project_coord",
		Workspace:    "/tmp/project",
		State:        StateExecuting,
		Contract:     DefaultAutonomyContract(),
		PackageRuns:  []PackageRun{{ID: "pkgrun_1", PackageID: "step_1", State: StateExecuting}},
		StartedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatalf("save run: %v", err)
	}
	req, err := r.SubmitCoordinationRequest(ctx, run.ID, CoordinationRequest{
		ID:              "coord_ws",
		SourcePackageID: "step_1",
		Operation:       CoordinationNeedInfo,
		Subject:         "verification",
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := r.ResolveCoordinationRequest(ctx, run.ID, req.ID, CoordinationResolved, " answer ", " receipt "); err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	if _, err := r.ResolveCoordinationRequest(ctx, run.ID, req.ID, CoordinationResolved, "answer", "receipt"); err != nil {
		t.Fatalf("normalized duplicate resolve must be idempotent: %v", err)
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


func TestValidateCoordinationRequestRejectsOversizedPayload(t *testing.T) {
	req := CoordinationRequest{
		ID:              "coord_oversized",
		RunID:           "run_oversized",
		SourcePackageID: "source",
		Operation:       CoordinationNeedInfo,
		Subject:         strings.Repeat("x", maxCoordinationSubjectBytes+1),
		Status:          CoordinationPending,
	}
	if err := ValidateCoordinationRequest(req); err == nil {
		t.Fatal("oversized coordination subject must fail closed")
	}

	req.Subject = "bounded"
	req.RequiredCapabilities = make([]string, maxCoordinationListItems+1)
	for i := range req.RequiredCapabilities {
		req.RequiredCapabilities[i] = "capability"
	}
	if err := ValidateCoordinationRequest(req); err == nil {
		t.Fatal("oversized coordination capability list must fail closed")
	}
}

func TestMissionRunnerRejectsCoordinationFanOutBeforeMaterializingChildren(t *testing.T) {
	now := time.Now().UTC()
	run := &MissionRun{
		ID:           "run_fanout",
		PlanID:       "plan_fanout",
		PlanRevision: 1,
		ProjectID:    "project_fanout",
		State:        StateExecuting,
		PackageRuns: []PackageRun{{
			ID:            "pkgrun_source",
			PackageID:     "source",
			AssignedAgent: "agent-source",
			State:         StateExecuting,
			StartedAt:     now,
		}},
		StartedAt: now,
		UpdatedAt: now,
	}
	requests := make([]CoordinationRequest, maxCoordinationRequestsPerTurn+1)
	for i := range requests {
		requests[i] = CoordinationRequest{
			Operation: CoordinationNeedInfo,
			Subject:   fmt.Sprintf("fact-%d", i),
		}
	}
	r := &MissionRunner{}
	if err := r.acceptExecutionCoordination(run, "source", requests); err == nil {
		t.Fatal("coordination fan-out above safe turn limit must fail closed")
	}
	if len(run.CoordinationRequests) != 0 || len(run.PackageRuns) != 1 {
		t.Fatalf("fan-out rejection must be atomic: requests=%d packages=%d", len(run.CoordinationRequests), len(run.PackageRuns))
	}
	if run.PackageRuns[0].State != StateExecuting {
		t.Fatalf("source package changed despite rejected fan-out: %s", run.PackageRuns[0].State)
	}
}

func TestMissionCoordinationTotalLimitPreservesIdempotentReplay(t *testing.T) {
	now := time.Now().UTC()
	run := &MissionRun{
		ID:           "run_total_limit",
		PlanID:       "plan_total_limit",
		PlanRevision: 1,
		ProjectID:    "project_total_limit",
		State:        StateExecuting,
		PackageRuns: []PackageRun{{
			ID:            "pkgrun_source",
			PackageID:     "source",
			AssignedAgent: "agent-source",
			State:         StateExecuting,
			StartedAt:     now,
		}},
		StartedAt: now,
		UpdatedAt: now,
	}
	var first CoordinationRequest
	for i := 0; i < maxCoordinationRequestsPerMission; i++ {
		req := CoordinationRequest{
			ID:              fmt.Sprintf("coord_limit_%d", i),
			RunID:           run.ID,
			SourcePackageID: "source",
			SourceAgentID:   "agent-source",
			Operation:       CoordinationNeedInfo,
			Subject:         fmt.Sprintf("fact-%d", i),
			Status:          CoordinationPending,
			CreatedAt:       now,
		}
		if i == 0 {
			first = req
		}
		if _, err := appendCoordinationRequestInRun(run, req); err != nil {
			t.Fatalf("append request %d: %v", i, err)
		}
	}
	if _, err := appendCoordinationRequestInRun(run, CoordinationRequest{
		ID:              "coord_over_limit",
		RunID:           run.ID,
		SourcePackageID: "source",
		SourceAgentID:   "agent-source",
		Operation:       CoordinationNeedInfo,
		Subject:         "one more fact",
		Status:          CoordinationPending,
		CreatedAt:       now,
	}); err == nil {
		t.Fatal("mission coordination total limit must reject new semantic work")
	}
	if _, err := appendCoordinationRequestInRun(run, first); err != nil {
		t.Fatalf("idempotent replay must remain valid at the total limit: %v", err)
	}
}
