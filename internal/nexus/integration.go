package nexus

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
	"github.com/IAPro-Community/nexus/internal/executil"
)

// IntegrationService provides protected integration via fast-forward ff-only merges.
// It is opt-in via ConductorPolicy.IntegrationOptIn.
type IntegrationService struct {
	st *store.Store
}

// NewIntegrationService creates a new integration service.
func NewIntegrationService(st *store.Store) *IntegrationService {
	return &IntegrationService{st: st}
}

// IntegrationRequest is the input for an integration attempt.
type IntegrationRequest struct {
	BaseBranch string `json:"base_branch"`
	HeadBranch string `json:"head_branch"`
	Workspace  string `json:"workspace"`
	ProjectID  string `json:"project_id"`
	AgentID    string `json:"agent_id"`
}

// IntegrationResult is the output of an integration attempt.
type IntegrationResult struct {
	Receipt   IntegrationReceipt `json:"receipt"`
	Success   bool               `json:"success"`
	Error     string             `json:"error,omitempty"`
	MergedSHA string             `json:"merged_sha,omitempty"`
}

// Integrate performs a fast-forward ff-only merge from headBranch into baseBranch.
// It is idempotent: repeating the same integration is safe.
func (s *IntegrationService) Integrate(ctx context.Context, req IntegrationRequest) (*IntegrationResult, error) {
	if s.st == nil {
		return nil, fmt.Errorf("integration service not initialized")
	}
	if req.Workspace == "" || req.BaseBranch == "" || req.HeadBranch == "" {
		return nil, fmt.Errorf("workspace, base_branch, and head_branch are required")
	}

	// Check if integration is enabled in policy.
	policy, err := GetConductorPolicy(s.st)
	if err != nil {
		return nil, fmt.Errorf("read conductor policy: %w", err)
	}
	if policy.IntegrationOptIn == nil || !*policy.IntegrationOptIn {
		return nil, fmt.Errorf("integration is not enabled; set integration_opt_in=true in conductor policy")
	}

	// Capture base SHA before merge.
	baseSHA, err := gitRevParse(ctx, req.Workspace, req.BaseBranch)
	if err != nil {
		return &IntegrationResult{
			Receipt: IntegrationReceipt{BaseSHA: baseSHA, Status: "FAILED", Error: err.Error(), PulledAt: time.Now().UTC()},
			Success: false,
			Error:   fmt.Sprintf("resolve base branch: %v", err),
		}, nil
	}

	// Attempt fast-forward merge.
	cmd := executil.CommandContext(ctx, "git", "-C", req.Workspace, "merge", "--ff-only", req.HeadBranch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(output))
		if errMsg == "" {
			errMsg = err.Error()
		}
		receipt := IntegrationReceipt{
			BaseSHA:  baseSHA,
			Status:   "CONFLICT",
			Error:    errMsg,
			PulledAt: time.Now().UTC(),
		}
		_ = s.recordReceipt(req, receipt)
		return &IntegrationResult{
			Receipt: receipt,
			Success: false,
			Error:   errMsg,
		}, nil
	}

	// Get the merged SHA.
	mergedSHA, err := gitRevParseHead(ctx, req.Workspace)
	if err != nil {
		mergedSHA = "unknown"
	}

	receipt := IntegrationReceipt{
		BaseSHA:  baseSHA,
		HeadSHA:  mergedSHA,
		Status:   "SUCCESS",
		PulledAt: time.Now().UTC(),
	}
	_ = s.recordReceipt(req, receipt)

	return &IntegrationResult{
		Receipt:   receipt,
		Success:   true,
		MergedSHA: mergedSHA,
	}, nil
}

func (s *IntegrationService) recordReceipt(req IntegrationRequest, receipt IntegrationReceipt) error {
	data := fmt.Sprintf(`{"base_sha":"%s","head_sha":"%s","status":"%s","error":"%s"}`,
		receipt.BaseSHA, receipt.HeadSHA, receipt.Status, receipt.Error)
	_, err := s.st.RecordEventMetadata(store.EventMetadata{
		AgentID:   req.AgentID,
		ProjectID: req.ProjectID,
		Kind:      "integration_receipt",
		Timestamp: receipt.PulledAt,
		Summary:   data,
	})
	return err
}

func gitRevParse(ctx context.Context, workspace, branch string) (string, error) {
	cmd := executil.CommandContext(ctx, "git", "-C", workspace, "rev-parse", "--verify", branch)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("rev-parse %s: %w", branch, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func gitRevParseHead(ctx context.Context, workspace string) (string, error) {
	cmd := executil.CommandContext(ctx, "git", "-C", workspace, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
