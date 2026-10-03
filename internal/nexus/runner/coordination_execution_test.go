package runner

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

type coordinationExecutionExecutor struct {
	t              *testing.T
	sourceRuns     int
	helperRuns     int
	expectedAnswer string
}

func (e *coordinationExecutionExecutor) Allocate(_ context.Context, _ *MissionRun, pkg *PackageRun) (AllocationResult, error) {
	agentID := strings.TrimSpace(pkg.ResumeAgentID)
	if agentID == "" {
		agentID = strings.TrimSpace(pkg.AssignedAgent)
	}
	if agentID == "" {
		if pkg.DynamicWork {
			agentID = "agent-helper"
		} else {
			agentID = "agent-source"
		}
	}
	return AllocationResult{AgentID: agentID, Workspace: e.t.TempDir()}, nil
}

func (e *coordinationExecutionExecutor) Compile(_ context.Context, _ *MissionRun, pkg *PackageRun) (PromptArtifact, error) {
	content := pkg.Goal
	if pkg.ContextCapsule != nil {
		content += "\n" + RenderContextCapsule(pkg.ContextCapsule)
	}
	return PromptArtifact{VersionID: fmt.Sprintf("prompt-%s-%d", pkg.PackageID, e.sourceRuns+e.helperRuns+1), Content: content}, nil
}

func (e *coordinationExecutionExecutor) Execute(_ context.Context, _ *MissionRun, pkg *PackageRun, prompt string) (ExecutionResult, error) {
	if pkg.DynamicWork {
		e.helperRuns++
		return ExecutionResult{RuntimeID: "runtime-helper", Output: e.expectedAnswer}, nil
	}
	e.sourceRuns++
	if e.sourceRuns == 1 {
		return ExecutionResult{
			RuntimeID:              "runtime-source-first",
			CoordinationReplaySafe: true,
			CoordinationRequests: []CoordinationRequest{{
				Operation: CoordinationNeedInfo,
				Subject:   "architecture.auth_endpoint",
				Objective: "resolve the canonical authentication endpoint before implementation",
			}},
		}, nil
	}
	if !strings.Contains(prompt, e.expectedAnswer) {
		e.t.Fatalf("resumed source prompt did not receive coordination receipt: %q", prompt)
	}
	return ExecutionResult{RuntimeID: "runtime-source-resumed", Output: "implementation completed"}, nil
}

func (e *coordinationExecutionExecutor) Review(context.Context, *MissionRun, *PackageRun) (ReviewVerdict, error) {
	return ReviewVerdict{}, fmt.Errorf("review not expected")
}

func TestMissionRunnerCoordinationRequestReturnsReceiptToSameSourceWork(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRunRepository()
	executor := &coordinationExecutionExecutor{
		t:              t,
		expectedAnswer: "POST /api/v1/auth/login is the canonical endpoint",
	}
	r := NewMissionRunner(repo, executor)
	plan := PlanSpec{
		ID:        "plan-coordination-e2e",
		ProjectID: "project-coordination-e2e",
		Revision:  1,
		Packages: []PackageSpec{{
			ID:        "source",
			PhaseType: phaseImplementation,
			Title:     "Implement authentication client",
			Goal:      "Implement the authentication client against the canonical endpoint",
		}},
	}
	run, err := r.StartMissionRun(ctx, plan, t.TempDir(), AutonomyContract{
		MaxRetries:          2,
		MaxTotalIterations:  40,
		MaxNoProgress:       3,
		RequireVerification: false,
	}, "agent-source")
	if err != nil {
		t.Fatal(err)
	}

	completed := false
	for i := 0; i < 40; i++ {
		current, done, stepErr := r.ExecuteNextStep(ctx, run.ID)
		if stepErr != nil {
			t.Fatalf("step %d failed in state %s: %v", i, current.State, stepErr)
		}
		if done {
			completed = true
			break
		}
	}
	if !completed {
		t.Fatal("mission did not complete after coordination")
	}

	got, err := r.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StateCompletedVerified {
		t.Fatalf("state=%s want %s", got.State, StateCompletedVerified)
	}
	if executor.sourceRuns != 2 || executor.helperRuns != 1 {
		t.Fatalf("unexpected executions source=%d helper=%d", executor.sourceRuns, executor.helperRuns)
	}
	if len(got.CoordinationRequests) != 1 {
		t.Fatalf("coordination requests=%d want 1", len(got.CoordinationRequests))
	}
	req := got.CoordinationRequests[0]
	if req.Status != CoordinationResolved || req.ResultReceiptID == "" || req.ChildPackageID == "" {
		t.Fatalf("coordination was not durably resolved: %+v", req)
	}
	source := packageByID(got, "source")
	child := packageByID(got, req.ChildPackageID)
	if source == nil || child == nil {
		t.Fatalf("source/child missing: source=%v child=%v", source, child)
	}
	if source.AssignedAgent != "agent-source" {
		t.Fatalf("source Agent identity changed after coordination: %q", source.AssignedAgent)
	}
	if child.WorkReceipt == nil || child.WorkReceipt.Summary != executor.expectedAnswer {
		t.Fatalf("coordination answer was not preserved in WorkReceipt: %+v", child.WorkReceipt)
	}
	foundDependency := false
	for _, dep := range source.Dependencies {
		if dep == child.PackageID {
			foundDependency = true
			break
		}
	}
	if !foundDependency {
		t.Fatalf("source package does not depend on coordination child: %v", source.Dependencies)
	}
}

