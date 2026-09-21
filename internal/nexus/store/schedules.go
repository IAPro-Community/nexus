package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

const (
	SchedulePending   = "PENDING"
	ScheduleRunning   = "RUNNING"
	ScheduleCompleted = "COMPLETED"
	ScheduleCanceled  = "CANCELED"
	ScheduleFailed    = "FAILED"
)

type MissionSchedule struct {
	ID            string     `json:"id"`
	PlanID        string     `json:"plan_id"`
	ProjectID     string     `json:"project_id"`
	Mode          string     `json:"mode"` // AT | AFTER_RUN | WHEN_RESOURCES
	ScheduledFor  *time.Time `json:"scheduled_for,omitempty"`
	AfterRunID    string     `json:"after_run_id,omitempty"`
	Status        string     `json:"status"`
	RunID         string     `json:"run_id,omitempty"`
	ContractJSON  string     `json:"contract_json"`
	LastError     string     `json:"last_error,omitempty"`
	AttemptCount  int        `json:"attempt_count"`
	LastAttemptAt *time.Time `json:"last_attempt_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (s *Store) CreateMissionSchedule(item MissionSchedule) (*MissionSchedule, error) {
	if item.ID == "" {
		item.ID = "schedule_" + ids.NewRuntimeID()
	}
	if item.Status == "" {
		item.Status = SchedulePending
	}
	if item.ContractJSON == "" {
		item.ContractJSON = "{}"
	}
	now := time.Now().UTC()
	item.CreatedAt, item.UpdatedAt = now, now
	_, err := s.db.Exec(`INSERT INTO mission_schedules(id,plan_id,project_id,mode,scheduled_for,after_run_id,status,run_id,contract_json,last_error,attempt_count,last_attempt_at,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)`, item.ID, item.PlanID, item.ProjectID, item.Mode, nullableTime(item.ScheduledFor), nullableString(item.AfterRunID), item.Status, nullableString(item.RunID), item.ContractJSON, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) GetMissionSchedule(id string) (*MissionSchedule, error) {
	row := s.db.QueryRow(`SELECT id,plan_id,project_id,mode,scheduled_for,after_run_id,status,run_id,contract_json,last_error,attempt_count,last_attempt_at,created_at,updated_at FROM mission_schedules WHERE id=?`, id)
	return scanMissionSchedule(row)
}

func (s *Store) ListMissionSchedules(projectID string) ([]MissionSchedule, error) {
	query := `SELECT id,plan_id,project_id,mode,scheduled_for,after_run_id,status,run_id,contract_json,last_error,attempt_count,last_attempt_at,created_at,updated_at FROM mission_schedules`
	args := []any{}
	if projectID != "" {
		query += ` WHERE project_id=?`
		args = append(args, projectID)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MissionSchedule
	for rows.Next() {
		item, err := scanMissionSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

// ListReadyMissionSchedules returns only schedules whose time/dependency trigger
// is satisfied. WHEN_RESOURCES remains pending until the Nexus scheduler also
// confirms an eligible provider resource.
func (s *Store) ListReadyMissionSchedules(now time.Time) ([]MissionSchedule, error) {
	rows, err := s.db.Query(`SELECT s.id,s.plan_id,s.project_id,s.mode,s.scheduled_for,s.after_run_id,s.status,s.run_id,s.contract_json,s.created_at,s.updated_at
		FROM mission_schedules s
		LEFT JOIN mission_runs r ON r.id=s.after_run_id
		WHERE s.status=? AND (
			(s.mode='AT' AND s.scheduled_for IS NOT NULL AND s.scheduled_for<=?) OR
			(s.mode='AFTER_RUN' AND r.state='COMPLETED_VERIFIED') OR
			s.mode='WHEN_RESOURCES'
		) ORDER BY s.created_at ASC`, SchedulePending, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MissionSchedule
	for rows.Next() {
		item, err := scanMissionSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (s *Store) UpdateMissionScheduleStatus(id, status string) error {
	res, err := s.db.Exec(`UPDATE mission_schedules SET status=?,updated_at=? WHERE id=?`, status, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("mission schedule not found")
	}
	return nil
}

// ClaimMissionSchedule atomically transitions a pending schedule to running.
// The conditional update is the concurrency boundary: only one scheduler
// worker can observe a successful claim for a given schedule.
func (s *Store) ClaimMissionSchedule(id string) (bool, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.db.Exec(`UPDATE mission_schedules SET status=?,attempt_count=attempt_count+1,last_attempt_at=?,last_error='',updated_at=? WHERE id=? AND status=?`, ScheduleRunning, now, now, id, SchedulePending)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// FailMissionSchedule records the durable reason an admitted schedule could not
// start or its bound Mission terminated unsuccessfully.
func (s *Store) FailMissionSchedule(id, reason string) error {
	res, err := s.db.Exec(`UPDATE mission_schedules SET status=?,last_error=?,updated_at=? WHERE id=?`, ScheduleFailed, reason, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("mission schedule not found")
	}
	return nil
}

func (s *Store) BindMissionScheduleRun(id, runID string) error {
	res, err := s.db.Exec(`UPDATE mission_schedules SET run_id=?,status=?,last_error='',updated_at=? WHERE id=? AND status IN (?,?) AND run_id IS NULL`, runID, ScheduleRunning, time.Now().UTC().Format(time.RFC3339Nano), id, SchedulePending, ScheduleRunning)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("mission schedule is no longer pending")
	}
	return nil
}

type missionScheduleScanner interface{ Scan(...any) error }

func scanMissionSchedule(row missionScheduleScanner) (*MissionSchedule, error) {
	var item MissionSchedule
	var scheduled, after, runID, lastAttempt sql.NullString
	var created, updated string
	if err := row.Scan(&item.ID, &item.PlanID, &item.ProjectID, &item.Mode, &scheduled, &after, &item.Status, &runID, &item.ContractJSON, &item.LastError, &item.AttemptCount, &lastAttempt, &created, &updated); err != nil {
		return nil, err
	}
	if scheduled.Valid {
		if t, err := time.Parse(time.RFC3339Nano, scheduled.String); err == nil {
			item.ScheduledFor = &t
		}
	}
	if after.Valid {
		item.AfterRunID = after.String
	}
	if runID.Valid {
		item.RunID = runID.String
	}
	if lastAttempt.Valid {
		if t, err := time.Parse(time.RFC3339Nano, lastAttempt.String); err == nil {
			item.LastAttemptAt = &t
		}
	}
	item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &item, nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
