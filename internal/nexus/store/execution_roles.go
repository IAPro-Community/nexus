package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

type RuntimeExecutionRoleBinding struct {
	ID                  string    `json:"id"`
	RuntimeGenerationID string    `json:"runtime_generation_id"`
	AgentID             string    `json:"agent_id"`
	RoleID              string    `json:"role_id"`
	Scope               string    `json:"scope"`
	ActivationStrategy  string    `json:"activation_strategy"`
	Enforcement           string     `json:"enforcement"`
	EnforcementEvidence   string     `json:"enforcement_evidence,omitempty"`
	EnforcementVerifiedAt *time.Time `json:"enforcement_verified_at,omitempty"`
	ProjectionReceiptID   string     `json:"projection_receipt_id,omitempty"`
	ActivatedBy           string     `json:"activated_by,omitempty"`
	ActivatedAt           time.Time  `json:"activated_at"`
}

// SetRuntimeExecutionRole persists the operational role of a concrete runtime
// generation. It never mutates Agent or AgentRevision.
func (s *Store) SetRuntimeExecutionRole(binding RuntimeExecutionRoleBinding) (RuntimeExecutionRoleBinding, error) {
	if strings.TrimSpace(binding.RuntimeGenerationID) == "" {
		return RuntimeExecutionRoleBinding{}, errors.New("runtime_generation_id is required")
	}
	if strings.TrimSpace(binding.RoleID) == "" {
		return RuntimeExecutionRoleBinding{}, errors.New("role_id is required")
	}

	var generationAgentID string
	if err := s.db.QueryRow(`SELECT agent_id FROM runtime_generations WHERE id=?`, binding.RuntimeGenerationID).Scan(&generationAgentID); err != nil {
		if err == sql.ErrNoRows {
			return RuntimeExecutionRoleBinding{}, ErrNotFound
		}
		return RuntimeExecutionRoleBinding{}, err
	}
	if binding.AgentID == "" {
		binding.AgentID = generationAgentID
	}
	if binding.AgentID != generationAgentID {
		return RuntimeExecutionRoleBinding{}, fmt.Errorf("runtime generation belongs to agent %s, not %s", generationAgentID, binding.AgentID)
	}
	if existing, existingErr := s.RuntimeExecutionRole(binding.RuntimeGenerationID); existingErr == nil {
		binding.ID = existing.ID
	} else if existingErr != ErrNotFound {
		return RuntimeExecutionRoleBinding{}, existingErr
	} else if binding.ID == "" {
		binding.ID = "xrole_" + ids.NewRuntimeID()
	}
	if binding.Scope == "" {
		binding.Scope = "RUNTIME"
	}
	if binding.Scope != "RUNTIME" {
		return RuntimeExecutionRoleBinding{}, fmt.Errorf("durable execution role scope must be RUNTIME, got %q", binding.Scope)
	}
	if binding.ActivationStrategy == "" {
		binding.ActivationStrategy = "NEXUS_OVERLAY"
	}
	if binding.Enforcement == "" {
		binding.Enforcement = "UNVERIFIED"
	}
	if binding.ActivatedAt.IsZero() {
		binding.ActivatedAt = time.Now().UTC()
	}

	_, err := s.db.Exec(`INSERT INTO runtime_execution_roles
		(id,runtime_generation_id,agent_id,role_id,scope,activation_strategy,enforcement,projection_receipt_id,activated_by,activated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(runtime_generation_id) DO UPDATE SET
			agent_id=excluded.agent_id,
			role_id=excluded.role_id,
			scope=excluded.scope,
			activation_strategy=excluded.activation_strategy,
			enforcement=excluded.enforcement,
			projection_receipt_id=excluded.projection_receipt_id,
			activated_by=excluded.activated_by,
			activated_at=excluded.activated_at`,
		binding.ID, binding.RuntimeGenerationID, binding.AgentID, binding.RoleID, binding.Scope,
		binding.ActivationStrategy, binding.Enforcement, binding.ProjectionReceiptID,
		binding.ActivatedBy, binding.ActivatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return RuntimeExecutionRoleBinding{}, fmt.Errorf("set runtime execution role: %w", err)
	}
	persisted, err := s.RuntimeExecutionRole(binding.RuntimeGenerationID)
	if err != nil {
		return RuntimeExecutionRoleBinding{}, fmt.Errorf("reload runtime execution role: %w", err)
	}
	return persisted, nil
}

func (s *Store) RuntimeExecutionRole(runtimeGenerationID string) (RuntimeExecutionRoleBinding, error) {
	var binding RuntimeExecutionRoleBinding
	var activatedAt string
	var enforcementVerifiedAt sql.NullString
	err := s.db.QueryRow(`SELECT id,runtime_generation_id,agent_id,role_id,scope,activation_strategy,enforcement,enforcement_evidence,enforcement_verified_at,projection_receipt_id,activated_by,activated_at
		FROM runtime_execution_roles WHERE runtime_generation_id=?`, runtimeGenerationID).
		Scan(&binding.ID, &binding.RuntimeGenerationID, &binding.AgentID, &binding.RoleID, &binding.Scope,
			&binding.ActivationStrategy, &binding.Enforcement, &binding.EnforcementEvidence, &enforcementVerifiedAt,
			&binding.ProjectionReceiptID, &binding.ActivatedBy, &activatedAt)
	if err == sql.ErrNoRows {
		return RuntimeExecutionRoleBinding{}, ErrNotFound
	}
	if err != nil {
		return RuntimeExecutionRoleBinding{}, err
	}
	binding.ActivatedAt, _ = time.Parse(time.RFC3339Nano, activatedAt)
	if enforcementVerifiedAt.Valid {
		if parsed, parseErr := time.Parse(time.RFC3339Nano, enforcementVerifiedAt.String); parseErr == nil {
			binding.EnforcementVerifiedAt = &parsed
		}
	}
	return binding, nil
}

