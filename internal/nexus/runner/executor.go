package runner

import (
	"context"
	"errors"
	"time"
)

// PackageExecutor bridges the deterministic runner to real Nexus Agents/providers.
// No method has a success fallback: missing execution/review capability is an error.
type PackageExecutor interface {
	Allocate(context.Context, *MissionRun, *PackageRun) (AllocationResult, error)
	Compile(context.Context, *MissionRun, *PackageRun) (PromptArtifact, error)
	Execute(context.Context, *MissionRun, *PackageRun, string) (ExecutionResult, error)
	Review(context.Context, *MissionRun, *PackageRun) (ReviewVerdict, error)
}

// ValidationEvidenceRecorder is an optional durability boundary. Real
// executors implement it so concrete verification results become claims in
// the canonical append-only evidence stream; lightweight runner fixtures may
// omit it without changing the state-machine contract.
type ValidationEvidenceRecorder interface {
	RecordValidationEvidence(context.Context, *MissionRun, *PackageRun, []VerificationResult) error
}

// GlobalValidationEvidenceRecorder records the final Definition-of-Done gate
// separately from package-local checks.
type GlobalValidationEvidenceRecorder interface {
	RecordGlobalValidationEvidence(context.Context, *MissionRun, []VerificationResult) error
}

var ErrDeliveryIntegrationRequired = errors.New("mission delivery integration is required")

// DeliveryIntegrationEvidence is a bounded factual record of materializing
// verified package outputs into one tree for downstream work/global DoD.
type DeliveryIntegrationEvidence struct {
	Workspace    string    `json:"workspace,omitempty"`
	BaseRevision string    `json:"base_revision,omitempty"`
	PackageIDs   []string  `json:"package_ids,omitempty"`
	ChangedFiles []string  `json:"changed_files,omitempty"`
	IntegratedAt time.Time `json:"integrated_at"`
}

// DeliveryIntegrator is optional. Product executors implement it when isolated
// package worktrees must be materialized into one deterministic verification
// tree. Lightweight runner fixtures can omit it and fail closed.
type DeliveryIntegrator interface {
	IntegrateDelivery(context.Context, *MissionRun) (DeliveryIntegrationEvidence, error)
}
