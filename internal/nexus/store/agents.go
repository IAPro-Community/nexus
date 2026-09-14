package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

const agentColumns = `id,project_id,name,role,status,current_revision_id,continuity_status,created_at,updated_at,last_started_at`

// CreateAgent inserts a persistent agent scoped to a project.
func (s *Store) CreateAgent(a Agent) (Agent, error) {
	if a.ProjectID == "" {
		return Agent{}, errors.New("project_id is required")
	}
	if _, err := s.GetProject(a.ProjectID); err != nil {
		return Agent{}, fmt.Errorf("agent project must exist: %w", err)
	}
	if a.ID == "" {
		a.ID = "agt_" + ids.NewRuntimeID()
	}
	if strings.TrimSpace(a.Name) == "" {
		return Agent{}, errors.New("agent name is required")
	}
	if a.Status == "" {
		a.Status = "STOPPED"
	}
	if a.ContinuityStatus == "" {
		a.ContinuityStatus = ContinuityLiveSameRuntime
	}
	now := time.Now().UTC()
	a.CreatedAt, a.UpdatedAt = now, now

	_, err := s.db.Exec(`INSERT INTO agents
		(id,project_id,name,role,status,current_revision_id,continuity_status,created_at,updated_at,last_started_at)
		VALUES(?,?,?,?,?,?,?,?,?,NULL)`,
		a.ID, a.ProjectID, a.Name, a.Role, a.Status, a.CurrentRevisionID, a.ContinuityStatus,
		a.CreatedAt.Format(time.RFC3339Nano), a.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Agent{}, fmt.Errorf("create agent: %w", err)
	}
	return a, nil
}

// GetAgent loads an agent by ID, verifying it belongs to projectID when given.
func (s *Store) GetAgent(id, projectID string) (Agent, error) {
	a, err := s.scanAgent(s.db.QueryRow(`SELECT `+agentColumns+` FROM agents WHERE id=?`, id))
	if err != nil {
		return Agent{}, err
	}
	if projectID != "" && a.ProjectID != projectID {
		return Agent{}, ErrNotFound // IDOR guard: agent must be scoped to the caller project
	}
	return a, nil
}

