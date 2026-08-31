package nexus

import "time"

// ContextCapsule is the bounded input for a Flow step. It intentionally has
// no transcript field: continuity is represented by durable facts and
// dependency receipts.
type ContextCapsule struct {
	ProjectID          string        `json:"project_id"`
	Branch             string        `json:"branch"`
	Head               string        `json:"head"`
	ContextFingerprint string        `json:"context_fingerprint"`
	FlowID             string        `json:"flow_id"`
	FlowRevision       int           `json:"flow_revision"`
	StepID             string        `json:"step_id"`
	RelevantMemory     []string      `json:"relevant_memory,omitempty"`
	DependencyReceipts []WorkReceipt `json:"dependency_receipts,omitempty"`
	RelevantPaths      []string      `json:"relevant_paths,omitempty"`
	MaestroSkills      []string      `json:"maestro_skills,omitempty"`
	AcceptanceCriteria []string      `json:"acceptance_criteria,omitempty"`
	Constraints        []string      `json:"constraints,omitempty"`
}

// WorkReceipt contains facts observed during execution. Empty evidence stays
// empty; callers must not manufacture commands, files, or verification.
type WorkReceipt struct {
	StepID          string    `json:"step_id"`
	Status          string    `json:"status"`
	Summary         string    `json:"summary,omitempty"`
	ChangedFiles    []string  `json:"changed_files,omitempty"`
	Commands        []string  `json:"commands,omitempty"`
	Tests           []string  `json:"tests,omitempty"`
	Decisions       []string  `json:"decisions,omitempty"`
	Artifacts       []string  `json:"artifacts,omitempty"`
	RemainingIssues []string  `json:"remaining_issues,omitempty"`
	Verification    []string  `json:"verification,omitempty"`
	AgentID         string    `json:"agent_id,omitempty"`
	BaseRevision    string    `json:"base_revision,omitempty"`
	ResultRevision  string    `json:"result_revision,omitempty"`
	StartedAt       time.Time `json:"started_at"`
	CompletedAt     time.Time `json:"completed_at"`
}

// NewContextCapsule builds a bounded step input and copies slices so later
// editor mutations cannot silently alter an in-flight execution request.
func NewContextCapsule(projectID, branch, head, fingerprint, flowID string, revision int, stepID string, receipts []WorkReceipt, paths, skills, acceptance, constraints []string) ContextCapsule {
	return ContextCapsule{ProjectID: projectID, Branch: branch, Head: head, ContextFingerprint: fingerprint, FlowID: flowID, FlowRevision: revision, StepID: stepID, DependencyReceipts: append([]WorkReceipt(nil), receipts...), RelevantPaths: append([]string(nil), paths...), MaestroSkills: append([]string(nil), skills...), AcceptanceCriteria: append([]string(nil), acceptance...), Constraints: append([]string(nil), constraints...)}
}
