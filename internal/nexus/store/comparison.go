package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

const (
	ComparisonPending   = "PENDING"
	ComparisonSelectA   = "SELECT_A"
	ComparisonSelectB   = "SELECT_B"
	ComparisonRejectAll = "REJECT_BOTH"
)

// ComparisonLabRecord links two completed runs from the same immutable plan
// revision. Decisions are durable review metadata and never mutate workspaces.
type ComparisonLabRecord struct {
	ID           string     `json:"id"`
	ProjectID    string     `json:"project_id"`
	PlanID       string     `json:"plan_id"`
	PlanRevision int        `json:"plan_revision"`
	RunAID       string     `json:"run_a_id"`
	RunBID       string     `json:"run_b_id"`
	EvidenceJSON string     `json:"-"`
	Decision     string     `json:"decision"`
	Reason       string     `json:"decision_reason,omitempty"`
	DecidedAt    *time.Time `json:"decided_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (s *Store) CreateComparisonLab(record ComparisonLabRecord) (*ComparisonLabRecord, error) {
	if strings.TrimSpace(record.ProjectID) == "" || strings.TrimSpace(record.PlanID) == "" ||
		record.PlanRevision <= 0 || strings.TrimSpace(record.RunAID) == "" || strings.TrimSpace(record.RunBID) == "" ||
		record.RunAID == record.RunBID {
		return nil, fmt.Errorf("comparison requires project, plan revision, and two distinct runs")
	}
	if !json.Valid([]byte(record.EvidenceJSON)) {
		return nil, fmt.Errorf("comparison requires a valid evidence snapshot")
	}
	if record.ID == "" {
		record.ID = "comparison_" + ids.NewRuntimeID()
	}
	now := time.Now().UTC()
	record.Decision = ComparisonPending
	record.CreatedAt = now
	record.UpdatedAt = now
	_, err := s.db.Exec(`INSERT INTO comparison_labs
		(id, project_id, plan_id, plan_revision, run_a_id, run_b_id, evidence_json, decision, decision_reason, decided_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, '', NULL, ?, ?)`,
		record.ID, record.ProjectID, record.PlanID, record.PlanRevision, record.RunAID, record.RunBID,
		record.EvidenceJSON, record.Decision, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("create comparison: %w", err)
	}
	return &record, nil
}

func (s *Store) GetComparisonLab(id string) (*ComparisonLabRecord, error) {
	return scanComparisonLab(s.db.QueryRow(`SELECT id, project_id, plan_id, plan_revision, run_a_id, run_b_id,
		evidence_json, decision, decision_reason, decided_at, created_at, updated_at FROM comparison_labs WHERE id=?`, id))
}

func (s *Store) ListComparisonLabs(projectID string) ([]ComparisonLabRecord, error) {
	rows, err := s.db.Query(`SELECT id, project_id, plan_id, plan_revision, run_a_id, run_b_id,
		evidence_json, decision, decision_reason, decided_at, created_at, updated_at FROM comparison_labs
		WHERE project_id=? ORDER BY created_at DESC LIMIT 50`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ComparisonLabRecord, 0)
	for rows.Next() {
		item, scanErr := scanComparisonLab(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (s *Store) DecideComparisonLab(id, decision, reason string) (*ComparisonLabRecord, error) {
	decision = strings.ToUpper(strings.TrimSpace(decision))
	reason = strings.TrimSpace(reason)
	if decision != ComparisonSelectA && decision != ComparisonSelectB && decision != ComparisonRejectAll {
		return nil, fmt.Errorf("decision must be SELECT_A, SELECT_B, or REJECT_BOTH")
	}
	if reason == "" || len(reason) > 2000 {
		return nil, fmt.Errorf("decision reason is required and must not exceed 2000 characters")
	}
	now := time.Now().UTC()
	result, err := s.db.Exec(`UPDATE comparison_labs SET decision=?, decision_reason=?, decided_at=?, updated_at=?
		WHERE id=? AND decision=?`, decision, reason, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), id, ComparisonPending)
	if err != nil {
		return nil, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if changed == 0 {
		current, getErr := s.GetComparisonLab(id)
		if getErr != nil {
			return nil, getErr
		}
		if current.Decision != decision || current.Reason != reason {
			return nil, fmt.Errorf("comparison already has a different human decision")
		}
		return current, nil
	}
	return s.GetComparisonLab(id)
}

type comparisonScanner interface{ Scan(...any) error }

func scanComparisonLab(row comparisonScanner) (*ComparisonLabRecord, error) {
	var item ComparisonLabRecord
	var decided sql.NullString
	var created, updated string
	if err := row.Scan(&item.ID, &item.ProjectID, &item.PlanID, &item.PlanRevision,
		&item.RunAID, &item.RunBID, &item.EvidenceJSON, &item.Decision, &item.Reason, &decided, &created, &updated); err != nil {
		return nil, err
	}
	item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	if decided.Valid {
		value, err := time.Parse(time.RFC3339Nano, decided.String)
		if err == nil {
			item.DecidedAt = &value
		}
	}
	return &item, nil
}
