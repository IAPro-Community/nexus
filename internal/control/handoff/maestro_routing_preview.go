package handoff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var maestroSafeAssignmentID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

// MaestroWorkAssignment selects the real external Maestro executor per task.
// Nexus must not replace or infer the provider when the Maestro Bridge is down.
type MaestroWorkAssignment struct {
	TaskID       string `json:"taskId"`
	ProviderID   string `json:"providerId"`
	Model        string `json:"model,omitempty"`
	Sandbox      string `json:"sandbox"`
	WorktreePath string `json:"worktreePath"`
}

// MaestroRoutePreviewRequest carries an existing finalized Composer artifact
// and caller-selected task assignments. The proposal remains validated by Maestro.
type MaestroRoutePreviewRequest struct {
	ArtifactID  string
	Version     int
	Content     string
	SHA256      string
	Proposal    json.RawMessage
	Assignments []MaestroWorkAssignment
}

type MaestroRouteAssignment struct {
	TaskID      string `json:"taskId"`
	ProviderID  string `json:"providerId"`
	Model       string `json:"model"`
	WorkspaceID string `json:"workspaceId"`
	Status      string `json:"status"`
	Code        string `json:"code,omitempty"`
}

type MaestroRouteBlocker struct {
	Code       string `json:"code"`
	TaskID     string `json:"taskId,omitempty"`
	ProviderID string `json:"providerId,omitempty"`
	Message    string `json:"message,omitempty"`
}

type MaestroRoutePreview struct {
	SchemaVersion    int                      `json:"schemaVersion"`
	Authority        string                   `json:"authority"`
	PreviewOnly      bool                     `json:"previewOnly"`
	Executable       bool                     `json:"executable"`
	ApprovalRequired bool                     `json:"approvalRequired"`
	Ready            bool                     `json:"ready"`
	Plan             json.RawMessage          `json:"plan"`
	Assignments      []MaestroRouteAssignment `json:"assignments"`
	Blockers         []MaestroRouteBlocker    `json:"blockers"`
}

func maestroRoutePreviewMethod(init maestroBridgeInit) (string, error) {
	if init.ProtocolVersion != maestroBridgeProtocolVersion || !init.Capabilities.PlansRoutePreview {
		return "", errors.New("external Maestro does not support canonical plans.routePreview")
	}
	return "plans.routePreview", nil
}

func decodeMaestroRoutePreview(result json.RawMessage, request MaestroRoutePreviewRequest) (MaestroRoutePreview, error) {
	var preview MaestroRoutePreview
	if err := json.Unmarshal(result, &preview); err != nil {
		return MaestroRoutePreview{}, errors.New("invalid Maestro route preview")
	}
	if preview.SchemaVersion != 1 || preview.Authority != "MAESTRO" ||
		!preview.PreviewOnly || preview.Executable || !preview.ApprovalRequired ||
		len(preview.Plan) == 0 || len(preview.Assignments) != len(request.Assignments) {
		return MaestroRoutePreview{}, errors.New("Maestro route preview lacks canonical approval/plan contract")
	}
	var plan struct {
		Authority string `json:"authority"`
		Artifact  struct {
			ID      string `json:"id"`
			Version int    `json:"version"`
			SHA256  string `json:"sha256"`
		} `json:"artifact"`
	}
	if err := json.Unmarshal(preview.Plan, &plan); err != nil ||
		plan.Authority != "MAESTRO" || plan.Artifact.ID != request.ArtifactID ||
		plan.Artifact.Version != request.Version ||
		!strings.EqualFold(plan.Artifact.SHA256, request.SHA256) {
		return MaestroRoutePreview{}, errors.New("Maestro preview artifact provenance mismatch")
	}
	choices := make(map[string]MaestroWorkAssignment, len(request.Assignments))
	for _, row := range request.Assignments {
		choices[row.TaskID] = row
	}
	seen := make(map[string]bool, len(preview.Assignments))
	for _, row := range preview.Assignments {
		expected, ok := choices[row.TaskID]
		if !ok || seen[row.TaskID] || row.ProviderID != expected.ProviderID ||
			row.Model != firstMaestroModel(expected.Model) ||
			(row.Status != "READY_FOR_REVIEW" && row.Status != "BLOCKED") {
			return MaestroRoutePreview{}, errors.New("Maestro route preview changed provider/model or duplicated a task")
		}
		seen[row.TaskID] = true
	}
	return preview, nil
}

func firstMaestroModel(model string) string {
	if model == "" {
		return "default"
	}
	return model
}

// FetchMaestroRoutePreview delegates to an *external* Maestro process; it is
// a read-only, bounded preflight. No run is created, no planner copied into Nexus.
func FetchMaestroRoutePreview(parent context.Context, workspace string, req MaestroRoutePreviewRequest) (MaestroRoutePreview, error) {
	if !maestroSafeAssignmentID.MatchString(req.ArtifactID) ||
		req.Version < 1 || req.Version > 1000000 || len(req.Content) < 8 ||
		len(req.Content) > 64000 || len(req.Proposal) == 0 ||
		len(req.Proposal) > 65536 || len(req.Assignments) == 0 || len(req.Assignments) > 32 {
		return MaestroRoutePreview{}, errors.New("invalid or unbounded Composer route preview")
	}
	sum := sha256.Sum256([]byte(req.Content))
	if !strings.EqualFold(req.SHA256, hex.EncodeToString(sum[:])) {
		return MaestroRoutePreview{}, errors.New("Composer PromptArtifact SHA256 mismatch")
	}
	seen := map[string]bool{}
	for _, assignment := range req.Assignments {
		if !maestroSafeAssignmentID.MatchString(assignment.TaskID) ||
			seen[assignment.TaskID] || strings.TrimSpace(assignment.ProviderID) == "" ||
			strings.TrimSpace(assignment.Sandbox) == "" ||
			strings.TrimSpace(assignment.WorktreePath) == "" {
			return MaestroRoutePreview{}, errors.New("invalid or duplicated Maestro task assignment")
		}
		seen[assignment.TaskID] = true
	}
	params := map[string]any{
		"projectPath": workspace,
		"artifact": map[string]any{
			"id": req.ArtifactID, "version": req.Version,
			"content": req.Content, "sha256": req.SHA256,
		},
		"proposal":    req.Proposal,
		"assignments": req.Assignments,
	}
	result, method, err := maestroBridgeCall(parent, workspace, 20*time.Second, maestroRoutePreviewMethod, params)
	if err != nil {
		return MaestroRoutePreview{}, fmt.Errorf("Maestro routing preview unavailable: %w", err)
	}
	if method != "plans.routePreview" {
		return MaestroRoutePreview{}, errors.New("invalid Maestro routing method")
	}
	return decodeMaestroRoutePreview(result, req)
}
