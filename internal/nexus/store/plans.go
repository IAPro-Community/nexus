package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

// WorkPlan is the structured, versioned root of engineering planning (Phase D).
type WorkPlan struct {
	ID              string            `json:"id"`
	ProjectID       string            `json:"project_id"`
	MissionID       string            `json:"mission_id,omitempty"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	Status          string            `json:"status"` // "DRAFT" | "READY" | "EXECUTING" | "COMPLETED" | "BLOCKED"
	CurrentRevision int               `json:"current_revision"`
	Phases          []PlanPhase       `json:"phases"`
	StructuredFacts map[string]string `json:"structured_facts,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// PlanPhase groups sequential or related WorkPackages.
type PlanPhase struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Description   string        `json:"description,omitempty"`
	Order         int           `json:"order"`
	Type          string        `json:"type,omitempty"`
	RequiredRole  string        `json:"required_role,omitempty"`
	FailurePolicy string        `json:"failure_policy,omitempty"`
	Packages      []WorkPackage `json:"packages"`
}

// WorkPackage is a concrete, reviewable unit of autonomous engineering work.
type WorkPackage struct {
	ID                       string   `json:"id"`
	Title                    string   `json:"title"`
	Goal                     string   `json:"goal"`
	Priority                 string   `json:"priority"` // "CRITICAL" | "HIGH" | "NORMAL" | "LOW"
	Status                   string   `json:"status"`   // "PENDING" | "READY" | "ALLOCATING" | "COMPILING" | "EXECUTING" | "TESTING" | "REVIEWING" | "VERIFIED" | "FAILED" | "BLOCKED"
	Dependencies             []string `json:"dependencies"`
	ParallelGroup            string   `json:"parallel_group,omitempty"`
	Role                     string   `json:"role"` // "implementer" | "reviewer" | "tester" | "architect"
	PhaseType                string   `json:"phase_type,omitempty"`
	ReviewRequired           bool     `json:"review_required,omitempty"`
	TaskRequirements         string   `json:"task_requirements,omitempty"`
	AgentAllocation          string   `json:"agent_allocation,omitempty"`
	AssignmentStrategy       string   `json:"assignment_strategy,omitempty"` // EXISTING | CREATE | AUTO
	ResourcePolicy           string   `json:"resource_policy,omitempty"`
	Provider                 string   `json:"provider,omitempty"`
	Profile                  string   `json:"profile,omitempty"`
	DesiredProvider          string   `json:"desired_provider,omitempty"`
	DesiredProfile           string   `json:"desired_profile,omitempty"`
	SkillIDs                 []string `json:"skill_ids,omitempty"`
	MaestroGates             []string `json:"maestro_gates,omitempty"` // legacy compatibility
	MaestroSkills            []string `json:"maestro_skills,omitempty"`
	RelevantPaths            []string `json:"relevant_paths,omitempty"`
	AcceptanceCriteria       []string `json:"acceptance_criteria"`
	VerificationRequirements []string `json:"verification_requirements,omitempty"`
	RoutingDecisionJSON      string   `json:"routing_decision,omitempty"`
	SharedArtifacts          []string `json:"shared_artifacts,omitempty"`
	EntryCriteria            []string `json:"entry_criteria,omitempty"`
	ExitCriteria             []string `json:"exit_criteria,omitempty"`
	RequiredEvidence         []string `json:"required_evidence,omitempty"`
	HandoffJSON              string   `json:"handoff,omitempty"`
	CompiledPrompt           string   `json:"compiled_prompt,omitempty"`
}

// PlanRevision records an immutable revision of a WorkPlan for safe diff/restore.
type PlanRevision struct {
	ID            string    `json:"id"`
	PlanID        string    `json:"plan_id"`
	Revision      int       `json:"revision"`
	SnapshotJSON  string    `json:"snapshot_json"`
	ChangeSummary string    `json:"change_summary"`
	CreatedAt     time.Time `json:"created_at"`
}

