package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/IAPro-Community/nexus/internal/control/computer"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

const maxComputerCoordinationOutputBytes = 64 << 10

type computerCoordinationSnapshot struct {
	ID                 string              `json:"id"`
	App                computer.AppInfo    `json:"app"`
	Window             computer.WindowInfo `json:"window"`
	Elements           []computer.Element  `json:"elements,omitempty"`
	TreeText           string              `json:"tree_text,omitempty"`
	ScreenshotCaptured bool                `json:"screenshot_captured,omitempty"`
	ScreenshotRef      string              `json:"screenshot_ref,omitempty"`
	CapturedAt         string              `json:"captured_at"`
	Truncated          bool                `json:"truncated,omitempty"`
}

type computerCoordinationActionResult struct {
	Snapshot     computerCoordinationSnapshot `json:"snapshot"`
	Verification computer.ActionVerification  `json:"verification"`
}

func isComputerCoordinationOperation(operation runner.CoordinationOperation) bool {
	return operation == runner.CoordinationComputerSnapshot ||
		operation == runner.CoordinationComputerAction
}

func validateComputerCoordinationToolRequest(
	contract runner.AutonomyContract,
	operation runner.CoordinationOperation,
	raw json.RawMessage,
) error {
	if !isComputerCoordinationOperation(operation) {
		return nil
	}
	if err := validateComputerCoordinationPayloadSecrets(raw); err != nil {
		return err
	}
	policy := computerPolicyFromContract(contract)
	switch operation {
	case runner.CoordinationComputerSnapshot:
		var request computer.SnapshotRequest
		if err := decodeComputerCoordinationPayload(raw, &request); err != nil {
			return err
		}
		request.App = strings.TrimSpace(request.App)
		if err := policy.ValidateSnapshot(request); err != nil {
			return err
		}
	case runner.CoordinationComputerAction:
		var request computer.ActionRequest
		if err := decodeComputerCoordinationPayload(raw, &request); err != nil {
			return err
		}
		request.App = strings.TrimSpace(request.App)
		if err := validateComputerActionShape(request); err != nil {
			return err
		}
		if err := policy.ValidateAction(request); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported computer coordination operation %q", operation)
	}
	return nil
}

func computerCoordinationSourcePackage(
	run *runner.MissionRun,
	pkg *runner.PackageRun,
	req *runner.CoordinationRequest,
) *runner.PackageRun {
	if run == nil || pkg == nil || req == nil {
		return nil
	}
	sourceID := strings.TrimSpace(req.SourcePackageID)
	if sourceID == "" {
		sourceID = strings.TrimSpace(pkg.ParentPackageID)
	}
	for i := range run.PackageRuns {
		if run.PackageRuns[i].PackageID == sourceID {
			return &run.PackageRuns[i]
		}
	}
	return nil
}

func (e *nexusPackageExecutor) allocateComputerCoordination(
	run *runner.MissionRun,
	pkg *runner.PackageRun,
) (runner.AllocationResult, bool, error) {
	req := coordinationRequestForDynamicPackage(run, pkg)
	if req == nil || !isComputerCoordinationOperation(req.Operation) {
		return runner.AllocationResult{}, false, nil
	}
	if !run.Contract.AllowComputerUse {
		return runner.AllocationResult{}, true, fmt.Errorf("Mission AutonomyContract does not allow computer use")
	}
	source := computerCoordinationSourcePackage(run, pkg, req)
	if source == nil {
		return runner.AllocationResult{}, true, fmt.Errorf("computer coordination source package is unavailable")
	}
	agentID := strings.TrimSpace(req.SourceAgentID)
	if agentID == "" {
		agentID = strings.TrimSpace(source.AssignedAgent)
	}
	workspace := strings.TrimSpace(source.Workspace)
	if agentID == "" || workspace == "" {
		return runner.AllocationResult{}, true, fmt.Errorf("computer coordination requires source Agent and workspace attribution")
	}
	return runner.AllocationResult{
		AgentID:   agentID,
		Workspace: workspace,
		ExecutionTarget: &runner.ExecutionTarget{
			ID:        "computer-local",
			Kind:      runner.ExecutionTargetLocal,
			Adapter:   "nexus-computer",
			MachineID: registry.LocalMachineID(),
			Location:  "local",
			Transport: "computer-use",
			Workspace: workspace,
			Source:    "mission_coordination",
		},
	}, true, nil
}

