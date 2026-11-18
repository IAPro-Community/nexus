package handoff

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var maestroRunAckStatus = map[string]bool{
	"PENDING_WORKER": true, "CLAIMED": true, "DISPATCHED": true,
	"CANCEL_REQUESTED": true, "RECONCILIATION_REQUIRED": true,
	"COMPLETED": true, "FAILED": true, "BLOCKED": true,
	"TIMED_OUT": true, "CANCELLED": true, //nolint:misspell // Canonical Maestro Bridge status.
}

// MaestroRunSubmitRequest carries the approved plan, per-task assignments and
// the persisted PromptArtifact. Content and hash always come from the Nexus
// store; the browser never supplies them.
type MaestroRunSubmitRequest struct {
	ArtifactID       string
	Version          int
	Content          string
	SHA256           string
	ApprovedModelUse bool
	Plan             json.RawMessage
	Assignments      []MaestroWorkAssignment
	ApprovedBy       string
	MaxCostUSD       *float64
}

// MaestroRunAck is the durable admission receipt. It proves the submission is
// queued (PENDING_WORKER); it is never a promise of execution or completion.
type MaestroRunAck struct {
	SchemaVersion        int    `json:"schemaVersion"`
	SubmissionID         string `json:"submissionId"`
	RequestDigest        string `json:"requestDigest"`
	ManifestID           string `json:"manifestId"`
	ManifestDigest       string `json:"manifestDigest"`
	ApprovalDigest       string `json:"approvalDigest"`
	Status               string `json:"status"`
	Replayed             bool   `json:"replayed"`
	WorkspaceFingerprint string `json:"workspaceFingerprint"`
	Branch               string `json:"branch"`
	HeadCommit           string `json:"headCommit"`
}

var (
	maestroSafeID          = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)
	maestroSHA256          = regexp.MustCompile(`^[a-f0-9]{64}$`)
	maestroHeadCommit      = regexp.MustCompile(`^(?:[a-f0-9]{40}|[a-f0-9]{64})$`)
	maestroBranchName      = regexp.MustCompile(`^[A-Za-z0-9._/-]{1,200}$`)
	maestroSubmissionParam = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)
)

func maestroSubmitMethod(init maestroBridgeInit) (string, error) {
	if init.ProtocolVersion != maestroBridgeProtocolVersion || !init.Capabilities.RunsSubmit {
		return "", errors.New("external Maestro does not support durable runs.submit")
	}
	return "runs.submit", nil
}

func maestroSubmissionCancelMethod(init maestroBridgeInit) (string, error) {
	if init.ProtocolVersion != maestroBridgeProtocolVersion || !init.Capabilities.RunsSubmissionsCancel {
		return "", errors.New("external Maestro does not support durable submission cancellation")
	}
	return "runs.submissions.cancel", nil
}

// maestroAckFromResult strictly validates the ACK contract. Unknown fields
// fail closed: a silently extended Bridge contract must not be trusted.
func decodeMaestroRunAck(raw json.RawMessage) (MaestroRunAck, error) {
	var ack MaestroRunAck
	if len(raw) == 0 || len(raw) > maestroBridgeReplyLimit {
		return ack, errors.New("maestro run ACK exceeds reply limit")
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&ack); err != nil {
		return ack, errors.New("invalid Maestro run ACK contract")
	}
	valid := ack.SchemaVersion == 1 &&
		maestroSafeID.MatchString(ack.SubmissionID) &&
		maestroSHA256.MatchString(ack.RequestDigest) &&
		maestroSafeID.MatchString(ack.ManifestID) &&
		maestroSHA256.MatchString(ack.ManifestDigest) &&
		maestroSHA256.MatchString(ack.ApprovalDigest) &&
		maestroRunAckStatus[ack.Status] &&
		maestroSHA256.MatchString(ack.WorkspaceFingerprint) &&
		maestroBranchName.MatchString(ack.Branch) &&
		!strings.Contains(ack.Branch, "..") &&
		(ack.HeadCommit == "" || maestroHeadCommit.MatchString(ack.HeadCommit))
	if !valid {
		return MaestroRunAck{}, errors.New("inconsistent Maestro run ACK")
	}
	return ack, nil
}

