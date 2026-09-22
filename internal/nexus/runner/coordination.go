package runner

import (
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