func (e *nexusPackageExecutor) compileComputerCoordination(
	run *runner.MissionRun,
	pkg *runner.PackageRun,
) (runner.PromptArtifact, bool, error) {
	req := coordinationRequestForDynamicPackage(run, pkg)
	if req == nil || !isComputerCoordinationOperation(req.Operation) {
		return runner.PromptArtifact{}, false, nil
	}
	if err := validateComputerCoordinationToolRequest(run.Contract, req.Operation, req.ToolPayload); err != nil {
		return runner.PromptArtifact{}, true, fmt.Errorf("validate computer coordination request: %w", err)
	}
	payload := string(canonicalComputerCoordinationPayload(req.ToolPayload))
	content := fmt.Sprintf(
		"Nexus local computer tool request\nOperation: %s\nSubject: %s\nSource package: %s\nSource Agent: %s\nTool payload: %s\nBoundary: execute only through the Mission AutonomyContract; no provider dispatch.",
		req.Operation,
		strings.TrimSpace(req.Subject),
		strings.TrimSpace(req.SourcePackageID),
		strings.TrimSpace(req.SourceAgentID),
		payload,
	)
	artifact, err := e.persistMissionPrompt(run, pkg, content)
	return artifact, true, err
}

func (e *nexusPackageExecutor) executeComputerCoordination(
	ctx context.Context,
	run *runner.MissionRun,
	pkg *runner.PackageRun,
) (runner.ExecutionResult, bool, error) {
	req := coordinationRequestForDynamicPackage(run, pkg)
	if req == nil || !isComputerCoordinationOperation(req.Operation) {
		return runner.ExecutionResult{}, false, nil
	}
	if err := validateComputerCoordinationToolRequest(run.Contract, req.Operation, req.ToolPayload); err != nil {
		return runner.ExecutionResult{}, true, runner.MarkDispatchNotSent(
			fmt.Errorf("computer coordination request rejected before native dispatch: %w", err),
		)
	}
	runtimeID := "computer:" + req.ID
	switch req.Operation {
	case runner.CoordinationComputerSnapshot:
		var request computer.SnapshotRequest
		if err := decodeComputerCoordinationPayload(req.ToolPayload, &request); err != nil {
			return runner.ExecutionResult{}, true, runner.MarkDispatchNotSent(err)
		}
		snapshot, err := e.n.ComputerSnapshot(ctx, run.Contract, request)
		if err != nil {
			// Snapshot is observational; failure cannot cause an external
			// mutation, so retry remains safe.
			return runner.ExecutionResult{}, true, runner.MarkDispatchNotSent(
				fmt.Errorf("computer snapshot failed: %w", err),
			)
		}
		output, err := computerSnapshotCoordinationOutput(snapshot)
		if err != nil {
			return runner.ExecutionResult{RuntimeID: runtimeID}, true, err
		}
		return runner.ExecutionResult{RuntimeID: runtimeID, Output: output}, true, nil

	case runner.CoordinationComputerAction:
		var request computer.ActionRequest
		if err := decodeComputerCoordinationPayload(req.ToolPayload, &request); err != nil {
			return runner.ExecutionResult{}, true, runner.MarkDispatchNotSent(err)
		}
		result, err := e.n.ComputerAct(ctx, run.Contract, request)
		if err != nil {
			// Once ComputerAct enters the native provider, Nexus cannot prove
			// whether a click/type/drag happened before the error. Return
			// runtime evidence but leave the error unclassified so MissionRunner
			// records UNKNOWN_EXTERNAL_OUTCOME and requires reconciliation.
			return runner.ExecutionResult{RuntimeID: runtimeID}, true,
				fmt.Errorf("computer action outcome is unknown: %w", err)
		}
		output, err := computerActionCoordinationOutput(result)
		if err != nil {
			return runner.ExecutionResult{RuntimeID: runtimeID}, true, err
		}
		return runner.ExecutionResult{RuntimeID: runtimeID, Output: output}, true, nil
	default:
		return runner.ExecutionResult{}, true, runner.MarkDispatchNotSent(
			fmt.Errorf("unsupported computer coordination operation %q", req.Operation),
		)
	}
}

