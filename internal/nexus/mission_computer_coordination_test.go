package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/computer"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

func TestMissionCoordinationPromptAdvertisesComputerOnlyWhenAuthorized(t *testing.T) {
	disabled := missionCoordinationPromptInstructions(runner.DefaultAutonomyContract())
	if strings.Contains(disabled, "COMPUTER_SNAPSHOT") || strings.Contains(disabled, "COMPUTER_ACTION") {
		t.Fatal("computer coordination was advertised without Mission permission")
	}

	contract := runner.DefaultAutonomyContract()
	contract.AllowComputerUse = true
	enabled := missionCoordinationPromptInstructions(contract)
	for _, expected := range []string{"COMPUTER_SNAPSHOT", "COMPUTER_ACTION", "tool_payload"} {
		if !strings.Contains(enabled, expected) {
			t.Fatalf("authorized coordination prompt is missing %q", expected)
		}
	}
}

func TestParseComputerCoordinationRequiresContractAndPreservesToolPayload(t *testing.T) {
	pkg := &runner.PackageRun{PackageID: "source", AssignedAgent: "agent-a"}
	output := `NEXUS_COORDINATION_REQUEST: {"operation":"COMPUTER_ACTION","subject":"click submit","tool_payload":{"kind":"CLICK","app":"code","element_id":"submit"}}`

	deniedRun := &runner.MissionRun{ID: "run-denied", Contract: runner.DefaultAutonomyContract()}
	if _, err := parseExecutionCoordinationRequests(output, deniedRun, pkg); err == nil {
		t.Fatal("computer coordination without allow_computer_use must fail closed")
	}

	allowed := runner.DefaultAutonomyContract()
	allowed.AllowComputerUse = true
	allowed.AllowPointerInput = true
	run := &runner.MissionRun{ID: "run-allowed", Contract: allowed}
	requests, err := parseExecutionCoordinationRequests(output, run, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 {
		t.Fatalf("requests=%d want=1", len(requests))
	}
	req := requests[0]
	if req.Operation != runner.CoordinationComputerAction || len(req.ToolPayload) == 0 {
		t.Fatalf("computer request lost typed payload: %+v", req)
	}
	var action computer.ActionRequest
	if err := json.Unmarshal(req.ToolPayload, &action); err != nil {
		t.Fatal(err)
	}
	if action.Kind != computer.ActionClick || action.App != "code" || action.ElementID != "submit" {
		t.Fatalf("unexpected action payload: %+v", action)
	}
}

func TestParseComputerCoordinationRejectsSecretPayload(t *testing.T) {
	contract := runner.DefaultAutonomyContract()
	contract.AllowComputerUse = true
	contract.AllowKeyboardInput = true
	run := &runner.MissionRun{ID: "run-secret", Contract: contract}
	pkg := &runner.PackageRun{PackageID: "source", AssignedAgent: "agent-a"}
	output := `NEXUS_COORDINATION_REQUEST: {"operation":"COMPUTER_ACTION","subject":"type value","tool_payload":{"kind":"TYPE_TEXT","app":"code","element_id":"editor","text":"OPENAI_API_KEY=sk-proj-1234567890abcdef1234567890abcdef1234567890"}}`
	if _, err := parseExecutionCoordinationRequests(output, run, pkg); err == nil {
		t.Fatal("secret-bearing computer payload must be rejected before persistence")
	}
}

func TestAllocateComputerCoordinationUsesSourceAttributionWithoutProvider(t *testing.T) {
	contract := runner.DefaultAutonomyContract()
	contract.AllowComputerUse = true
	run := &runner.MissionRun{
		ID:       "run-1",
		Contract: contract,
		PackageRuns: []runner.PackageRun{
			{PackageID: "source", AssignedAgent: "agent-source", Workspace: "/workspace/source"},
			{
				PackageID:             "coord-1",
				DynamicWork:           true,
				ParentPackageID:       "source",
				CoordinationRequestID: "req-1",
			},
		},
		CoordinationRequests: []runner.CoordinationRequest{
			{
				ID:              "req-1",
				RunID:           "run-1",
				SourcePackageID: "source",
				SourceAgentID:   "agent-source",
				Operation:       runner.CoordinationComputerSnapshot,
				Subject:         "inspect editor",
				ToolPayload:     json.RawMessage(`{"app":"code"}`),
				Status:          runner.CoordinationPending,
				CreatedAt:       time.Now().UTC(),
			},
		},
	}
	executor := &nexusPackageExecutor{n: &Nexus{}}
	allocation, handled, err := executor.allocateComputerCoordination(run, &run.PackageRuns[1])
	if err != nil {
		t.Fatal(err)
	}
	if !handled {
		t.Fatal("computer coordination was not handled locally")
	}
	if allocation.AgentID != "agent-source" || allocation.Workspace != "/workspace/source" {
		t.Fatalf("source attribution lost: %+v", allocation)
	}
	if allocation.ExecutionTarget == nil ||
		allocation.ExecutionTarget.Adapter != "nexus-computer" ||
		allocation.ExecutionTarget.Transport != "computer-use" {
		t.Fatalf("computer coordination did not use local tool target: %+v", allocation.ExecutionTarget)
	}
}

func TestExecuteComputerCoordinationSnapshotUsesCoreWithoutProviderModel(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_COMPUTER_USE", "1")
	provider := availableComputerProvider()
	provider.snapshot = computer.Snapshot{
		ID: "snapshot-1",
		App: computer.AppInfo{Name: "code", PID: 42},
		Window: computer.WindowInfo{ID: "window-1", Title: "Editor", Width: 800, Height: 600},
		Elements: []computer.Element{{ID: "submit", Index: 1, Role: "Button", Name: "Submit"}},
		CapturedAt: time.Now().UTC(),
	}
	withFakeComputerProvider(t, provider)

	contract := runner.DefaultAutonomyContract()
	contract.AllowComputerUse = true
	run := &runner.MissionRun{
		ID:       "run-snapshot",
		Contract: contract,
		CoordinationRequests: []runner.CoordinationRequest{{
			ID:              "req-snapshot",
			RunID:           "run-snapshot",
			SourcePackageID: "source",
			SourceAgentID:   "agent-source",
			Operation:       runner.CoordinationComputerSnapshot,
			Subject:         "inspect editor",
			ToolPayload:     json.RawMessage(`{"app":"code"}`),
			Status:          runner.CoordinationPending,
			CreatedAt:       time.Now().UTC(),
		}},
	}
	pkg := &runner.PackageRun{
		PackageID:             "coord",
		DynamicWork:           true,
		CoordinationRequestID: "req-snapshot",
	}
	executor := &nexusPackageExecutor{n: &Nexus{}}
	outcome, handled, err := executor.executeComputerCoordination(context.Background(), run, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if !handled || outcome.RuntimeID != "computer:req-snapshot" || !strings.Contains(outcome.Output, "snapshot-1") {
		t.Fatalf("unexpected computer snapshot outcome: handled=%v outcome=%+v", handled, outcome)
	}
	if provider.snapshotCalls != 1 || provider.actionCalls != 0 {
		t.Fatalf("unexpected native provider calls snapshot/action=%d/%d", provider.snapshotCalls, provider.actionCalls)
	}
}

func TestExecuteComputerActionFailureIsNotMarkedSafeToReplay(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_COMPUTER_USE", "1")
	provider := availableComputerProvider()
	provider.actionErr = errors.New("transport lost after native action")
	withFakeComputerProvider(t, provider)

	contract := runner.DefaultAutonomyContract()
	contract.AllowComputerUse = true
	contract.AllowWindowManagement = true
	run := &runner.MissionRun{
		ID:       "run-action",
		Contract: contract,
		CoordinationRequests: []runner.CoordinationRequest{{
			ID:              "req-action",
			RunID:           "run-action",
			SourcePackageID: "source",
			SourceAgentID:   "agent-source",
			Operation:       runner.CoordinationComputerAction,
			Subject:         "focus editor",
			ToolPayload:     json.RawMessage(`{"kind":"FOCUS_WINDOW","app":"code"}`),
			Status:          runner.CoordinationPending,
			CreatedAt:       time.Now().UTC(),
		}},
	}
	pkg := &runner.PackageRun{
		PackageID:             "coord",
		DynamicWork:           true,
		CoordinationRequestID: "req-action",
	}
	executor := &nexusPackageExecutor{n: &Nexus{}}
	outcome, handled, err := executor.executeComputerCoordination(context.Background(), run, pkg)
	if err == nil || !handled {
		t.Fatalf("native action failure must propagate: handled=%v err=%v", handled, err)
	}
	if outcome.RuntimeID != "computer:req-action" {
		t.Fatalf("action failure must retain dispatch evidence, got %+v", outcome)
	}
	if errors.Is(err, runner.ErrDispatchNotSent) {
		t.Fatalf("post-dispatch action failure was incorrectly marked replay-safe: %v", err)
	}
	if provider.actionCalls != 1 {
		t.Fatalf("native action calls=%d want=1", provider.actionCalls)
	}
}
