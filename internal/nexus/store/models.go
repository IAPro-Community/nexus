package store

import (
	"encoding/json"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
)

// Continuity statuses (§24 honest continuity model).
const (
	ContinuityLiveSameRuntime        = "LIVE_SAME_RUNTIME"
	ContinuityReattachedSameRuntime  = "REATTACHED_SAME_RUNTIME"
	ContinuityNativeResumeVerified   = "NATIVE_RESUME_VERIFIED"
	ContinuityNativeResumeUnverified = "NATIVE_RESUME_UNVERIFIED"
	ContinuityContextRecovered       = "CONTEXT_RECOVERED_NEW_SESSION"
	ContinuityNewSession             = "NEW_SESSION"
	ContinuityFailed                 = "CONTINUITY_FAILED"
)

// Maestro modes (§52).
const (
	MaestroOff         = "OFF"
	MaestroAssist      = "ASSIST"
	MaestroOrchestrate = "ORCHESTRATE"
)

// Agent lifecycle states (§40, §29). RECOVERABLE/RECOVERING describe agents whose
// runtime died (e.g. machine reboot): the Agent persists, the Runtime does not.
const (
	AgentStopped     = "STOPPED"
	AgentStarting    = "STARTING"
	AgentWorking     = "WORKING"
	AgentWaiting     = "WAITING"
	AgentApproval    = "APPROVAL"
	AgentRateLimited = "RATE_LIMITED"
	AgentDetached    = "DETACHED"
	AgentReconfig    = "RECONFIGURING"
	AgentHandoff     = "HANDOFF"
	AgentRecoverable = "RECOVERABLE"
	AgentRecovering  = "RECOVERING"
	AgentStopping    = "STOPPING"
	AgentFailed      = "FAILED"
	AgentStale       = "STALE"
)

// Project is the root of the Nexus domain (§14-15). Identity is a stable
// ULID-based ID, never derived from basename/path.
type Project struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Slug             string         `json:"slug"`
	CanonicalPath    string         `json:"canonical_path"`
	DisplayPath      string         `json:"display_path,omitempty"`
	IdentityKind     string         `json:"identity_kind,omitempty"`
	IdentityKey      string         `json:"identity_key,omitempty"`
	PathRef          config.PathRef `json:"path_ref,omitempty"`
	RepoRemote       string         `json:"repo_remote"`
	RepoURL          string         `json:"repo_url"`
	DefaultBranch    string         `json:"default_branch"`
	MaestroMode      string         `json:"maestro_mode"`
	ResourcePolicy   string         `json:"resource_policy"` // JSON object
	DefaultIsolation string         `json:"default_isolation"`
	Settings         string         `json:"settings"` // JSON object
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	LastOpenedAt     *time.Time     `json:"last_opened_at,omitempty"`
}

// Agent is the primary operational unit (§19-21). AgentID is stable across
// runtime restarts, account/provider changes, and terminal reconnects.
type Agent struct {
	ID                string     `json:"id"`
	ProjectID         string     `json:"project_id"`
	Name              string     `json:"name"`
	Role              string     `json:"role"`
	Status            string     `json:"status"`
	CurrentRevisionID string     `json:"current_revision_id"`
	ContinuityStatus  string     `json:"continuity_status"`
	IdentityKind      string     `json:"identity_kind,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	LastStartedAt     *time.Time `json:"last_started_at,omitempty"`
}

// AgentRevision is an immutable, revisioned agent configuration (§23).
type AgentRevision struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	Revision  int       `json:"revision"`
	Config    string    `json:"config"` // JSON object
	CreatedAt time.Time `json:"created_at"`
}

// RuntimeGeneration links an agent to a concrete runtime incarnation (§22).
type RuntimeGeneration struct {
	ID              string     `json:"id"`
	AgentID         string     `json:"agent_id"`
	RevisionID      string     `json:"revision_id"`
	RuntimeID       string     `json:"runtime_id"`
	Provider        string     `json:"provider"`
	Profile         string     `json:"profile"`
	ProviderSession string     `json:"provider_session"`
	Continuity      string     `json:"continuity"`
	StartedAt       time.Time  `json:"started_at"`
	StoppedAt       *time.Time `json:"stopped_at,omitempty"`
	State           string     `json:"state"`
}

// LineageEntry records an account or context handoff edge (§39).
type LineageEntry struct {
	ID            string    `json:"id"`
	AgentID       string    `json:"agent_id"`
	Relation      string    `json:"relation"` // ACCOUNT_HANDOFF | CONTEXT_HANDOFF
	SourceRuntime string    `json:"source_runtime"`
	SourceSession string    `json:"source_session"`
	TargetRuntime string    `json:"target_runtime"`
	TargetSession string    `json:"target_session"`
	CheckpointID  string    `json:"checkpoint_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// ProjectLayout is the persisted per-project cockpit layout (§34).
type ProjectLayout struct {
	ProjectID string    `json:"project_id"`
	Layout    string    `json:"layout"` // JSON object
	UpdatedAt time.Time `json:"updated_at"`
}

// MustJSON is a small helper for JSON-typed columns.
func MustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// Mission lifecycle states (Gate 7 Beta).
const (
	MissionDraft     = "DRAFT"
	MissionPlanning  = "PLANNING"
	MissionReady     = "READY"
	MissionActive    = "ACTIVE"
	MissionPaused    = "PAUSED"
	MissionCompleted = "COMPLETED"
	MissionFailed    = "FAILED"
	MissionCancelled = "CANCELED"
)

// MissionTask lifecycle states.
const (
	TaskPending   = "PENDING"
	TaskReady     = "READY"
	TaskActive    = "ACTIVE"
	TaskBlocked   = "BLOCKED"
	TaskCompleted = "COMPLETED"
	TaskFailed    = "FAILED"
	TaskSkipped   = "SKIPPED"
)

// Mission is a high-level goal分解 into tasks (Gate 7 Beta).
type Mission struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Goal        string     `json:"goal"`
	Scope       string     `json:"scope"`      // "project" | "agent" | "task"
	RiskLevel   string     `json:"risk_level"` // "low" | "medium" | "high"
	Config      string     `json:"config"`     // JSON object
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// MissionTask is a single unit of work within a mission.
type MissionTask struct {
	ID           string     `json:"id"`
	MissionID    string     `json:"mission_id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Status       string     `json:"status"`
	Kind         string     `json:"kind"` // "action" | "config" | "security" | "verify"
	Priority     int        `json:"priority"`
	Dependencies string     `json:"dependencies"` // JSON array of task IDs
	Config       string     `json:"config"`       // JSON object
	Result       string     `json:"result"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// MissionAssignment links a task to an agent.
type MissionAssignment struct {
	ID          string     `json:"id"`
	MissionID   string     `json:"mission_id"`
	TaskID      string     `json:"task_id"`
	AgentID     string     `json:"agent_id"`
	Status      string     `json:"status"` // "ASSIGNED" | "ACCEPTED" | "REJECTED" | "COMPLETED"
	AssignedAt  time.Time  `json:"assigned_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
