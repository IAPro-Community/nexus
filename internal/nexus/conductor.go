package nexus

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// ─────────────────────────────────────────────────────────────────────
// § C.1  ConductorPolicy (nexus_settings, key="conductor:policy")
// ─────────────────────────────────────────────────────────────────────

// ConductorPolicy is the persisted policy document for the Nexus Conductor.
// It is stored as a versioned JSON blob in nexus_settings under key
// "conductor:policy". A nil pointer means "use computed defaults".
type ConductorPolicy struct {
	Version int `json:"version"`
	// ReviewMode selects the read_only_review capability:
	//   nil = fall back to computed defaults based on detected capabilities.
	ReviewMode *string `json:"review_mode,omitempty"`
	// RetentionDays controls event purge (default 90).
	RetentionDays *int `json:"retention_days,omitempty"`
	// IntegrationOptIn enables the integration service (fast-forward).
	IntegrationOptIn *bool `json:"integration_opt_in,omitempty"`
}

const conductorPolicyKey = "conductor:policy"

// GetConductorPolicy reads the global policy from the store. When the key is
// absent or the document is invalid, a computed default is returned.
func GetConductorPolicy(st *store.Store) (*ConductorPolicy, error) {
	raw, err := st.GetSetting(conductorPolicyKey)
	if errors.Is(err, store.ErrNotFound) {
		return DefaultConductorPolicy(), nil
	}
	if err != nil {
		return nil, err
	}
	var p ConductorPolicy
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return DefaultConductorPolicy(), nil // invalid doc → fall back
	}
	applyDefaults(&p)
	return &p, nil
}

// PutConductorPolicy persists the policy document. It is idempotent.
func PutConductorPolicy(st *store.Store, p *ConductorPolicy) error {
	if p == nil {
		p = DefaultConductorPolicy()
	}
	applyDefaults(p)
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("encode conductor policy: %w", err)
	}
	return st.PutSetting(conductorPolicyKey, string(data))
}

func DefaultConductorPolicy() *ConductorPolicy {
	p := &ConductorPolicy{Version: 1}
	applyDefaults(p)
	return p
}

func applyDefaults(p *ConductorPolicy) {
	if p == nil {
		return
	}
	if p.Version == 0 {
		p.Version = 1
	}
	if p.RetentionDays == nil || *p.RetentionDays <= 0 {
		d := 90
		p.RetentionDays = &d
	}
	if p.IntegrationOptIn == nil {
		v := false
		p.IntegrationOptIn = &v
	}
}

// ─────────────────────────────────────────────────────────────────────
// § C.2  GoalContract (frozen copy of promoted goal)
// ─────────────────────────────────────────────────────────────────────

// GoalContractFactKey stores the immutable promoted goal in WorkPlan facts.
const GoalContractFactKey = "nexus.dispatch.goal_contract"

// GoalContract captures the immutable goal definition at promotion time.
// It is frozen into the ExecutionSnapshot.
type GoalContract struct {
	GoalDescription string `json:"goal_description"`
	// AcceptedAssumptions captures reviewer-approved assumptions.
	AcceptedAssumptions []string `json:"accepted_assumptions,omitempty"`
	// AutoPromoteToMission means: if the goal needs phase execution,
	// the Conductor will auto-promote to a Mission without user confirm.
	AutoPromoteToMission bool `json:"auto_promote_to_mission"`
	// LeadID is the interactive lead that owns this goal.
	LeadID string `json:"lead_id"`
	// ProjectID is the owning project.
	ProjectID string `json:"project_id"`
	// GoalID is the unique goal identifier (for idempotency).
	GoalID string `json:"goal_id"`
	// CreatedAt is the time the goal was created.
	CreatedAt time.Time `json:"created_at"`
}

func PersistGoalContractFact(existing map[string]string, goal GoalContract) (map[string]string, error) {
	raw, err := json.Marshal(goal)
	if err != nil {
		return nil, fmt.Errorf("encode goal contract: %w", err)
	}
	facts := make(map[string]string, len(existing)+1)
	for key, value := range existing {
		facts[key] = value
	}
	facts[GoalContractFactKey] = string(raw)
	return facts, nil
}

