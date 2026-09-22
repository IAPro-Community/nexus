package runner

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// CoordinationOperation is a closed set of requests an executing Agent may
// submit to the Mission. Agents never spawn providers or mutate the Flow
// directly; MissionRunner remains the execution authority.
type CoordinationOperation string

const (
	CoordinationNeedInfo        CoordinationOperation = "NEED_INFO"
	CoordinationDelegate        CoordinationOperation = "DELEGATE"
	CoordinationArtifactRequest CoordinationOperation = "ARTIFACT_REQUEST"
	CoordinationReviewRequest   CoordinationOperation = "REVIEW_REQUEST"
)

// CoordinationStatus is the durable lifecycle of a coordination request.
type CoordinationStatus string

const (
	CoordinationPending  CoordinationStatus = "PENDING"
	CoordinationResolved CoordinationStatus = "RESOLVED"
	CoordinationRejected CoordinationStatus = "REJECTED"
	CoordinationCanceled CoordinationStatus = "CANCELED"
)

// CoordinationRequest is a Mission-owned request raised by an executing Agent.
// It is intentionally provider-neutral: source/target are Agent/task concepts,
// while provider/profile/model selection stays with ResourceScheduler.
type CoordinationRequest struct {
	ID                   string                `json:"id"`
	RunID                string                `json:"run_id"`
	SourcePackageID      string                `json:"source_package_id"`
	SourceAgentID        string                `json:"source_agent_id,omitempty"`
	Operation            CoordinationOperation `json:"operation"`
	Subject              string                `json:"subject"`
	Objective            string                `json:"objective,omitempty"`
	PreferredRole        string                `json:"preferred_role,omitempty"`
	PreferredProvider    string                `json:"preferred_provider,omitempty"`
	RequiredCapabilities []string              `json:"required_capabilities,omitempty"`
	ArtifactRefs         []string              `json:"artifact_refs,omitempty"`
	Status               CoordinationStatus    `json:"status"`
	CreatedAt            time.Time             `json:"created_at"`
	ResolvedAt           *time.Time            `json:"resolved_at,omitempty"`
	ResultReceiptID      string                `json:"result_receipt_id,omitempty"`
	ResolutionSummary    string                `json:"resolution_summary,omitempty"`
}

// ValidateCoordinationRequest enforces the architectural boundary before a
// request can become durable Mission state.
func ValidateCoordinationRequest(req CoordinationRequest) error {
	if strings.TrimSpace(req.ID) == "" {
		return fmt.Errorf("coordination request id is required")
	}
	if strings.TrimSpace(req.RunID) == "" {
		return fmt.Errorf("coordination run_id is required")
	}
	if strings.TrimSpace(req.SourcePackageID) == "" {
		return fmt.Errorf("coordination source_package_id is required")
	}
	switch req.Operation {
	case CoordinationNeedInfo, CoordinationDelegate, CoordinationArtifactRequest, CoordinationReviewRequest:
	default:
		return fmt.Errorf("unsupported coordination operation %q", req.Operation)
	}
	if strings.TrimSpace(req.Subject) == "" {
		return fmt.Errorf("coordination subject is required")
	}
	if req.Operation == CoordinationDelegate && strings.TrimSpace(req.Objective) == "" {
		return fmt.Errorf("DELEGATE coordination requires objective")
	}
	switch req.Status {
	case CoordinationPending, CoordinationResolved, CoordinationRejected, CoordinationCanceled:
	default:
		return fmt.Errorf("unsupported coordination status %q", req.Status)
	}
	return nil
}

func cloneCoordinationRequest(in CoordinationRequest) CoordinationRequest {
	out := in
	out.RequiredCapabilities = append([]string(nil), in.RequiredCapabilities...)
	out.ArtifactRefs = append([]string(nil), in.ArtifactRefs...)
	return out
}

