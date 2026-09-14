package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

// CreateMission inserts a new mission.
func (s *Store) CreateMission(m *Mission) error {
	now := time.Now().UTC()
	if m.ID == "" {
		m.ID = "mis_" + ids.NewRuntimeID()
	}
	m.CreatedAt = now
	m.UpdatedAt = now
	if m.Status == "" {
		m.Status = MissionDraft
	}
	_, err := s.db.Exec(`INSERT INTO missions(id, project_id, name, description, status, goal, scope, risk_level, config, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		m.ID, m.ProjectID, m.Name, m.Description, m.Status, m.Goal, m.Scope, m.RiskLevel, m.Config, m.CreatedAt.Format(time.RFC3339Nano), m.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

// GetMission returns a mission by ID.
func (s *Store) GetMission(id string) (*Mission, error) {
	m, err := scanMission(s.db.QueryRow(`SELECT id, project_id, name, description, status, goal, scope, risk_level, config, created_at, updated_at, started_at, completed_at
		FROM missions WHERE id=?`, id))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("mission not found: %s", id)
	}
	return m, err
}

// ListMissions returns all missions for a project.
func (s *Store) ListMissions(projectID string) ([]Mission, error) {
	rows, err := s.db.Query(`SELECT id, project_id, name, description, status, goal, scope, risk_level, config, created_at, updated_at, started_at, completed_at
		FROM missions WHERE project_id=? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var missions []Mission
	for rows.Next() {
		m, err := scanMission(rows)
		if err != nil {
			return nil, err
		}
		missions = append(missions, *m)
	}
	return missions, rows.Err()
}

// UpdateMission updates mutable mission fields.
func (s *Store) UpdateMission(m *Mission) error {
	m.UpdatedAt = time.Now().UTC()
	_, err := s.db.Exec(`UPDATE missions SET name=?, description=?, status=?, goal=?, scope=?, risk_level=?, config=?, updated_at=?, started_at=?, completed_at=? WHERE id=?`,
		m.Name, m.Description, m.Status, m.Goal, m.Scope, m.RiskLevel, m.Config, m.UpdatedAt.Format(time.RFC3339Nano), formatNullableTime(m.StartedAt), formatNullableTime(m.CompletedAt), m.ID)
	return err
}

// DeleteMission removes a mission and cascades tasks/assignments.
func (s *Store) DeleteMission(id string) error {
	_, err := s.db.Exec(`DELETE FROM missions WHERE id=?`, id)
	return err
}

// CreateTask inserts a new task within a mission.
func (s *Store) CreateTask(t *MissionTask) error {
	now := time.Now().UTC()
	if t.ID == "" {
		t.ID = "tsk_" + ids.NewRuntimeID()
	}
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.Status == "" {
		t.Status = TaskPending
	}
	if t.Dependencies == "" {
		t.Dependencies = "[]"
	}
	_, err := s.db.Exec(`INSERT INTO mission_tasks(id, mission_id, name, description, status, kind, priority, dependencies, config, result, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.MissionID, t.Name, t.Description, t.Status, t.Kind, t.Priority, t.Dependencies, t.Config, t.Result, t.CreatedAt.Format(time.RFC3339Nano), t.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

// GetTask returns a task by ID.
func (s *Store) GetTask(id string) (*MissionTask, error) {
	t, err := scanMissionTask(s.db.QueryRow(`SELECT id, mission_id, name, description, status, kind, priority, dependencies, config, result, created_at, updated_at, started_at, completed_at
		FROM mission_tasks WHERE id=?`, id))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	return t, err
}

// ListTasks returns all tasks for a mission.
func (s *Store) ListTasks(missionID string) ([]MissionTask, error) {
	rows, err := s.db.Query(`SELECT id, mission_id, name, description, status, kind, priority, dependencies, config, result, created_at, updated_at, started_at, completed_at
		FROM mission_tasks WHERE mission_id=? ORDER BY priority, created_at`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []MissionTask
	for rows.Next() {
		t, err := scanMissionTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *t)
	}
	return tasks, rows.Err()
}

// UpdateTask updates mutable task fields.
func (s *Store) UpdateTask(t *MissionTask) error {
	t.UpdatedAt = time.Now().UTC()
	_, err := s.db.Exec(`UPDATE mission_tasks SET name=?, description=?, status=?, kind=?, priority=?, dependencies=?, config=?, result=?, updated_at=?, started_at=?, completed_at=? WHERE id=?`,
		t.Name, t.Description, t.Status, t.Kind, t.Priority, t.Dependencies, t.Config, t.Result, t.UpdatedAt.Format(time.RFC3339Nano), formatNullableTime(t.StartedAt), formatNullableTime(t.CompletedAt), t.ID)
	return err
}

// CreateAssignment inserts a new agent assignment to a task.
func (s *Store) CreateAssignment(a *MissionAssignment) error {
	if a.ID == "" {
		a.ID = "asn_" + ids.NewRuntimeID()
	}
	a.AssignedAt = time.Now().UTC()
	if a.Status == "" {
		a.Status = "ASSIGNED"
	}
	_, err := s.db.Exec(`INSERT INTO mission_assignments(id, mission_id, task_id, agent_id, status, assigned_at)
		VALUES(?,?,?,?,?,?)`,
		a.ID, a.MissionID, a.TaskID, a.AgentID, a.Status, a.AssignedAt.Format(time.RFC3339Nano))
	return err
}

// ListAssignments returns all assignments for a mission.
func (s *Store) ListAssignments(missionID string) ([]MissionAssignment, error) {
	rows, err := s.db.Query(`SELECT id, mission_id, task_id, agent_id, status, assigned_at, completed_at
		FROM mission_assignments WHERE mission_id=? ORDER BY assigned_at`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var assigns []MissionAssignment
	for rows.Next() {
		a, err := scanMissionAssignment(rows)
		if err != nil {
			return nil, err
		}
		assigns = append(assigns, *a)
	}
	return assigns, rows.Err()
}

// UpdateAssignment updates an assignment's status.
func (s *Store) UpdateAssignment(a *MissionAssignment) error {
	_, err := s.db.Exec(`UPDATE mission_assignments SET status=?, completed_at=? WHERE id=?`,
		a.Status, formatNullableTime(a.CompletedAt), a.ID)
	return err
}

func scanMission(scanner rowScanner) (*Mission, error) {
	m := &Mission{}
	var created, updated string
	var started, completed sql.NullString
	err := scanner.Scan(&m.ID, &m.ProjectID, &m.Name, &m.Description, &m.Status, &m.Goal, &m.Scope, &m.RiskLevel, &m.Config, &created, &updated, &started, &completed)
	if err != nil {
		return nil, err
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	m.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	m.StartedAt = parseNullableTime(started)
	m.CompletedAt = parseNullableTime(completed)
	return m, nil
}

func scanMissionTask(scanner rowScanner) (*MissionTask, error) {
	t := &MissionTask{}
	var created, updated string
	var started, completed sql.NullString
	err := scanner.Scan(&t.ID, &t.MissionID, &t.Name, &t.Description, &t.Status, &t.Kind, &t.Priority, &t.Dependencies, &t.Config, &t.Result, &created, &updated, &started, &completed)
	if err != nil {
		return nil, err
	}
	t.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	t.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	t.StartedAt = parseNullableTime(started)
	t.CompletedAt = parseNullableTime(completed)
	return t, nil
}

func scanMissionAssignment(scanner rowScanner) (*MissionAssignment, error) {
	a := &MissionAssignment{}
	var assigned string
	var completed sql.NullString
	if err := scanner.Scan(&a.ID, &a.MissionID, &a.TaskID, &a.AgentID, &a.Status, &assigned, &completed); err != nil {
		return nil, err
	}
	a.AssignedAt, _ = time.Parse(time.RFC3339Nano, assigned)
	a.CompletedAt = parseNullableTime(completed)
	return a, nil
}

func parseNullableTime(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return nil
	}
	return &t
}

func formatNullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.Format(time.RFC3339Nano)
}

// MissionStats returns aggregate stats for a mission.
func (s *Store) MissionStats(missionID string) (total, pending, active, completed, failed int, err error) {
	err = s.db.QueryRow(`SELECT COUNT(*) FROM mission_tasks WHERE mission_id=?`, missionID).Scan(&total)
	if err != nil {
		return
	}
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM mission_tasks WHERE mission_id=? AND status=?`, missionID, TaskPending).Scan(&pending)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM mission_tasks WHERE mission_id=? AND status=?`, missionID, TaskActive).Scan(&active)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM mission_tasks WHERE mission_id=? AND status=?`, missionID, TaskCompleted).Scan(&completed)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM mission_tasks WHERE mission_id=? AND status=?`, missionID, TaskFailed).Scan(&failed)
	return
}