// SubmitMaestroRun performs durable admission over the canonical external
// Maestro Bridge. The ACK is a fenced queue entry, not an execution promise;
// dispatch is exclusively Maestro-owned (`maestro runs worker`).
func SubmitMaestroRun(parent context.Context, workspace string, request MaestroRunSubmitRequest) (MaestroRunAck, error) {
	if request.ArtifactID == "" || request.Version < 1 ||
		strings.TrimSpace(request.Content) == "" || !maestroSHA256.MatchString(strings.ToLower(request.SHA256)) ||
		len(request.Plan) == 0 || len(request.Plan) > 128*1024 ||
		len(request.Assignments) == 0 || len(request.Assignments) > 32 {
		return MaestroRunAck{}, errors.New("invalid Maestro run submission request")
	}
	if !request.ApprovedModelUse {
		return MaestroRunAck{}, errors.New("external model use must be explicitly approved")
	}
	for _, assignment := range request.Assignments {
		if !maestroSafeAssignmentID.MatchString(assignment.TaskID) ||
			!maestroSafeAssignmentID.MatchString(assignment.ProviderID) ||
			assignment.Sandbox == "" {
			return MaestroRunAck{}, errors.New("invalid Maestro task assignment")
		}
	}
	params := map[string]any{
		"clientKey": "nexus-" + request.ArtifactID + "-" + request.SHA256[:16],
		"artifact": map[string]any{
			"id": request.ArtifactID, "version": request.Version,
			"content": request.Content, "sha256": strings.ToLower(request.SHA256),
		},
		"approvedModelUse": true,
		"plan":             request.Plan,
		"assignments":      request.Assignments,
		"approval": map[string]any{
			"decision": "approved",
			"approvedBy": func() string {
				if strings.TrimSpace(request.ApprovedBy) == "" {
					return "nexus-operator"
				}
				return request.ApprovedBy
			}(),
		},
	}
	if request.MaxCostUSD != nil {
		if *request.MaxCostUSD < 0 || *request.MaxCostUSD > 100 {
			return MaestroRunAck{}, errors.New("invalid Maestro submission budget")
		}
		params["budget"] = map[string]any{"maxCostUSD": *request.MaxCostUSD}
	}
	result, method, err := maestroBridgeCall(parent, workspace, 30*time.Second, maestroSubmitMethod, params)
	if err != nil {
		return MaestroRunAck{}, fmt.Errorf("maestro run admission unavailable: %w", err)
	}
	if method != "runs.submit" {
		return MaestroRunAck{}, errors.New("maestro submission method mismatch")
	}
	return decodeMaestroRunAck(result)
}

// CancelMaestroSubmission requests durable cancellation of a queued or
// running Maestro-owned submission. The returned row is the sanitized
// canonical projection; cancellation of CLAIMED/DISPATCHED work remains a
// request, not a confirmation.
func CancelMaestroSubmission(parent context.Context, workspace, submissionID string) (MaestroSubmissionStatus, error) {
	if !maestroSubmissionParam.MatchString(submissionID) {
		return MaestroSubmissionStatus{}, errors.New("invalid Maestro submission id")
	}
	result, method, err := maestroBridgeCall(parent, workspace, 7*time.Second,
		maestroSubmissionCancelMethod, map[string]any{"submissionId": submissionID})
	if err != nil {
		return MaestroSubmissionStatus{}, fmt.Errorf("maestro submission cancellation unavailable: %w", err)
	}
	if method != "runs.submissions.cancel" {
		return MaestroSubmissionStatus{}, errors.New("maestro cancellation method mismatch")
	}
	return decodeMaestroSubmissionRow(result)
}

// decodeMaestroSubmissionRow validates one sanitized canonical submission row.
func decodeMaestroSubmissionRow(raw json.RawMessage) (MaestroSubmissionStatus, error) {
	var row MaestroSubmissionStatus
	if len(raw) == 0 || len(raw) > maestroBridgeReplyLimit {
		return row, errors.New("maestro submission row exceeds reply limit")
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&row); err != nil {
		return row, errors.New("invalid Maestro submission row contract")
	}
	statuses := map[string]bool{
		"PENDING_WORKER": true, "CLAIMED": true, "DISPATCHED": true,
		"CANCEL_REQUESTED": true, "RECONCILIATION_REQUIRED": true,
		"COMPLETED": true, "FAILED": true, "BLOCKED": true,
		"TIMED_OUT": true, "CANCELLED": true, //nolint:misspell // Canonical Maestro Bridge status.
	}
	if row.SchemaVersion != 1 || row.Authority != "MAESTRO" || row.Owner != "MAESTRO" ||
		row.ResultEvidence != "RUN_STORE_OBSERVED" ||
		!maestroSubmissionID.MatchString(row.SubmissionID) || !statuses[row.Status] ||
		row.TaskCount < 1 || row.TaskCount > 32 || len(row.Providers) != row.TaskCount {
		return MaestroSubmissionStatus{}, errors.New("invalid Maestro submission row")
	}
	if err := validateMaestroSubmissionUsage(row.Usage); err != nil {
		return MaestroSubmissionStatus{}, fmt.Errorf("invalid Maestro submission row: %w", err)
	}
	return row, nil
}
