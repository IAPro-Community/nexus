package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type AgentProjectionReceiptRecord struct {
	ID                  string     `json:"id"`
	RuntimeGenerationID string     `json:"runtime_generation_id"`
	AgentID             string     `json:"agent_id"`
	Provider            string     `json:"provider"`
	Profile             string     `json:"profile"`
	RoleID              string     `json:"role_id"`
	Scope               string     `json:"scope"`
	Strategy            string     `json:"strategy"`
	ManagedRoot         string     `json:"managed_root"`
	RendererVersion     string     `json:"renderer_version"`
	SourceHash          string     `json:"source_hash"`
	PayloadJSON         string     `json:"payload_json"`
	AppliedAt           time.Time  `json:"applied_at"`
	VerifiedAt          *time.Time `json:"verified_at,omitempty"`
	RemovedAt           *time.Time `json:"removed_at,omitempty"`
}

func (s *Store) SaveAgentProjectionReceipt(record AgentProjectionReceiptRecord) error {
	if strings.TrimSpace(record.ID) == "" {
		return errors.New("agent projection receipt id is required")
	}
	if strings.TrimSpace(record.RuntimeGenerationID) == "" {
		return errors.New("runtime_generation_id is required")
	}
	if strings.TrimSpace(record.AgentID) == "" {
		return errors.New("agent_id is required")
	}
	if strings.TrimSpace(record.Provider) == "" {
		return errors.New("provider is required")
	}
	if strings.TrimSpace(record.PayloadJSON) == "" {
		return errors.New("payload_json is required")
	}
	if record.AppliedAt.IsZero() {
		record.AppliedAt = time.Now().UTC()
	}

	var generationAgentID string
	err := s.db.QueryRow(`SELECT agent_id FROM runtime_generations WHERE id=?`, record.RuntimeGenerationID).Scan(&generationAgentID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if generationAgentID != record.AgentID {
		return fmt.Errorf("runtime generation belongs to agent %s, not %s", generationAgentID, record.AgentID)
	}

	_, err = s.db.Exec(`INSERT INTO agent_projection_receipts
		(id,runtime_generation_id,agent_id,provider,profile,role_id,scope,strategy,managed_root,renderer_version,source_hash,payload_json,applied_at,verified_at,removed_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			payload_json=excluded.payload_json,
			strategy=excluded.strategy,
			managed_root=excluded.managed_root,
			renderer_version=excluded.renderer_version,
			source_hash=excluded.source_hash,
			verified_at=excluded.verified_at,
			removed_at=excluded.removed_at`,
		record.ID,
		record.RuntimeGenerationID,
		record.AgentID,
		record.Provider,
		record.Profile,
		record.RoleID,
		record.Scope,
		record.Strategy,
		record.ManagedRoot,
		record.RendererVersion,
		record.SourceHash,
		record.PayloadJSON,
		record.AppliedAt.UTC().Format(time.RFC3339Nano),
		nullableTime(record.VerifiedAt),
		nullableTime(record.RemovedAt),
	)
	if err != nil {
		return fmt.Errorf("save agent projection receipt: %w", err)
	}
	return nil
}

func (s *Store) AgentProjectionReceipt(id string) (AgentProjectionReceiptRecord, error) {
	var record AgentProjectionReceiptRecord
	var appliedAt string
	var verifiedAt, removedAt sql.NullString
	err := s.db.QueryRow(`SELECT
		id,runtime_generation_id,agent_id,provider,profile,role_id,scope,strategy,managed_root,renderer_version,source_hash,payload_json,applied_at,verified_at,removed_at
		FROM agent_projection_receipts WHERE id=?`, id).Scan(
		&record.ID,
		&record.RuntimeGenerationID,
		&record.AgentID,
		&record.Provider,
		&record.Profile,
		&record.RoleID,
		&record.Scope,
		&record.Strategy,
		&record.ManagedRoot,
		&record.RendererVersion,
		&record.SourceHash,
		&record.PayloadJSON,
		&appliedAt,
		&verifiedAt,
		&removedAt,
	)
	if err == sql.ErrNoRows {
		return AgentProjectionReceiptRecord{}, ErrNotFound
	}
	if err != nil {
		return AgentProjectionReceiptRecord{}, err
	}
	record.AppliedAt, _ = time.Parse(time.RFC3339Nano, appliedAt)
	if verifiedAt.Valid {
		if parsed, parseErr := time.Parse(time.RFC3339Nano, verifiedAt.String); parseErr == nil {
			record.VerifiedAt = &parsed
		}
	}
	if removedAt.Valid {
		if parsed, parseErr := time.Parse(time.RFC3339Nano, removedAt.String); parseErr == nil {
			record.RemovedAt = &parsed
		}
	}
	return record, nil
}

func (s *Store) MarkAgentProjectionVerified(id string, verifiedAt time.Time, payloadJSON string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("agent projection receipt id is required")
	}
	if verifiedAt.IsZero() {
		verifiedAt = time.Now().UTC()
	}
	res, err := s.db.Exec(`UPDATE agent_projection_receipts
		SET verified_at=?, payload_json=CASE WHEN ?='' THEN payload_json ELSE ? END
		WHERE id=? AND removed_at IS NULL`,
		verifiedAt.UTC().Format(time.RFC3339Nano), payloadJSON, payloadJSON, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) MarkAgentProjectionRemoved(id string, removedAt time.Time) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("agent projection receipt id is required")
	}
	if removedAt.IsZero() {
		removedAt = time.Now().UTC()
	}
	res, err := s.db.Exec(`UPDATE agent_projection_receipts SET removed_at=? WHERE id=? AND removed_at IS NULL`,
		removedAt.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func nullableTime(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}