// SubmitCoordinationRequest durably appends a validated request to the owning
// MissionRun. Replaying the same request ID is idempotent only when the
// semantic request is unchanged.
func (r *MissionRunner) SubmitCoordinationRequest(ctx context.Context, runID string, req CoordinationRequest) (*CoordinationRequest, error) {
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("mission runner repository is unavailable")
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, fmt.Errorf("coordination run id is required")
	}
	run, err := r.repo.AcquireLease(ctx, runID, r.owner, r.leaseTTL)
	if err != nil {
		return nil, err
	}
	leaseToken := run.LeaseToken
	defer func() { _ = r.repo.ReleaseLease(context.Background(), runID, r.owner, leaseToken) }()

	if req.RunID == "" {
		req.RunID = runID
	}
	if req.RunID != runID {
		return nil, fmt.Errorf("coordination request run_id %q does not match owning run %q", req.RunID, runID)
	}
	if req.Status == "" {
		req.Status = CoordinationPending
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now().UTC()
	}
	if err := ValidateCoordinationRequest(req); err != nil {
		return nil, err
	}
	if isTerminalRunState(run.State) {
		return nil, fmt.Errorf("mission run %s is terminal and cannot accept coordination", run.ID)
	}
	if packageIndex(run, req.SourcePackageID) < 0 {
		return nil, fmt.Errorf("coordination source package %q does not belong to mission run %s", req.SourcePackageID, run.ID)
	}
	for i := range run.CoordinationRequests {
		existing := &run.CoordinationRequests[i]
		if existing.ID != req.ID {
			continue
		}
		if existing.RunID == req.RunID &&
			existing.SourcePackageID == req.SourcePackageID &&
			existing.Operation == req.Operation &&
			existing.Subject == req.Subject &&
			existing.Objective == req.Objective {
			copy := cloneCoordinationRequest(*existing)
			return &copy, nil
		}
		return nil, fmt.Errorf("coordination request id %q already exists with different semantics", req.ID)
	}
	run.CoordinationRequests = append(run.CoordinationRequests, cloneCoordinationRequest(req))
	run.UpdatedAt = time.Now().UTC()
	if err := r.saveRun(ctx, run); err != nil {
		return nil, err
	}
	saved := cloneCoordinationRequest(req)
	return &saved, nil
}

// ResolveCoordinationRequest closes a durable coordination request without
// mutating the sealed mission objective. Child-work materialization remains a
// separate MissionRunner concern.
func (r *MissionRunner) ResolveCoordinationRequest(ctx context.Context, runID, requestID string, status CoordinationStatus, summary, receiptID string) (*CoordinationRequest, error) {
	if status != CoordinationResolved && status != CoordinationRejected && status != CoordinationCanceled {
		return nil, fmt.Errorf("coordination resolution status %q is not terminal", status)
	}
	run, err := r.repo.AcquireLease(ctx, runID, r.owner, r.leaseTTL)
	if err != nil {
		return nil, err
	}
	leaseToken := run.LeaseToken
	defer func() { _ = r.repo.ReleaseLease(context.Background(), runID, r.owner, leaseToken) }()

	for i := range run.CoordinationRequests {
		req := &run.CoordinationRequests[i]
		if req.ID != requestID {
			continue
		}
		if req.Status != CoordinationPending {
			if req.Status == status && req.ResolutionSummary == summary && req.ResultReceiptID == receiptID {
				copy := cloneCoordinationRequest(*req)
				return &copy, nil
			}
			return nil, fmt.Errorf("coordination request %q is already %s", requestID, req.Status)
		}
		now := time.Now().UTC()
		req.Status = status
		req.ResolvedAt = &now
		req.ResolutionSummary = strings.TrimSpace(summary)
		req.ResultReceiptID = strings.TrimSpace(receiptID)
		run.UpdatedAt = now
		if err := r.saveRun(ctx, run); err != nil {
			return nil, err
		}
		copy := cloneCoordinationRequest(*req)
		return &copy, nil
	}
	return nil, fmt.Errorf("coordination request %q not found in mission run %s", requestID, runID)
}