// ExecutionSnapshot captures the full state of active runs and verification evidence.
type ExecutionSnapshot struct {
	ID         string    `json:"id"`
	PlanID     string    `json:"plan_id"`
	RevisionID string    `json:"revision_id"`
	StateJSON  string    `json:"state_json"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *Store) CreateWorkPlan(p WorkPlan) (*WorkPlan, error) {
	if p.ID == "" {
		p.ID = "plan_" + ids.NewRuntimeID()
	}
	if p.Status == "" {
		p.Status = "DRAFT"
	}
	if p.CurrentRevision == 0 {
		p.CurrentRevision = 1
	}
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	phasesJSON, _ := json.Marshal(p.Phases)
	if p.Phases == nil {
		phasesJSON = []byte("[]")
	}
	factsJSON, _ := json.Marshal(p.StructuredFacts)
	if p.StructuredFacts == nil {
		factsJSON = []byte("{}")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var missionIDVal any
	if p.MissionID != "" {
		missionIDVal = p.MissionID
	}

	_, err = tx.Exec(`INSERT INTO work_plans(id, project_id, mission_id, title, description, status, current_revision, phases_json, facts_json, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.ProjectID, missionIDVal, p.Title, p.Description, p.Status, p.CurrentRevision, string(phasesJSON), string(factsJSON),
		now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("insert work plan: %w", err)
	}

	// Create initial Revision 1
	planBytes, _ := json.Marshal(p)
	revID := "rev_" + ids.NewRuntimeID()
	_, err = tx.Exec(`INSERT INTO plan_revisions(id, plan_id, revision, snapshot_json, change_summary, created_at)
		VALUES(?, ?, ?, ?, ?, ?)`,
		revID, p.ID, 1, string(planBytes), "Initial plan creation", now.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("create initial plan revision: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &p, nil
}

func (s *Store) GetWorkPlan(id string) (*WorkPlan, error) {
	var p WorkPlan
	var phasesRaw, factsRaw, createdAtStr, updatedAtStr string
	var missionID sql.NullString

	err := s.db.QueryRow(`SELECT id, project_id, mission_id, title, description, status, current_revision, phases_json, facts_json, created_at, updated_at
		FROM work_plans WHERE id=?`, id).Scan(
		&p.ID, &p.ProjectID, &missionID, &p.Title, &p.Description, &p.Status, &p.CurrentRevision,
		&phasesRaw, &factsRaw, &createdAtStr, &updatedAtStr)
	if err != nil {
		return nil, err
	}

	if missionID.Valid {
		p.MissionID = missionID.String
	}
	_ = json.Unmarshal([]byte(phasesRaw), &p.Phases)
	_ = json.Unmarshal([]byte(factsRaw), &p.StructuredFacts)
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &p, nil
}

func (s *Store) ListWorkPlans(projectID string) ([]WorkPlan, error) {
	rows, err := s.db.Query(`SELECT id, project_id, mission_id, title, description, status, current_revision, phases_json, facts_json, created_at, updated_at
		FROM work_plans WHERE project_id=? ORDER BY updated_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []WorkPlan
	for rows.Next() {
		var p WorkPlan
		var phasesRaw, factsRaw, createdAtStr, updatedAtStr string
		var missionID sql.NullString
		if err := rows.Scan(&p.ID, &p.ProjectID, &missionID, &p.Title, &p.Description, &p.Status, &p.CurrentRevision,
			&phasesRaw, &factsRaw, &createdAtStr, &updatedAtStr); err != nil {
			return nil, err
		}
		if missionID.Valid {
			p.MissionID = missionID.String
		}
		_ = json.Unmarshal([]byte(phasesRaw), &p.Phases)
		_ = json.Unmarshal([]byte(factsRaw), &p.StructuredFacts)
		p.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		list = append(list, p)
	}
	return list, nil
}

func (s *Store) UpdateWorkPlan(p WorkPlan, changeSummary string) (*WorkPlan, *PlanRevision, error) {
	now := time.Now().UTC()
	expectedRevision := p.CurrentRevision
	p.UpdatedAt = now
	p.CurrentRevision = expectedRevision + 1

	phasesJSON, _ := json.Marshal(p.Phases)
	factsJSON, _ := json.Marshal(p.StructuredFacts)

	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	result, err := tx.Exec(`UPDATE work_plans SET title=?, description=?, status=?, current_revision=?, phases_json=?, facts_json=?, updated_at=?
		WHERE id=? AND current_revision=?`,
		p.Title, p.Description, p.Status, p.CurrentRevision, string(phasesJSON), string(factsJSON),
		now.Format(time.RFC3339), p.ID, expectedRevision)
	if err != nil {
		return nil, nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, nil, err
	}
	if rowsAffected == 0 {
		var currentRevision int
		if err := tx.QueryRow(`SELECT current_revision FROM work_plans WHERE id=?`, p.ID).Scan(&currentRevision); err != nil {
			return nil, nil, err
		}
		if err := validateExpectedPlanRevision(expectedRevision, currentRevision); err != nil {
			return nil, nil, err
		}
		return nil, nil, ErrPlanRevisionConflict
	}

	planBytes, _ := json.Marshal(p)
	revID := "rev_" + ids.NewRuntimeID()
	if changeSummary == "" {
		changeSummary = fmt.Sprintf("Revision %d update", p.CurrentRevision)
	}
	rev := PlanRevision{
		ID:            revID,
		PlanID:        p.ID,
		Revision:      p.CurrentRevision,
		SnapshotJSON:  string(planBytes),
		ChangeSummary: changeSummary,
		CreatedAt:     now,
	}

	_, err = tx.Exec(`INSERT INTO plan_revisions(id, plan_id, revision, snapshot_json, change_summary, created_at)
		VALUES(?, ?, ?, ?, ?, ?)`,
		rev.ID, rev.PlanID, rev.Revision, rev.SnapshotJSON, rev.ChangeSummary, now.Format(time.RFC3339))
	if err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}

	return &p, &rev, nil
}

// ErrPlanDeleteProtected prevents cascaded destruction of durable Mission
// evidence and scheduled work. A used plan is a historical execution record.
var ErrPlanDeleteProtected = fmt.Errorf("Este Flow possui execução, evidências, vínculo com Mission ou agendamento. Para preservar o histórico, não pode ser excluído.")

// DeleteWorkPlan removes only an unused draft. SQLite FK cascades otherwise
// delete Mission runs, execution snapshots and validation receipts: always
// inspect dependencies in the same write transaction before deleting.
func (s *Store) DeleteWorkPlan(id string) error {
	return s.DeleteWorkPlanExpected(id, 0)
}

func (s *Store) DeleteWorkPlanExpected(id string, expectedRevision int) error {
	tx, err := s.db.Begin()
	if err != nil { return err }
	defer tx.Rollback()
	var currentRevision int
	var legacyMissionID, status string
	if err := tx.QueryRow(`SELECT current_revision, COALESCE(mission_id,''), status FROM work_plans WHERE id=?`, id).
		Scan(&currentRevision, &legacyMissionID, &status); err != nil {
		return err
	}
	if expectedRevision > 0 && currentRevision != expectedRevision {
		return ErrPlanRevisionConflict
	}
	if legacyMissionID != "" || status == "EXECUTING" || status == "COMPLETED" {
		return ErrPlanDeleteProtected
	}
	for _, query := range []string{
		`SELECT EXISTS(SELECT 1 FROM mission_runs WHERE plan_id=?)`,
		`SELECT EXISTS(SELECT 1 FROM execution_snapshots WHERE plan_id=?)`,
		`SELECT EXISTS(SELECT 1 FROM mission_schedules WHERE plan_id=?)`,
	} {
		var used int
		if err := tx.QueryRow(query, id).Scan(&used); err != nil { return err }
		if used != 0 { return ErrPlanDeleteProtected }
	}
	res, err := tx.Exec(`DELETE FROM work_plans WHERE id=? AND current_revision=?`, id, currentRevision)
	if err != nil { return err }
	if count, err := res.RowsAffected(); err != nil { return err } else if count != 1 {
		return ErrPlanRevisionConflict
	}
	return tx.Commit()
}

func (s *Store) ListPlanRevisions(planID string) ([]PlanRevision, error) {
	rows, err := s.db.Query(`SELECT id, plan_id, revision, snapshot_json, change_summary, created_at
		FROM plan_revisions WHERE plan_id=? ORDER BY revision DESC`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []PlanRevision
	for rows.Next() {
		var r PlanRevision
		var createdStr string
		if err := rows.Scan(&r.ID, &r.PlanID, &r.Revision, &r.SnapshotJSON, &r.ChangeSummary, &createdStr); err != nil {
			return nil, err
		}
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		list = append(list, r)
	}
	return list, nil
}

func (s *Store) CreateExecutionSnapshot(planID, revisionID string, stateJSON string) (*ExecutionSnapshot, error) {
	now := time.Now().UTC()
	snap := ExecutionSnapshot{
		ID:         "snap_" + ids.NewRuntimeID(),
		PlanID:     planID,
		RevisionID: revisionID,
		StateJSON:  stateJSON,
		CreatedAt:  now,
	}
	_, err := s.db.Exec(`INSERT INTO execution_snapshots(id, plan_id, revision_id, state_json, created_at)
		VALUES(?, ?, ?, ?, ?)`,
		snap.ID, snap.PlanID, snap.RevisionID, snap.StateJSON, now.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

// GetPlanRevision returns the immutable snapshot for one exact plan revision.
func (s *Store) GetPlanRevision(planID string, revision int) (*PlanRevision, error) {
	var r PlanRevision
	var created string
	err := s.db.QueryRow(`SELECT id, plan_id, revision, snapshot_json, change_summary, created_at
		FROM plan_revisions WHERE plan_id=? AND revision=?`, planID, revision).Scan(
		&r.ID, &r.PlanID, &r.Revision, &r.SnapshotJSON, &r.ChangeSummary, &created)
	if err != nil {
		return nil, err
	}
	r.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &r, nil
}

// GetExecutionSnapshot returns an immutable mission execution snapshot by ID.
func (s *Store) GetExecutionSnapshot(id string) (*ExecutionSnapshot, error) {
	var snap ExecutionSnapshot
	var created string
	err := s.db.QueryRow(`SELECT id, plan_id, revision_id, state_json, created_at
		FROM execution_snapshots WHERE id=?`, id).Scan(
		&snap.ID, &snap.PlanID, &snap.RevisionID, &snap.StateJSON, &created)
	if err != nil {
		return nil, err
	}
	snap.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &snap, nil
}
