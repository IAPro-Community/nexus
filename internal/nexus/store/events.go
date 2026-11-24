package store

import (
	"fmt"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

// EventMetadata represents persistent activity metadata in SQLite.
type EventMetadata struct {
	ID            string `json:"id"`
	AgentID       string `json:"agent_id"`
	ProjectID     string `json:"project_id"`
	CorrelationID string `json:"correlation_id,omitempty"`
	Source        string `json:"source,omitempty"`
	RuntimeID     string `json:"runtime_id,omitempty"`
	// RuntimeGenerationID links the event to the durable runtime generation
	// that owned the runtime when the event was recorded.
	RuntimeGenerationID string             `json:"runtime_generation_id,omitempty"`
	Kind                string             `json:"kind"`
	Timestamp           time.Time          `json:"ts"`
	Summary             string             `json:"summary"`
	AccountScope        model.AccountScope `json:"account_scope,omitempty"`
}

const eventMetadataColumns = `id,agent_id,project_id,correlation_id,source,runtime_id,runtime_generation_id,kind,ts,summary,account_provider_id,account_id,identity_version`

// RecordEventMetadata records an activity event into the durable SQLite store.
func (s *Store) RecordEventMetadata(e EventMetadata) (EventMetadata, error) {
	if e.ID == "" {
		e.ID = "evt_" + ids.NewRuntimeID()
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}

	_, err := s.db.Exec(`INSERT INTO events_metadata (id, agent_id, project_id, correlation_id, source, runtime_id, runtime_generation_id, kind, ts, summary, account_provider_id, account_id, identity_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.AgentID, e.ProjectID, e.CorrelationID, e.Source, e.RuntimeID, e.RuntimeGenerationID, e.Kind, e.Timestamp.Format(time.RFC3339Nano), e.Summary,
		e.AccountScope.ProviderID, e.AccountScope.AccountID, e.AccountScope.IdentityVersion)
	if err != nil {
		return EventMetadata{}, fmt.Errorf("record event metadata: %w", err)
	}
	return e, nil
}

// Timeline identity expressions resolve the durable generation, Agent and
// project of a row from the runtime that emitted it. Rows recorded before the
// generation commit keep their original values and never receive a fabricated
// identity.
const (
	timelineGenerationExpr = `IFNULL(COALESCE(NULLIF(e.runtime_generation_id,''), (SELECT g.id FROM runtime_generations g WHERE g.runtime_id = e.runtime_id ORDER BY g.started_at DESC, g.id DESC LIMIT 1)),'')`
	timelineAgentExpr      = `IFNULL(COALESCE(NULLIF(e.agent_id,''), (SELECT g.agent_id FROM runtime_generations g WHERE g.runtime_id = e.runtime_id ORDER BY g.started_at DESC, g.id DESC LIMIT 1)),'')`
	timelineProjectExpr    = `IFNULL(COALESCE(NULLIF(e.project_id,''), (SELECT a.project_id FROM agents a WHERE a.id = ` + timelineAgentExpr + `)),'')`
)

// ListEventsMetadata retrieves activity events optionally filtered by project or agent.
func (s *Store) ListEventsMetadata(projectID, agentID string, limit int) ([]EventMetadata, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT e.id, ` + timelineAgentExpr + `, ` + timelineProjectExpr + `,
		e.correlation_id, e.source, e.runtime_id,
		` + timelineGenerationExpr + `,
		e.kind, e.ts, e.summary, e.account_provider_id, e.account_id, e.identity_version
		FROM events_metadata e`
	var args []any
	var conditions []string

	if projectID != "" {
		conditions = append(conditions, timelineProjectExpr+" = ?")
		args = append(args, projectID)
	}
	if agentID != "" {
		conditions = append(conditions, timelineAgentExpr+" = ?")
		args = append(args, agentID)
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	query += " ORDER BY e.ts DESC LIMIT ?"
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
		if err := rows.Scan(&em.ID, &em.AgentID, &em.ProjectID, &em.CorrelationID, &em.Source, &em.RuntimeID, &em.RuntimeGenerationID, &em.Kind, &tsStr, &em.Summary, &em.AccountScope.ProviderID, &em.AccountScope.AccountID, &em.AccountScope.IdentityVersion); err != nil {
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
