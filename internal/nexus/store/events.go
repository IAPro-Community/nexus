package store

import (
	"fmt"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

// EventMetadata represents persistent activity metadata in SQLite.
type EventMetadata struct {
	ID            string             `json:"id"`
	AgentID       string             `json:"agent_id"`
	ProjectID     string             `json:"project_id"`
	CorrelationID string             `json:"correlation_id,omitempty"`
	Kind          string             `json:"kind"`
	Timestamp     time.Time          `json:"ts"`
	Summary       string             `json:"summary"`
	AccountScope  model.AccountScope `json:"account_scope,omitempty"`
}

const eventMetadataColumns = `id,agent_id,project_id,correlation_id,kind,ts,summary,account_provider_id,account_id,identity_version`

// RecordEventMetadata records an activity event into the durable SQLite store.
func (s *Store) RecordEventMetadata(e EventMetadata) (EventMetadata, error) {
	if e.ID == "" {
		e.ID = "evt_" + ids.NewRuntimeID()
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}

	_, err := s.db.Exec(`INSERT INTO events_metadata (id, agent_id, project_id, correlation_id, kind, ts, summary, account_provider_id, account_id, identity_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.AgentID, e.ProjectID, e.CorrelationID, e.Kind, e.Timestamp.Format(time.RFC3339Nano), e.Summary,
		e.AccountScope.ProviderID, e.AccountScope.AccountID, e.AccountScope.IdentityVersion)
	if err != nil {
		return EventMetadata{}, fmt.Errorf("record event metadata: %w", err)
	}
	return e, nil
}

// ListEventsMetadata retrieves activity events optionally filtered by project or agent.
func (s *Store) ListEventsMetadata(projectID, agentID string, limit int) ([]EventMetadata, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT ` + eventMetadataColumns + ` FROM events_metadata`
	var args []any
	var conditions []string

	if projectID != "" {
		conditions = append(conditions, "project_id = ?")
		args = append(args, projectID)
	}
	if agentID != "" {
		conditions = append(conditions, "agent_id = ?")
		args = append(args, agentID)
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	query += " ORDER BY ts DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list events metadata: %w", err)
	}
	defer rows.Close()

	var eventsList []EventMetadata
	for rows.Next() {
		var em EventMetadata
		var tsStr string
		if err := rows.Scan(&em.ID, &em.AgentID, &em.ProjectID, &em.CorrelationID, &em.Kind, &tsStr, &em.Summary, &em.AccountScope.ProviderID, &em.AccountScope.AccountID, &em.AccountScope.IdentityVersion); err != nil {
			return nil, fmt.Errorf("scan event metadata: %w", err)
		}
		if t, err := time.Parse(time.RFC3339Nano, tsStr); err == nil {
			em.Timestamp = t
		} else if t, err := time.Parse(time.RFC3339, tsStr); err == nil {
			em.Timestamp = t
		}
		eventsList = append(eventsList, em)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events metadata: %w", err)
	}
	if eventsList == nil {
		eventsList = []EventMetadata{}
	}
	return eventsList, nil
}
