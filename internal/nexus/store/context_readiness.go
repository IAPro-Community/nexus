package store

import (
	"encoding/json"
	"fmt"
	"time"
)

type ContextReadinessRecord struct {
	ProjectID       string    `json:"project_id"`
	State           string    `json:"state"`
	FingerprintJSON string    `json:"fingerprint_json"`
	Reason          string    `json:"reason"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (s *Store) GetContextReadiness(projectID string) (*ContextReadinessRecord, error) {
	var r ContextReadinessRecord
	var updated string
	err := s.db.QueryRow(`SELECT project_id,state,fingerprint_json,reason,updated_at FROM context_readiness WHERE project_id=?`, projectID).Scan(&r.ProjectID, &r.State, &r.FingerprintJSON, &r.Reason, &updated)
	if err != nil {
		return nil, err
	}
	r.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &r, nil
}

func (s *Store) SaveContextReadiness(projectID, state string, fingerprint any, reason string) (*ContextReadinessRecord, error) {
	b, err := json.Marshal(fingerprint)
	if err != nil {
		return nil, fmt.Errorf("marshal context fingerprint: %w", err)
	}
	now := time.Now().UTC()
	_, err = s.db.Exec(`INSERT INTO context_readiness(project_id,state,fingerprint_json,reason,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(project_id) DO UPDATE SET state=excluded.state,fingerprint_json=excluded.fingerprint_json,reason=excluded.reason,updated_at=excluded.updated_at`, projectID, state, string(b), reason, now.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	return s.GetContextReadiness(projectID)
}