// ListAgents returns agents of a project, most recently updated first.
func (s *Store) ListAgents(projectID string) ([]Agent, error) {
	rows, err := s.db.Query(`SELECT `+agentColumns+` FROM agents WHERE project_id=? ORDER BY updated_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out = []Agent{}
	for rows.Next() {
		a, err := scanAgentRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ProjectAgentSummary is a lightweight per-project agent count for the project rail.
type ProjectAgentSummary struct {
	ProjectID    string `json:"project_id"`
	AgentCount   int    `json:"agent_count"`
	WorkingCount int    `json:"working_count"`
}

// ListAgentSummaries returns agent_count and working_count grouped by project.
// Projects with zero agents are omitted; the client merges against the project list.
func (s *Store) ListAgentSummaries() ([]ProjectAgentSummary, error) {
	rows, err := s.db.Query(`
		SELECT project_id,
			COUNT(*) AS agent_count,
			SUM(CASE WHEN status = 'WORKING' THEN 1 ELSE 0 END) AS working_count
		FROM agents
		GROUP BY project_id
		ORDER BY project_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ProjectAgentSummary{}
	for rows.Next() {
		var summary ProjectAgentSummary
		if err := rows.Scan(&summary.ProjectID, &summary.AgentCount, &summary.WorkingCount); err != nil {
			return nil, err
		}
		out = append(out, summary)
	}
	return out, rows.Err()
}

// UpdateAgent patches mutable agent fields by ID (project scoped).
func (s *Store) UpdateAgent(a Agent) error {
	a.UpdatedAt = time.Now().UTC()
	_, err := s.db.Exec(`UPDATE agents SET name=?,role=?,status=?,current_revision_id=?,continuity_status=?,updated_at=?
		WHERE id=? AND project_id=?`,
		a.Name, a.Role, a.Status, a.CurrentRevisionID, a.ContinuityStatus,
		a.UpdatedAt.Format(time.RFC3339Nano), a.ID, a.ProjectID)
	return err
}

// SetAgentStatus updates only the lifecycle status of an agent.
func (s *Store) SetAgentStatus(id, status string) error {
	_, err := s.db.Exec(`UPDATE agents SET status=?, updated_at=? WHERE id=?`,
		status, time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}

// ClaimInteractiveLead atomically reserves one idle interactive Lead. The
// STARTING transition closes the read/modify/write race between concurrent
// supervised launches; callers that lose the update must use another Lead.
func (s *Store) ClaimInteractiveLead(projectID string) (Agent, bool, error) {
	agents, err := s.ListAgents(projectID)
	if err != nil {
		return Agent{}, false, err
	}
	for _, candidate := range agents {
		if candidate.Role != "lead-engineer" {
			continue
		}
		res, err := s.db.Exec(`UPDATE agents SET status=?, updated_at=?
			WHERE id=? AND project_id=? AND role=? AND status IN (?,?,?,?)`,
			AgentStarting, time.Now().UTC().Format(time.RFC3339Nano), candidate.ID, projectID,
			"lead-engineer", AgentStopped, AgentDetached, AgentRecoverable, AgentWorking)
		if err != nil {
			return Agent{}, false, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return Agent{}, false, err
		}
		if n == 1 {
			claimed, err := s.GetAgent(candidate.ID, projectID)
			return claimed, true, err
		}
	}
	return Agent{}, false, nil
}

// ReleaseInteractiveLeadReservation returns a pre-launch reservation to the
// idle pool. It is conditional so cleanup cannot reset a later runtime.
func (s *Store) ReleaseInteractiveLeadReservation(id string) error {
	_, err := s.db.Exec(`UPDATE agents SET status=?, updated_at=? WHERE id=? AND status=?`,
		AgentStopped, time.Now().UTC().Format(time.RFC3339Nano), id, AgentStarting)
	return err
}

// DeleteAgent removes an agent (cascades revisions/generations/lineage).
func (s *Store) DeleteAgent(id, projectID string) error {
	res, err := s.db.Exec(`DELETE FROM agents WHERE id=? AND project_id=?`, id, projectID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// NextRevision returns the next revision number for an agent.
func (s *Store) NextRevision(agentID string) (int, error) {
	var max sql.NullInt64
	if err := s.db.QueryRow(`SELECT MAX(revision) FROM agent_revisions WHERE agent_id=?`, agentID).Scan(&max); err != nil {
		return 0, err
	}
	if !max.Valid {
		return 1, nil
	}
	return int(max.Int64) + 1, nil
}

// AddRevision stores a new immutable config revision for an agent.
func (s *Store) AddRevision(agentID, config string) (AgentRevision, error) {
	rev, err := s.NextRevision(agentID)
	if err != nil {
		return AgentRevision{}, err
	}
	ar := AgentRevision{
		ID:        "rev_" + ids.NewRuntimeID(),
		AgentID:   agentID,
		Revision:  rev,
		Config:    config,
		CreatedAt: time.Now().UTC(),
	}
	if ar.Config == "" {
		ar.Config = "{}"
	}
	_, err = s.db.Exec(`INSERT INTO agent_revisions(id,agent_id,revision,config,created_at) VALUES(?,?,?,?,?)`,
		ar.ID, ar.AgentID, ar.Revision, ar.Config, ar.CreatedAt.Format(time.RFC3339Nano))
	return ar, err
}

// GetRevision loads a config revision by ID.
func (s *Store) GetRevision(id string) (AgentRevision, error) {
	var ar AgentRevision
	var createdAt string
	err := s.db.QueryRow(`SELECT id,agent_id,revision,config,created_at FROM agent_revisions WHERE id=?`, id).
		Scan(&ar.ID, &ar.AgentID, &ar.Revision, &ar.Config, &createdAt)
	if err == sql.ErrNoRows {
		return AgentRevision{}, ErrNotFound
	}
	if err != nil {
		return AgentRevision{}, err
	}
	ar.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return ar, nil
}

// ListRevisions returns all revisions of an agent, newest first.
func (s *Store) ListRevisions(agentID string) ([]AgentRevision, error) {
	rows, err := s.db.Query(`SELECT id,agent_id,revision,config,created_at FROM agent_revisions WHERE agent_id=? ORDER BY revision DESC`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out = []AgentRevision{}
	for rows.Next() {
		var ar AgentRevision
		var createdAt string
		if err := rows.Scan(&ar.ID, &ar.AgentID, &ar.Revision, &ar.Config, &createdAt); err != nil {
			return nil, err
		}
		ar.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		out = append(out, ar)
	}
	return out, rows.Err()
}

// AddGeneration records a runtime incarnation of an agent.
func (s *Store) AddGeneration(g RuntimeGeneration) (RuntimeGeneration, error) {
	if g.ID == "" {
		g.ID = "gen_" + ids.NewRuntimeID()
	}
	if g.State == "" {
		g.State = "RUNNING"
	}
	if g.Continuity == "" {
		g.Continuity = ContinuityLiveSameRuntime
	}
	_, err := s.db.Exec(`INSERT INTO runtime_generations
		(id,agent_id,revision_id,runtime_id,provider,profile,provider_session,continuity,started_at,stopped_at,state)
		VALUES(?,?,?,?,?,?,?,?,?,NULL,?)`,
		g.ID, g.AgentID, g.RevisionID, g.RuntimeID, g.Provider, g.Profile, g.ProviderSession,
		g.Continuity, g.StartedAt.UTC().Format(time.RFC3339Nano), g.State)
	return g, err
}

// ListGenerations returns an agent's runtime generations, newest first.
func (s *Store) ListGenerations(agentID string) ([]RuntimeGeneration, error) {
	rows, err := s.db.Query(`SELECT id,agent_id,revision_id,runtime_id,provider,profile,provider_session,continuity,started_at,stopped_at,state
		FROM runtime_generations WHERE agent_id=? ORDER BY started_at DESC`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGenerations(rows)
}

// CurrentGeneration returns the latest RUNNING/active generation of an agent.
func (s *Store) CurrentGeneration(agentID string) (RuntimeGeneration, error) {
	rows, err := s.db.Query(`SELECT id,agent_id,revision_id,runtime_id,provider,profile,provider_session,continuity,started_at,stopped_at,state
		FROM runtime_generations WHERE agent_id=? ORDER BY started_at DESC LIMIT 1`, agentID)
	if err != nil {
		return RuntimeGeneration{}, err
	}
	defer rows.Close()
	gens, err := scanGenerations(rows)
	if err != nil {
		return RuntimeGeneration{}, err
	}
	if len(gens) == 0 {
		return RuntimeGeneration{}, ErrNotFound
	}
	return gens[0], nil
}

// GenerationByRuntimeID returns the generation that owns the given runtime ID.
func (s *Store) GenerationByRuntimeID(runtimeID string) (RuntimeGeneration, error) {
	rows, err := s.db.Query(`SELECT id,agent_id,revision_id,runtime_id,provider,profile,provider_session,continuity,started_at,stopped_at,state
		FROM runtime_generations WHERE runtime_id=? LIMIT 1`, runtimeID)
	if err != nil {
		return RuntimeGeneration{}, err
	}
	defer rows.Close()
	gens, err := scanGenerations(rows)
	if err != nil {
		return RuntimeGeneration{}, err
	}
	if len(gens) == 0 {
		return RuntimeGeneration{}, ErrNotFound
	}
	return gens[0], nil
}

// AddLineage records an account/context handoff edge for an agent.
func (s *Store) AddLineage(l LineageEntry) error {
	if l.ID == "" {
		l.ID = "lin_" + ids.NewRuntimeID()
	}
	_, err := s.db.Exec(`INSERT INTO lineage(id,agent_id,relation,source_runtime,source_session,target_runtime,target_session,checkpoint_id,created_at)
		VALUES(?,?,?,?,?,?,?,?,?)`,
		l.ID, l.AgentID, l.Relation, l.SourceRuntime, l.SourceSession, l.TargetRuntime, l.TargetSession,
		l.CheckpointID, l.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

// ListLineage returns an agent's lineage, newest first.
func (s *Store) ListLineage(agentID string) ([]LineageEntry, error) {
	rows, err := s.db.Query(`SELECT id,agent_id,relation,source_runtime,source_session,target_runtime,target_session,checkpoint_id,created_at
		FROM lineage WHERE agent_id=? ORDER BY created_at DESC`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out = []LineageEntry{}
	for rows.Next() {
		var l LineageEntry
		var createdAt string
		if err := rows.Scan(&l.ID, &l.AgentID, &l.Relation, &l.SourceRuntime, &l.SourceSession,
			&l.TargetRuntime, &l.TargetSession, &l.CheckpointID, &createdAt); err != nil {
			return nil, err
		}
		l.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		out = append(out, l)
	}
	return out, rows.Err()
}

// StopGeneration marks a runtime generation as stopped.
func (s *Store) StopGeneration(id string, stoppedAt time.Time) error {
	_, err := s.db.Exec(`UPDATE runtime_generations SET stopped_at=?, state='STOPPED' WHERE id=?`,
		stoppedAt.UTC().Format(time.RFC3339Nano), id)
	return err
}

func (s *Store) scanAgent(row *sql.Row) (Agent, error) { return scanAgentRows(row) }

type agentScanner interface{ Scan(dest ...any) error }

func scanAgentRows(row agentScanner) (Agent, error) {
	var a Agent
	var createdAt, updatedAt string
	var lastStarted sql.NullString
	err := row.Scan(&a.ID, &a.ProjectID, &a.Name, &a.Role, &a.Status, &a.CurrentRevisionID,
		&a.ContinuityStatus, &createdAt, &updatedAt, &lastStarted)
	if err == sql.ErrNoRows {
		return Agent{}, ErrNotFound
	}
	if err != nil {
		return Agent{}, err
	}
	a.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	a.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	if lastStarted.Valid {
		if t, err := time.Parse(time.RFC3339Nano, lastStarted.String); err == nil {
			a.LastStartedAt = &t
		}
	}
	return a, nil
}

func scanGenerations(rows *sql.Rows) ([]RuntimeGeneration, error) {
	var out = []RuntimeGeneration{}
	for rows.Next() {
		var g RuntimeGeneration
		var startedAt, state string
		var stoppedAt sql.NullString
		if err := rows.Scan(&g.ID, &g.AgentID, &g.RevisionID, &g.RuntimeID, &g.Provider, &g.Profile,
			&g.ProviderSession, &g.Continuity, &startedAt, &stoppedAt, &state); err != nil {
			return nil, err
		}
		g.StartedAt, _ = time.Parse(time.RFC3339Nano, startedAt)
		g.State = state
		if stoppedAt.Valid {
			if t, err := time.Parse(time.RFC3339Nano, stoppedAt.String); err == nil {
				g.StoppedAt = &t
			}
		}
		out = append(out, g)
	}
	return out, rows.Err()
}
