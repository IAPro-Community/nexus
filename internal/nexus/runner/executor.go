package runner

import "context"

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