func canonicalComputerCoordinationPayload(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return append(json.RawMessage(nil), raw...)
	}
	return append(json.RawMessage(nil), compact.Bytes()...)
}

func decodeComputerCoordinationPayload(raw json.RawMessage, out any) error {
	if len(raw) == 0 {
		return fmt.Errorf("computer coordination tool_payload is required")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode computer coordination tool_payload: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return fmt.Errorf("computer coordination tool_payload contains trailing JSON")
	}
	return nil
}

func validateComputerCoordinationPayloadSecrets(raw json.RawMessage) error {
	text := string(raw)
	if security.Redact(text) != text {
		return fmt.Errorf("computer coordination tool_payload appears to contain secret material")
	}
	return nil
}

func computerSnapshotCoordinationOutput(snapshot *computer.Snapshot) (string, error) {
	if snapshot == nil {
		return "", fmt.Errorf("computer snapshot is unavailable")
	}
	projected := projectComputerSnapshotForCoordination(*snapshot)
	return encodeBoundedComputerCoordination(projected)
}

func computerActionCoordinationOutput(result *computer.ActionResult) (string, error) {
	if result == nil {
		return "", fmt.Errorf("computer action result is unavailable")
	}
	projected := computerCoordinationActionResult{
		Snapshot:     projectComputerSnapshotForCoordination(result.Snapshot),
		Verification: result.Verification,
	}
	return encodeBoundedComputerCoordination(projected)
}

func projectComputerSnapshotForCoordination(snapshot computer.Snapshot) computerCoordinationSnapshot {
	const maxElements = 64
	projected := computerCoordinationSnapshot{
		ID:                 snapshot.ID,
		App:                snapshot.App,
		Window:             snapshot.Window,
		TreeText:           boundComputerCoordinationText(snapshot.TreeText, 8192),
		ScreenshotCaptured: strings.TrimSpace(snapshot.Screenshot) != "",
		CapturedAt:         snapshot.CapturedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	}
	if projected.ScreenshotCaptured {
		projected.ScreenshotRef = filepath.Base(snapshot.Screenshot)
	}
	limit := len(snapshot.Elements)
	if limit > maxElements {
		limit = maxElements
		projected.Truncated = true
	}
	projected.Elements = make([]computer.Element, 0, limit)
	for i := 0; i < limit; i++ {
		element := snapshot.Elements[i]
		element.Name = boundComputerCoordinationText(element.Name, 256)
		element.Value = boundComputerCoordinationText(element.Value, 512)
		if len(element.Actions) > 12 {
			element.Actions = append([]string(nil), element.Actions[:12]...)
			projected.Truncated = true
		} else {
			element.Actions = append([]string(nil), element.Actions...)
		}
		projected.Elements = append(projected.Elements, element)
	}
	if projected.TreeText != snapshot.TreeText {
		projected.Truncated = true
	}
	return projected
}

func encodeBoundedComputerCoordination(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode computer coordination result: %w", err)
	}
	if len(raw) <= maxComputerCoordinationOutputBytes {
		return string(raw), nil
	}

	// The semantic projection should normally stay under the ceiling. If a
	// provider still returns unexpectedly large content, fail closed rather
	// than silently handing an unbounded context blob to the next Agent turn.
	return "", fmt.Errorf(
		"computer coordination result exceeds %d bytes after semantic projection",
		maxComputerCoordinationOutputBytes,
	)
}

func boundComputerCoordinationText(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	candidate := value[:maxBytes]
	for !utf8.ValidString(candidate) && len(candidate) > 0 {
		candidate = candidate[:len(candidate)-1]
	}
	return candidate + "…"
}
