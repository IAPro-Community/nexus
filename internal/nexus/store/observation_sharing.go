package store

import (
	"fmt"
	"time"
)

// ObservationShareGrantRecord is the durable mirror of one viewer grant
// (workspace-sharing phase 2). The in-memory registry remains the
// authorization authority; this row lets grants survive a host restart.
type ObservationShareGrantRecord struct {
	ID        string
	ProjectID string
	AgentID   string
	Role      string
	CreatedAt time.Time
	ExpiresAt *time.Time
	RevokedAt *time.Time
}

// UpsertObservationShareGrant inserts or updates one grant row. It is the
// write-through path for Grant and Revoke; it never deletes rows (revoke
// stamps revoked_at so the history stays auditable).
func (s *Store) UpsertObservationShareGrant(rec ObservationShareGrantRecord) error {
	if rec.ID == "" || rec.ProjectID == "" || rec.Role == "" {
		return fmt.Errorf("observation share grant requires id, project and role")
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}
	expiresAt := ""
	if rec.ExpiresAt != nil {
		expiresAt = formatStoreTime(*rec.ExpiresAt)
	}
	revokedAt := ""
	if rec.RevokedAt != nil {
		revokedAt = formatStoreTime(*rec.RevokedAt)
	}
	_, err := s.db.Exec(`INSERT INTO observation_share_grants(id,project_id,agent_id,role,created_at,expires_at,revoked_at)
		VALUES(?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			agent_id=excluded.agent_id,
			role=excluded.role,
			expires_at=excluded.expires_at,
			revoked_at=excluded.revoked_at`,
		rec.ID, rec.ProjectID, rec.AgentID, rec.Role,
		formatStoreTime(rec.CreatedAt), expiresAt, revokedAt)
	if err != nil {
		return fmt.Errorf("upsert observation share grant: %w", err)
	}
	return nil
}

// ListObservationShareGrants returns every durable grant ordered by creation.
func (s *Store) ListObservationShareGrants() ([]ObservationShareGrantRecord, error) {
	rows, err := s.db.Query(`SELECT id,project_id,agent_id,role,created_at,expires_at,revoked_at
		FROM observation_share_grants ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list observation share grants: %w", err)
	}
	defer rows.Close()
	out := make([]ObservationShareGrantRecord, 0)
	for rows.Next() {
		var rec ObservationShareGrantRecord
		var createdAt, expiresAt, revokedAt string
		if err := rows.Scan(&rec.ID, &rec.ProjectID, &rec.AgentID, &rec.Role, &createdAt, &expiresAt, &revokedAt); err != nil {
			return nil, fmt.Errorf("scan observation share grant: %w", err)
		}
		rec.CreatedAt, err = parseStoreTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse observation share grant created_at: %w", err)
		}
		if expiresAt != "" {
			parsed, err := parseStoreTime(expiresAt)
			if err != nil {
				return nil, fmt.Errorf("parse observation share grant expires_at: %w", err)
			}
			rec.ExpiresAt = &parsed
		}
		if revokedAt != "" {
			parsed, err := parseStoreTime(revokedAt)
			if err != nil {
				return nil, fmt.Errorf("parse observation share grant revoked_at: %w", err)
			}
			rec.RevokedAt = &parsed
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate observation share grants: %w", err)
	}
	return out, nil
}