type unsafeCoordinationExecutor struct {
	workspace string
}

func (e unsafeCoordinationExecutor) Allocate(_ context.Context, _ *MissionRun, _ *PackageRun) (AllocationResult, error) {
	return AllocationResult{AgentID: "agent-source", Workspace: e.workspace}, nil
}

func (unsafeCoordinationExecutor) Compile(_ context.Context, _ *MissionRun, pkg *PackageRun) (PromptArtifact, error) {
	return PromptArtifact{VersionID: "prompt-" + pkg.PackageID, Content: pkg.Goal}, nil
}

func (unsafeCoordinationExecutor) Execute(_ context.Context, _ *MissionRun, _ *PackageRun, _ string) (ExecutionResult, error) {
	return ExecutionResult{
		RuntimeID: "runtime-source",
		CoordinationRequests: []CoordinationRequest{{
			Operation: CoordinationNeedInfo,
			Subject:   "architecture.auth_endpoint",
			Objective: "resolve the canonical endpoint",
		}},
	}, nil
}

func (unsafeCoordinationExecutor) Review(context.Context, *MissionRun, *PackageRun) (ReviewVerdict, error) {
	return ReviewVerdict{}, fmt.Errorf("review not expected")
}

func TestMissionRunnerBlocksCoordinationWithoutReplaySafetyProof(t *testing.T) {
	ctx := context.Background()
	r := NewMissionRunner(NewMemoryRunRepository(), unsafeCoordinationExecutor{workspace: t.TempDir()})
	run, err := r.StartMissionRun(ctx, PlanSpec{
		ID:        "plan-unsafe-coordination",
		ProjectID: "project-unsafe-coordination",
		Revision:  1,
		Packages: []PackageSpec{{
			ID:    "source",
			Title: "Source",
			Goal:  "Implement source",
		}},
	}, t.TempDir(), AutonomyContract{
		MaxRetries:         1,
		MaxTotalIterations: 10,
		MaxNoProgress:      2,
	}, "agent-source")
	if err != nil {
		t.Fatal(err)
	}

	var stepErr error
	for i := 0; i < 5; i++ {
		run, _, stepErr = r.ExecuteNextStep(ctx, run.ID)
		if stepErr != nil {
			break
		}
	}
	if stepErr == nil {
		t.Fatal("coordination without replay-safety proof must block")
	}
	if run.State != StateBlockedNeedsUser {
		t.Fatalf("state=%s want %s", run.State, StateBlockedNeedsUser)
	}
	if run.NeedsHuman == nil || run.NeedsHuman.ReasonCode != "COORDINATION_REPLAY_UNPROVEN" {
		t.Fatalf("unexpected human intervention: %+v", run.NeedsHuman)
	}
	if len(run.CoordinationRequests) != 0 {
		t.Fatalf("unsafe request must not be materialized, got %d requests", len(run.CoordinationRequests))
	}
}


type computerCoordinationExecutionExecutor struct {
	executeCalls int
	reviewCalls  int
}

func (e *computerCoordinationExecutionExecutor) Allocate(
	_ context.Context,
	_ *MissionRun,
	_ *PackageRun,
) (AllocationResult, error) {
	return AllocationResult{
		AgentID:   "agent-source",
		Workspace: "/tmp/nexus-computer-coordination",
		ExecutionTarget: &ExecutionTarget{
			ID:        "computer-local",
			Kind:      ExecutionTargetLocal,
			Adapter:   "nexus-computer",
			Transport: "computer-use",
			Workspace: "/tmp/nexus-computer-coordination",
		},
	}, nil
}

