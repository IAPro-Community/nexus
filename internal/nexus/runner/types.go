package runner

import (
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

// State is an explicit durable state in the autonomous execution lifecycle.
type State string

// DispatchState records the durable provider-dispatch boundary. INTENT is
// persisted before invoking the provider. If Nexus restarts with an INTENT and
// no completion evidence, the runner fails closed instead of dispatching the
// same Step a second time.
type DispatchState string

const (
	DispatchNone                   DispatchState = ""
	DispatchIntent                 DispatchState = "INTENT"
	DispatchCompleted              DispatchState = "COMPLETED"
	DispatchFailed                 DispatchState = "FAILED" // provider launched and returned a classified failure
	DispatchFailedBeforeDispatch   DispatchState = "FAILED_BEFORE_DISPATCH"
	DispatchUnknownExternalOutcome DispatchState = "UNKNOWN_EXTERNAL_OUTCOME"
)

const (
	StatePending              State = "PENDING"
	StateReady                State = "READY"
	StateAllocating           State = "ALLOCATING"
	StateCompiling            State = "COMPILING"
	StateExecuting            State = "EXECUTING"
	StateTesting              State = "TESTING"
	StateReviewing            State = "REVIEWING"
	StateVerifying            State = "VERIFYING"
	StateRemediating          State = "REMEDIATING"
	StateVerified             State = "VERIFIED"
	StateCompletedVerified    State = "COMPLETED_VERIFIED"
	StateFailed               State = "FAILED"
	StateFailedNoProgress     State = "FAILED_NO_PROGRESS"
	StateFailedBudgetExceeded State = "FAILED_BUDGET_EXCEEDED"
	StateFailedVerification   State = "FAILED_VERIFICATION"
	StateBlockedNeedsUser     State = "BLOCKED_NEEDS_USER"
	StateEscalated            State = "ESCALATED" // compatibility for older clients
	StatePaused               State = "PAUSED"
	StateCanceledByUser       State = "CANCELED_BY_USER"
	// StateCompleted is retained as an alias for old integrations.
	StateCompleted = StateCompletedVerified
)

// PlanSpec is the immutable execution input consumed by runner. It deliberately
// does not import the SQLite store package, keeping the state machine testable.
type PlanSpec struct {
	ID                  string        `json:"id"`
	ProjectID           string        `json:"project_id"`
	Revision            int           `json:"revision"`
	LeadAgentID         string        `json:"lead_agent_id,omitempty"`
	ExecutionSnapshotID string        `json:"execution_snapshot_id,omitempty"`
	Autonomous          bool          `json:"autonomous,omitempty"`
	Packages            []PackageSpec `json:"packages"`
}

type PackageSpec struct {
	ID                       string   `json:"id"`
	PhaseID                  string   `json:"phase_id"`
	PhaseType                string   `json:"phase_type,omitempty"`
	Title                    string   `json:"title"`
	Goal                     string   `json:"goal"`
	Priority                 string   `json:"priority"`
	Dependencies             []string `json:"dependencies,omitempty"`
	ParallelGroup            string   `json:"parallel_group,omitempty"`
	Role                     string   `json:"role,omitempty"`
	ReviewRequired           bool     `json:"review_required,omitempty"`
	TaskRequirements         string   `json:"task_requirements,omitempty"`
	AgentAllocation          string   `json:"agent_allocation,omitempty"`
	AssignmentStrategy       string   `json:"assignment_strategy,omitempty"`
	ResourcePolicy           string   `json:"resource_policy,omitempty"`
	Provider                 string   `json:"provider,omitempty"`
	Profile                  string   `json:"profile,omitempty"`
	DesiredProvider          string   `json:"desired_provider,omitempty"`
	DesiredProfile           string   `json:"desired_profile,omitempty"`
	SkillIDs                 []string `json:"skill_ids,omitempty"`
	MaestroSkills            []string `json:"maestro_skills,omitempty"`
	RelevantPaths            []string `json:"relevant_paths,omitempty"`
	AcceptanceCriteria       []string `json:"acceptance_criteria,omitempty"`
	VerificationRequirements []string `json:"verification_requirements,omitempty"`
	EntryCriteria            []string `json:"entry_criteria,omitempty"`
	ExitCriteria             []string `json:"exit_criteria,omitempty"`
	RequiredEvidence         []string `json:"required_evidence,omitempty"`
	HandoffJSON              string   `json:"handoff,omitempty"`
	RoutingDecisionJSON      string   `json:"routing_decision,omitempty"`
}

type ReviewFindingStatus string

const (
	FindingNew       ReviewFindingStatus = "NEW"
	FindingOpen      ReviewFindingStatus = "OPEN"
	FindingResolved  ReviewFindingStatus = "RESOLVED"
	FindingRegressed ReviewFindingStatus = "REGRESSED"
)

// ReviewFinding is durable, semantic review evidence. ID is derived from the
// normalized category/location/description and remains stable across retries.
type ReviewFinding struct {
	ID               string              `json:"id"`
	Severity         string              `json:"severity,omitempty"`
	Category         string              `json:"category,omitempty"`
	File             string              `json:"file,omitempty"`
	Location         string              `json:"location,omitempty"`
	Description      string              `json:"description"`
	Rationale        string              `json:"rationale,omitempty"`
	RequiredAction   string              `json:"required_action,omitempty"`
	Evidence         []string            `json:"evidence,omitempty"`
	Status           ReviewFindingStatus `json:"status"`
	FirstSeenAttempt int                 `json:"first_seen_attempt,omitempty"`
	LastSeenAttempt  int                 `json:"last_seen_attempt,omitempty"`
}

// ReviewVerdict records independent evaluation evidence. A verdict without a
// reviewer identity is never accepted by MissionRunner.
type ReviewVerdict struct {
	Approved            bool            `json:"approved"`
	ReviewerAgentID     string          `json:"reviewer_agent_id"`
	Findings            []string        `json:"findings,omitempty"`
	StructuredFindings  []ReviewFinding `json:"structured_findings,omitempty"`
	RemediationTips     []string        `json:"remediation_tips,omitempty"`
	ReviewedAt          time.Time       `json:"reviewed_at"`
	ReviewedFingerprint string          `json:"reviewed_fingerprint,omitempty"`
	ReviewIndependence  string          `json:"review_independence,omitempty"`
}

type VerificationResult struct {
	Command       string    `json:"command"`
	Passed        bool      `json:"passed"`
	ExitCode      int       `json:"exit_code"`
	OutputSnippet string    `json:"output_snippet"`
	DurationMs    int64     `json:"duration_ms"`
	VerifiedAt    time.Time `json:"verified_at"`
}

// ContextCapsule is the bounded typed context frozen for one Flow Step before
// prompt compilation/provider dispatch. It intentionally has no raw transcript
// or whole-repository content field.
type ContextCapsule struct {
	ID                        string             `json:"id"`
	RunID                     string             `json:"run_id"`
	ProjectID                 string             `json:"project_id"`
	FlowID                    string             `json:"flow_id"`
	FlowRevision              int                `json:"flow_revision"`
	Branch                    string             `json:"branch,omitempty"`
	Head                      string             `json:"head,omitempty"`
	DirtyFingerprint          string             `json:"dirty_fingerprint,omitempty"`
	Step                      ContextCapsuleStep `json:"step"`
	RelevantPaths             []string           `json:"relevant_paths,omitempty"`
	DurableContextRefs        []string           `json:"durable_context_refs,omitempty"`
	DependencyReceipts        []WorkReceipt      `json:"dependency_receipts,omitempty"`
	SkillIDs                  []string           `json:"skill_ids,omitempty"`
	MaestroSkills             []string           `json:"maestro_skills,omitempty"`
	AcceptanceCriteria        []string           `json:"acceptance_criteria,omitempty"`
	Constraints               []string           `json:"constraints,omitempty"`
	HandoffJSON               string             `json:"handoff,omitempty"`
	BaselineWorkspaceSnapshot map[string]string  `json:"baseline_workspace_snapshot,omitempty"`
	CreatedAt                 time.Time          `json:"created_at"`
}

type ContextCapsuleStep struct {
	ID                       string   `json:"id"`
	Title                    string   `json:"title"`
	Goal                     string   `json:"goal,omitempty"`
	Role                     string   `json:"role,omitempty"`
	Dependencies             []string `json:"dependencies,omitempty"`
	AssignmentStrategy       string   `json:"assignment_strategy,omitempty"`
	VerificationRequirements []string `json:"verification_requirements,omitempty"`
}

// WorkReceipt is factual completion/failure evidence. Empty evidence remains
// empty; provider output is never promoted to proof merely because it says done.
type WorkReceipt struct {
	ID              string               `json:"id"`
	RunID           string               `json:"run_id"`
	StepID          string               `json:"step_id"`
	Status          string               `json:"status"`
	Summary         string               `json:"summary"`
	ChangedFiles    []string             `json:"changed_files"`
	Commands        []string             `json:"commands"`
	Tests           []VerificationResult `json:"tests"`
	Decisions       []string             `json:"decisions"`
	Artifacts       []string             `json:"artifacts"`
	RemainingIssues []string             `json:"remaining_issues"`
	HandoffJSON     string               `json:"handoff,omitempty"`
	Verification    []VerificationResult `json:"verification"`
	AgentID         string               `json:"agent_id,omitempty"`
	BaseRevision    string               `json:"base_revision,omitempty"`
	ResultRevision  string               `json:"result_revision,omitempty"`
	StartedAt       time.Time            `json:"started_at"`
	CompletedAt     time.Time            `json:"completed_at"`
}

type PackageRun struct {
	ID                       string               `json:"id"`
	PackageID                string               `json:"package_id"`
	PhaseID                  string               `json:"phase_id,omitempty"`
	PhaseType                string               `json:"phase_type,omitempty"`
	Title                    string               `json:"title"`
	Goal                     string               `json:"goal,omitempty"`
	Priority                 string               `json:"priority,omitempty"`
	Role                     string               `json:"role,omitempty"`
	ReviewRequired           bool                 `json:"review_required,omitempty"`
	TaskRequirements         string               `json:"task_requirements,omitempty"`
	Dependencies             []string             `json:"dependencies,omitempty"`
	ParallelGroup            string               `json:"parallel_group,omitempty"`
	AssignmentStrategy       string               `json:"assignment_strategy,omitempty"`
	ResourcePolicy           string               `json:"resource_policy,omitempty"`
	Provider                 string               `json:"provider,omitempty"`
	Profile                  string               `json:"profile,omitempty"`
	DesiredProvider          string               `json:"desired_provider,omitempty"`
	DesiredProfile           string               `json:"desired_profile,omitempty"`
	SkillIDs                 []string             `json:"skill_ids,omitempty"`
	MaestroSkills            []string             `json:"maestro_skills,omitempty"`
	RelevantPaths            []string             `json:"relevant_paths,omitempty"`
	AcceptanceCriteria       []string             `json:"acceptance_criteria,omitempty"`
	VerificationRequirements []string             `json:"verification_requirements,omitempty"`
	EntryCriteria            []string             `json:"entry_criteria,omitempty"`
	ExitCriteria             []string             `json:"exit_criteria,omitempty"`
	RequiredEvidence         []string             `json:"required_evidence,omitempty"`
	HandoffJSON              string               `json:"handoff,omitempty"`
	DynamicWork              bool                 `json:"dynamic_work,omitempty"`
	ParentPackageID          string               `json:"parent_package_id,omitempty"`
	CoordinationRequestID    string               `json:"coordination_request_id,omitempty"`
	ResumeAgentID            string               `json:"resume_agent_id,omitempty"`
	ResultSummary            string               `json:"result_summary,omitempty"`
	ContextCapsule           *ContextCapsule      `json:"context_capsule,omitempty"`
	ContextRefreshRequired   bool                 `json:"context_refresh_required,omitempty"`
	WorkReceipt              *WorkReceipt         `json:"work_receipt,omitempty"`
	State                    State                `json:"state"`
	Attempt                  int                  `json:"attempt"`
	AssignedAgent            string               `json:"assigned_agent"`
	AssignedRuntime          string               `json:"assigned_runtime,omitempty"`
	DispatchID               string               `json:"dispatch_id,omitempty"`
	DispatchState            DispatchState        `json:"dispatch_state,omitempty"`
	DispatchStartedAt        *time.Time           `json:"dispatch_started_at,omitempty"`
	DispatchFinishedAt       *time.Time           `json:"dispatch_finished_at,omitempty"`
	Workspace                string               `json:"workspace,omitempty"`
	PromptVersionID          string               `json:"prompt_version_id,omitempty"`
	CompiledPrompt           string               `json:"compiled_prompt,omitempty"`
	RemediationContext       string               `json:"remediation_context,omitempty"`
	StrategyVariant          string               `json:"strategy_variant,omitempty"`
	RetryFrom                State                `json:"retry_from,omitempty"`
	LastFailureSignature     string               `json:"last_failure_signature,omitempty"`
	NoProgressCount          int                  `json:"no_progress_count,omitempty"`
	Verdicts                 []ReviewVerdict      `json:"verdicts,omitempty"`
	ReviewFindings           []ReviewFinding      `json:"review_findings,omitempty"`
	ReviewInvalidated        bool                 `json:"review_invalidated,omitempty"`
	ReviewInvalidationReason string               `json:"review_invalidation_reason,omitempty"`
	Verifications            []VerificationResult `json:"verifications,omitempty"`
	ErrorMessage             string               `json:"error_message,omitempty"`
	FailureKind              model.FailureKind    `json:"failure_kind,omitempty"`
	RoutingDecisionJSON      string               `json:"routing_decision,omitempty"`
	StartedAt                time.Time            `json:"started_at"`
	FinishedAt               *time.Time           `json:"finished_at,omitempty"`
}

// PhaseRun is the durable projection of phase lifecycle. PackageRun remains
// the execution authority; this read model lets UI/CLI show one phase even
// when it contains multiple parallel work packages.
type PhaseRun struct {
	ID             string    `json:"id"`
	Type           string    `json:"type,omitempty"`
	State          State     `json:"state"`
	Attempt        int       `json:"attempt"`
	PackageIDs     []string  `json:"package_ids,omitempty"`
	AgentIDs       []string  `json:"agent_ids,omitempty"`
	RoutingReasons []string  `json:"routing_reasons,omitempty"`
	HandoffJSON    string    `json:"handoff,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// InterventionOperation is a closed set of operations that the core can
// safely interpret. Transport clients submit an option ID, never an arbitrary
// command or prose decision.
type InterventionOperation string

const (
	InterventionRetrySafePackage       InterventionOperation = "RETRY_SAFE_PACKAGE"
	InterventionReplanPackage          InterventionOperation = "REPLAN_PACKAGE"
	InterventionConfirmExternalOutcome InterventionOperation = "CONFIRM_EXTERNAL_COMPLETION"
)

type InterventionOption struct {
	ID        string                `json:"id"`
	Operation InterventionOperation `json:"operation"`
	Label     string                `json:"label"`
	PackageID string                `json:"package_id,omitempty"`
}

type InterventionResolution struct {
	InterventionID string    `json:"intervention_id"`
	Version        int       `json:"version"`
	OptionID       string    `json:"option_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	ResolvedAt     time.Time `json:"resolved_at"`
	ResolvedBy     string    `json:"resolved_by"`
}

type MissionResumeRequest struct {
	ID             string     `json:"id"`
	IdempotencyKey string     `json:"idempotency_key"`
	Status         string     `json:"status"` // PENDING, STARTED, COMPLETED
	RequestedAt    time.Time  `json:"requested_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// HumanIntervention is the durable, actionable contract surfaced whenever a
// mission cannot safely continue autonomously. The summary/error string remains
// available for legacy clients, but UI and recovery code can rely on these
// structured fields instead of parsing prose.
type HumanIntervention struct {
	ID                 string                  `json:"id"`
	ReasonCode         string                  `json:"reason_code"`
	Summary            string                  `json:"summary"`
	Question           string                  `json:"question"`
	Context            string                  `json:"context"`
	RecommendedActions []string                `json:"recommended_actions"`
	Impact             string                  `json:"impact"`
	MissionID          string                  `json:"mission_id"`
	TaskID             string                  `json:"task_id"`
	Source             string                  `json:"source"`
	Timestamp          time.Time               `json:"timestamp"`
	Version            int                     `json:"version"`
	Scope              string                  `json:"scope"`
	Options            []InterventionOption    `json:"options"`
	Resolved           bool                    `json:"resolved"`
	ResolvedAt         *time.Time              `json:"resolved_at,omitempty"`
	ResolutionDecision string                  `json:"resolution_decision,omitempty"`
	ResolutionChosen   string                  `json:"resolution_chosen,omitempty"`
	Resolution         *InterventionResolution `json:"resolution,omitempty"`
}

type MissionRun struct {
	ID                   string                       `json:"id"`
	PlanID               string                       `json:"plan_id"`
	PlanRevision         int                          `json:"plan_revision"`
	ExecutionSnapshotID  string                       `json:"execution_snapshot_id,omitempty"`
	ProjectID            string                       `json:"project_id"`
	LeadAgentID          string                       `json:"lead_agent_id,omitempty"`
	Autonomous           bool                         `json:"autonomous,omitempty"`
	Workspace            string                       `json:"workspace"`
	IntegratedWorkspace  string                       `json:"integrated_workspace,omitempty"`
	DeliveryIntegration  *DeliveryIntegrationEvidence `json:"delivery_integration,omitempty"`
	State                State                        `json:"state"`
	Contract             AutonomyContract             `json:"contract"`
	CurrentPkgIndex      int                          `json:"current_pkg_index"` // compatibility/UI hint only
	CurrentPhaseID       string                       `json:"current_phase_id,omitempty"`
	TotalIterations      int                          `json:"total_iterations"`
	PackageRuns          []PackageRun                 `json:"package_runs"`
	PhaseRuns            []PhaseRun                   `json:"phase_runs,omitempty"`
	LeaseOwner           string                       `json:"lease_owner,omitempty"`
	LeaseToken           string                       `json:"lease_token,omitempty"`
	LeaseExpiresAt       *time.Time                   `json:"lease_expires_at,omitempty"`
	HeartbeatAt          *time.Time                   `json:"heartbeat_at,omitempty"`
	PausedReason         string                       `json:"paused_reason,omitempty"`
	StartedAt            time.Time                    `json:"started_at"`
	LastProgressAt       time.Time                    `json:"last_progress_at,omitempty"`
	UpdatedAt            time.Time                    `json:"updated_at"`
	CompletedAt          *time.Time                   `json:"completed_at,omitempty"`
	GlobalVerifications  []VerificationResult         `json:"global_verifications,omitempty"`
	NeedsHuman           *HumanIntervention           `json:"needs_human,omitempty"`
	ResumeRequest        *MissionResumeRequest        `json:"resume_request,omitempty"`
	CoordinationRequests []CoordinationRequest        `json:"coordination_requests,omitempty"`

	watchFingerprint string `json:"-"`
}

type AllocationResult struct {
	AgentID   string `json:"agent_id"`
	Workspace string `json:"workspace"`
	// Kind indicates the identity kind of the allocated agent (PERSISTENT or EPHEMERAL_WORKER).
	Kind string `json:"kind,omitempty"`
}

// ExecutionIdentityKind enumerates agent identity types.
type ExecutionIdentityKind string

const (
	// ExecutionIdentityPersistent is the default for user-created agents.
	ExecutionIdentityPersistent ExecutionIdentityKind = "PERSISTENT"
	// ExecutionIdentityEphemeralWorker is for throwaway worker agents used by the Conductor.
	ExecutionIdentityEphemeralWorker ExecutionIdentityKind = "EPHEMERAL_WORKER"
)

// ExecutionIdentity carries the identity kind through the execution envelope.
type ExecutionIdentity struct {
	Kind ExecutionIdentityKind `json:"kind"`
}

type PromptArtifact struct {
	VersionID string `json:"version_id"`
	Content   string `json:"content"`
}

type ExecutionResult struct {
	RuntimeID                string                `json:"runtime_id"`
	Output                   string                `json:"output,omitempty"`
	Failure                  model.Failure         `json:"failure,omitempty"`
	CoordinationRequests     []CoordinationRequest `json:"coordination_requests,omitempty"`
	CoordinationReplaySafe   bool                  `json:"coordination_replay_safe,omitempty"`
	CoordinationUnsafeReason string                `json:"coordination_unsafe_reason,omitempty"`
}