func (s *Store) RuntimeExecutionRoleByRuntimeID(runtimeID string) (RuntimeExecutionRoleBinding, error) {
	var generationID string
	err := s.db.QueryRow(`SELECT id FROM runtime_generations WHERE runtime_id=? LIMIT 1`, runtimeID).Scan(&generationID)
	if err == sql.ErrNoRows {
		return RuntimeExecutionRoleBinding{}, ErrNotFound
	}
	if err != nil {
		return RuntimeExecutionRoleBinding{}, err
	}
	return s.RuntimeExecutionRole(generationID)
}


// AttachRuntimeExecutionRoleProjection links a materialized native-agent
// projection to the runtime role without changing the role or claiming native
// activation. ActivationStrategy/enforcement remain owned by the execution-role
// service and provider evidence.
func (s *Store) AttachRuntimeExecutionRoleProjection(runtimeGenerationID, receiptID string) error {
	runtimeGenerationID = strings.TrimSpace(runtimeGenerationID)
	receiptID = strings.TrimSpace(receiptID)
	if runtimeGenerationID == "" || receiptID == "" {
		return errors.New("runtime_generation_id and projection receipt id are required")
	}
	res, err := s.db.Exec(`UPDATE runtime_execution_roles
		SET projection_receipt_id=?
		WHERE runtime_generation_id=?`, receiptID, runtimeGenerationID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DetachRuntimeExecutionRoleProjection clears the projection link only when the
// caller still owns the current receipt. This prevents an old/superseded
// receipt from detaching a newer native projection.
func (s *Store) DetachRuntimeExecutionRoleProjection(runtimeGenerationID, receiptID string) error {
	runtimeGenerationID = strings.TrimSpace(runtimeGenerationID)
	receiptID = strings.TrimSpace(receiptID)
	if runtimeGenerationID == "" || receiptID == "" {
		return errors.New("runtime_generation_id and projection receipt id are required")
	}
	res, err := s.db.Exec(`UPDATE runtime_execution_roles
		SET projection_receipt_id='',
			activation_strategy='NEXUS_OVERLAY',
			enforcement=CASE WHEN enforcement='ENFORCED' THEN 'BEST_EFFORT' ELSE enforcement END,
			enforcement_evidence='',
			enforcement_verified_at=NULL
		WHERE runtime_generation_id=? AND projection_receipt_id=?`, runtimeGenerationID, receiptID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}


// MarkRuntimeExecutionRoleActivation records native activation evidence without
// changing enforcement. The receipt predicate prevents stale evidence from
// promoting a newer projection accidentally.
func (s *Store) MarkRuntimeExecutionRoleActivation(runtimeGenerationID, receiptID, strategy string) error {
	runtimeGenerationID = strings.TrimSpace(runtimeGenerationID)
	receiptID = strings.TrimSpace(receiptID)
	strategy = strings.TrimSpace(strategy)
	if runtimeGenerationID == "" || receiptID == "" || strategy == "" {
		return errors.New("runtime_generation_id, projection receipt id and activation strategy are required")
	}
	res, err := s.db.Exec(`UPDATE runtime_execution_roles
		SET activation_strategy=?
		WHERE runtime_generation_id=? AND projection_receipt_id=?`,
		strategy, runtimeGenerationID, receiptID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}


// MarkRuntimeExecutionRoleEnforced promotes a read-only runtime role only after
// provider-backed evidence proves that the active projection actually rejected
// a write attempt. The current receipt and native activation are mandatory so
// stale or overlay-only evidence cannot promote enforcement.
func (s *Store) MarkRuntimeExecutionRoleEnforced(runtimeGenerationID, receiptID, evidence string, verifiedAt time.Time) error {
	runtimeGenerationID = strings.TrimSpace(runtimeGenerationID)
	receiptID = strings.TrimSpace(receiptID)
	evidence = strings.TrimSpace(evidence)
	if runtimeGenerationID == "" || receiptID == "" || evidence == "" {
		return errors.New("runtime_generation_id, projection receipt id and enforcement evidence are required")
	}
	if verifiedAt.IsZero() {
		verifiedAt = time.Now().UTC()
	}
	res, err := s.db.Exec(`UPDATE runtime_execution_roles
		SET enforcement='ENFORCED',
			enforcement_evidence=?,
			enforcement_verified_at=?
		WHERE runtime_generation_id=?
		  AND projection_receipt_id=?
		  AND activation_strategy='NATIVE_ON_LAUNCH'`,
		evidence,
		verifiedAt.UTC().Format(time.RFC3339Nano),
		runtimeGenerationID,
		receiptID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