func (e *computerCoordinationExecutionExecutor) Compile(
	_ context.Context,
	_ *MissionRun,
	pkg *PackageRun,
) (PromptArtifact, error) {
	return PromptArtifact{
		VersionID: "prompt-" + pkg.PackageID,
		Content:   "immutable local computer tool audit artifact",
	}, nil
}

func (e *computerCoordinationExecutionExecutor) Execute(
	_ context.Context,
	_ *MissionRun,
	_ *PackageRun,
	_ string,
) (ExecutionResult, error) {
	e.executeCalls++
	return ExecutionResult{
		RuntimeID: "computer:req-computer",
		Output:    `{"kind":"CLICK","verification":{"state":"OBSERVED"}}`,
	}, nil
}

func (e *computerCoordinationExecutionExecutor) Review(
	context.Context,
	*MissionRun,
	*PackageRun,
) (ReviewVerdict, error) {
	e.reviewCalls++
	return ReviewVerdict{}, fmt.Errorf("computer coordination must not enter review")
}

func TestMissionRunnerComputerCoordinationVerifiesWithoutCodeTesting(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRunRepository()
	executor := &computerCoordinationExecutionExecutor{}
	r := NewMissionRunner(repo, executor)

	workspace := t.TempDir()
	now := time.Now().UTC()
	run := &MissionRun{
		ID:          "run-computer-tool",
		PlanID:      "plan-computer-tool",
		PlanRevision: 1,
		ProjectID:   "project-computer-tool",
		Workspace:   workspace,
		State:       StateExecuting,
		Contract: AutonomyContract{
			MaxRetries:          1,
			MaxTotalIterations:  12,
			MaxNoProgress:       2,
			RequireVerification: true,
			VerificationCommands: []string{
				"this-command-must-never-run-for-computer-tool",
			},
			AllowComputerUse:  true,
			AllowPointerInput: true,
		},
		StartedAt:      now,
		LastProgressAt: now,
		UpdatedAt:      now,
		PackageRuns: []PackageRun{{
			ID:                    "pkgrun-computer",
			PackageID:             "coord-computer",
			PhaseID:               "coord-computer",
			PhaseType:             "COORDINATION",
			Title:                 "COMPUTER_ACTION · click submit",
			Goal:                  "execute one bounded computer action",
			Priority:              "NORMAL",
			DynamicWork:           true,
			ParentPackageID:       "source",
			CoordinationRequestID: "req-computer",
			State:                 StateReady,
			Attempt:               1,
			StartedAt:             now,
		}},
		CoordinationRequests: []CoordinationRequest{{
			ID:              "req-computer",
			RunID:           "run-computer-tool",
			SourcePackageID: "source",
			SourceAgentID:   "agent-source",
			Operation:       CoordinationComputerAction,
			Subject:         "click submit",
			ToolPayload:     []byte(`{"kind":"CLICK","app":"code","element_id":"submit"}`),
			Status:          CoordinationPending,
			ChildPackageID:  "coord-computer",
			CreatedAt:       now,
			MaterializedAt:  &now,
		}},
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}

	for step := 0; step < 4; step++ {
		current, done, err := r.ExecuteNextStep(ctx, run.ID)
		if err != nil {
			t.Fatalf("step %d failed in run=%s: %v", step, current.State, err)
		}
		if done {
			t.Fatalf("computer child unexpectedly completed whole run at step %d", step)
		}
	}

	got, err := r.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	child := packageByID(got, "coord-computer")
	if child == nil {
		t.Fatal("computer coordination child missing")
	}
	if child.State != StateVerified {
		t.Fatalf("computer coordination state=%s want=%s", child.State, StateVerified)
	}
	if len(child.Verifications) != 0 {
		t.Fatalf("computer coordination unexpectedly ran code verification: %+v", child.Verifications)
	}
	if child.WorkReceipt == nil || child.WorkReceipt.Status != "VERIFIED" ||
		!strings.Contains(child.WorkReceipt.Summary, "CLICK") {
		t.Fatalf("computer coordination receipt missing: %+v", child.WorkReceipt)
	}
	if child.AssignedRuntime != "computer:req-computer" {
		t.Fatalf("computer runtime evidence=%q", child.AssignedRuntime)
	}
	if executor.executeCalls != 1 || executor.reviewCalls != 0 {
		t.Fatalf("unexpected executor calls execute=%d review=%d", executor.executeCalls, executor.reviewCalls)
	}
	req := got.CoordinationRequests[0]
	if req.Status != CoordinationResolved || req.ResultReceiptID == "" {
		t.Fatalf("computer coordination request was not reconciled from receipt: %+v", req)
	}
}