func ReadGoalContractFact(facts map[string]string) (*GoalContract, error) {
	raw := strings.TrimSpace(facts[GoalContractFactKey])
	if raw == "" {
		return nil, nil
	}
	var goal GoalContract
	if err := json.Unmarshal([]byte(raw), &goal); err != nil {
		return nil, fmt.Errorf("decode goal contract: %w", err)
	}
	if strings.TrimSpace(goal.GoalID) == "" || strings.TrimSpace(goal.ProjectID) == "" || strings.TrimSpace(goal.GoalDescription) == "" {
		return nil, fmt.Errorf("goal contract is missing goal_id, project_id, or goal_description")
	}
	return &goal, nil
}

// ─────────────────────────────────────────────────────────────────────
// § C.3  GoalAssessment (pass/fail verdict after read_only_review)
// ─────────────────────────────────────────────────────────────────────

// GoalAssessmentResult enumerates possible assessment outcomes.
type GoalAssessmentResult string

const (
	GoalAssessmentPass GoalAssessmentResult = "PASS"
	GoalAssessmentFail GoalAssessmentResult = "FAIL"
	GoalAssessmentRedo GoalAssessmentResult = "REDO"
)

// GoalAssessment is the structured verdict produced by read_only_review.
type GoalAssessment struct {
	Result GoalAssessmentResult `json:"result"`
	// Summary is a human-readable assessment summary.
	Summary string `json:"summary"`
	// Violations lists specific invariant violations (empty when PASS).
	Violations []string `json:"violations,omitempty"`
	// Assumptions records any assumptions the reviewer accepted.
	Assumptions []string `json:"assumptions,omitempty"`
	// AssessedAt is when the assessment was produced.
	AssessedAt time.Time `json:"assessed_at"`
}

func (g GoalAssessment) String() string {
	switch g.Result {
	case GoalAssessmentPass:
		return "PASS"
	case GoalAssessmentFail:
		return "FAIL"
	case GoalAssessmentRedo:
		return "REDO"
	default:
		return "UNKNOWN"
	}
}

// ─────────────────────────────────────────────────────────────────────
// § C.4  AgentInvocation (execution record per goal)
// ─────────────────────────────────────────────────────────────────────

// AgentInvocationStatus enumerates invocation lifecycle states.
type AgentInvocationStatus string

const (
	AgentInvocationRunning   AgentInvocationStatus = "RUNNING"
	AgentInvocationCompleted AgentInvocationStatus = "COMPLETED"
	AgentInvocationFailed    AgentInvocationStatus = "FAILED"
	AgentInvocationCancelled AgentInvocationStatus = "CANCELED"
)

// AgentInvocation records a single goal execution attempt.
// Stored as JSON in event metadata (EventsMetaKey = "agent_invocation").
type AgentInvocation struct {
	GoalID     string                `json:"goal_id"`
	AgentID    string                `json:"agent_id"`
	ProjectID  string                `json:"project_id"`
	Status     AgentInvocationStatus `json:"status"`
	StartedAt  time.Time             `json:"started_at"`
	FinishedAt *time.Time            `json:"finished_at,omitempty"`
	Assessment *GoalAssessment       `json:"assessment,omitempty"`
	Errors     []string              `json:"errors,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────
// § C.5  IntegrationReceipt (optimistic pull)
// ─────────────────────────────────────────────────────────────────────

// IntegrationReceipt records a single integration attempt outcome.
type IntegrationReceipt struct {
	BaseSHA  string    `json:"base_sha"`
	HeadSHA  string    `json:"head_sha"`
	Status   string    `json:"status"` // "SUCCESS" | "CONFLICT"
	Error    string    `json:"error,omitempty"`
	PulledAt time.Time `json:"pulled_at"`
}
