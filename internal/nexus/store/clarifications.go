package store

import (
	"fmt"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

const (
	ClarificationPending   = "PENDING"
	ClarificationResolved  = "RESOLVED"
	ClarificationCancelled = "CANCELED"
)

// Clarification persists an Intelligence ambiguity checkpoint without importing
// the intelligence package into the store layer. Structured payloads are JSON.
type Clarification struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	Goal         string    `json:"goal"`
	Status       string    `json:"status"`
	IntentJSON   string    `json:"intent_json"`
	UnknownsJSON string    `json:"unknowns_json"`
	FactsJSON    string    `json:"facts_json"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (s *Store) CreateClarification(c Clarification) (*Clarification, error) {
	if c.ID == "" {
		c.ID = "clr_" + ids.NewRuntimeID()
	}
	if c.Status == "" {
		c.Status = ClarificationPending
	}
	if c.IntentJSON == "" {
		c.IntentJSON = "{}"
	}
	if c.UnknownsJSON == "" {
		c.UnknownsJSON = "[]"
	}
	if c.FactsJSON == "" {
		c.FactsJSON = "{}"
	}
	now := time.Now().UTC()
	c.CreatedAt, c.UpdatedAt = now, now
	_, err := s.db.Exec(`INSERT INTO clarifications(id,project_id,goal,status,intent_json,unknowns_json,facts_json,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?)`, c.ID, c.ProjectID, c.Goal, c.Status, c.IntentJSON, c.UnknownsJSON, c.FactsJSON,
		now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("create clarification: %w", err)
	}
	return &c, nil
}

func (s *Store) GetClarification(id string) (*Clarification, error) {
	var c Clarification
	var created, updated string
	err := s.db.QueryRow(`SELECT id,project_id,goal,status,intent_json,unknowns_json,facts_json,created_at,updated_at
		FROM clarifications WHERE id=?`, id).Scan(&c.ID, &c.ProjectID, &c.Goal, &c.Status, &c.IntentJSON, &c.UnknownsJSON, &c.FactsJSON, &created, &updated)
	if err != nil {
		return nil, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &c, nil
}

func (s *Store) UpdateClarification(c Clarification) error {
	c.UpdatedAt = time.Now().UTC()
	res, err := s.db.Exec(`UPDATE clarifications SET status=?,intent_json=?,unknowns_json=?,facts_json=?,updated_at=? WHERE id=?`,
		c.Status, c.IntentJSON, c.UnknownsJSON, c.FactsJSON, c.UpdatedAt.Format(time.RFC3339Nano), c.ID)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListClarifications(projectID, status string) ([]Clarification, error) {
	query := `SELECT id,project_id,goal,status,intent_json,unknowns_json,facts_json,created_at,updated_at FROM clarifications WHERE project_id=?`
	args := []any{projectID}
	if status != "" {
		query += ` AND status=?`
		args = append(args, status)
	}
	query += ` ORDER BY updated_at DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Clarification
	for rows.Next() {
		var c Clarification
		var created, updated string
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Goal, &c.Status, &c.IntentJSON, &c.UnknownsJSON, &c.FactsJSON, &created, &updated); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, c)
	}
	return out, rows.Err()
}
