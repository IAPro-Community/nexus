package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

type MissionInterventionResolutionRecord struct {
	ID             string
	RunID          string
	InterventionID string
	Version        int
	OptionID       string
	IdempotencyKey string
	ResolvedAt     time.Time
	ResolvedBy     string
	ResumeStatus   string
}

type MissionRunEventRecord struct {
	ID             string
	RunID          string
	EventType      string
	IdempotencyKey string
	PayloadJSON    string
	CreatedAt      time.Time
}

type MissionRunRecord struct {
	ID             string
	PlanID         string
	ProjectID      string
	State          string
	PayloadJSON    string
	LeaseOwner     string
	LeaseToken     string
	LeaseExpiresAt *time.Time
	HeartbeatAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (s *Store) UpsertMissionRun(rec MissionRunRecord) error {
	now := time.Now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	rec.UpdatedAt = now

	// Lease metadata is authoritative and mutated only by Acquire/Renew/Release.
	// A worker that holds a fencing token may update payload/state only while the
	// same token is still current. User/control-plane writes without a token are
	// accepted only while no live worker owns the run.
	if rec.LeaseToken != "" {
		res, err := s.db.Exec(`UPDATE mission_runs SET state=?,payload_json=?,updated_at=? WHERE id=? AND lease_token=?`,
			rec.State, rec.PayloadJSON, rec.UpdatedAt.Format(time.RFC3339Nano), rec.ID, rec.LeaseToken)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("mission run lease fencing mismatch")
		}
		return nil
	}

	// First persist is an INSERT. Existing rows are updated only if their lease
	// is empty/expired, preventing pause/cancel/resume from racing a live worker.
	res, err := s.db.Exec(`UPDATE mission_runs SET state=?,payload_json=?,updated_at=?
		WHERE id=? AND (lease_token='' OR lease_expires_at IS NULL OR lease_expires_at<=?)`,
		rec.State, rec.PayloadJSON, rec.UpdatedAt.Format(time.RFC3339Nano), rec.ID, now.Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	var exists int
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM mission_runs WHERE id=?)`, rec.ID).Scan(&exists); err != nil {
		return err
	}
	if exists == 1 {
		return fmt.Errorf("mission run lease held")
	}
	_, err = s.db.Exec(`INSERT INTO mission_runs(id,plan_id,project_id,state,payload_json,lease_owner,lease_token,lease_expires_at,heartbeat_at,created_at,updated_at)
		VALUES(?,?,?,?,?,'','',NULL,NULL,?,?)`,
		rec.ID, rec.PlanID, rec.ProjectID, rec.State, rec.PayloadJSON, rec.CreatedAt.Format(time.RFC3339Nano), rec.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

// CommitMissionInterventionResolution atomically stores the updated run and
// its durable resolution/resume intent. The event rows are an outbox for
// projections; they are committed before any in-memory worker is scheduled.
func (s *Store) CommitMissionInterventionResolution(run MissionRunRecord, resolution MissionInterventionResolutionRecord) error {
	if run.ID == "" || resolution.IdempotencyKey == "" || resolution.RunID != run.ID {
		return fmt.Errorf("mission intervention commit requires matching run and resolution")
	}
	now := time.Now().UTC()
	if run.UpdatedAt.IsZero() {
		run.UpdatedAt = now
	}
	if resolution.ResolvedAt.IsZero() {
		resolution.ResolvedAt = now
	}
	if resolution.ID == "" {
		resolution.ID = "resolution_" + ids.NewRuntimeID()
	}
	if resolution.ResumeStatus == "" {
		resolution.ResumeStatus = "PENDING"
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin intervention commit: %w", err)
	}
	defer tx.Rollback()
	var result sql.Result
	if run.LeaseToken != "" {
		result, err = tx.Exec(`UPDATE mission_runs SET state=?,payload_json=?,updated_at=? WHERE id=? AND lease_token=?`, run.State, run.PayloadJSON, run.UpdatedAt.Format(time.RFC3339Nano), run.ID, run.LeaseToken)
	} else {
		result, err = tx.Exec(`UPDATE mission_runs SET state=?,payload_json=?,updated_at=? WHERE id=? AND (lease_token='' OR lease_expires_at IS NULL OR lease_expires_at<=?)`, run.State, run.PayloadJSON, run.UpdatedAt.Format(time.RFC3339Nano), run.ID, now.Format(time.RFC3339Nano))
	}
	if err != nil {
		return fmt.Errorf("persist intervention run: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return fmt.Errorf("mission run lease fencing mismatch")
	}
	_, err = tx.Exec(`INSERT OR IGNORE INTO mission_intervention_resolutions(id,run_id,intervention_id,version,option_id,idempotency_key,resolved_at,resolved_by,resume_status,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		resolution.ID, resolution.RunID, resolution.InterventionID, resolution.Version, resolution.OptionID, resolution.IdempotencyKey, resolution.ResolvedAt.Format(time.RFC3339Nano), resolution.ResolvedBy, resolution.ResumeStatus, now.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("persist intervention resolution: %w", err)
	}
	payload, _ := json.Marshal(map[string]any{"intervention_id": resolution.InterventionID, "version": resolution.Version, "option_id": resolution.OptionID, "resolved_by": resolution.ResolvedBy})
	if err := insertMissionRunEventTx(tx, resolution.RunID, "human_intervention.resolved", resolution.IdempotencyKey, string(payload), resolution.ResolvedAt); err != nil {
		return err
	}
	if err := insertMissionRunEventTx(tx, resolution.RunID, "mission.resume_requested", resolution.IdempotencyKey, string(payload), resolution.ResolvedAt); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit intervention resolution: %w", err)
	}
	return nil
}

func insertMissionRunEventTx(tx *sql.Tx, runID, eventType, idempotencyKey, payload string, createdAt time.Time) error {
	if _, err := tx.Exec(`INSERT OR IGNORE INTO mission_run_events(id,run_id,event_type,idempotency_key,payload_json,created_at) VALUES(?,?,?,?,?,?)`,
		"mission_event_"+ids.NewRuntimeID(), runID, eventType, idempotencyKey, payload, createdAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("persist mission event %s: %w", eventType, err)
	}
	return nil
}

func (s *Store) ListMissionRunEvents(runID string) ([]MissionRunEventRecord, error) {
	rows, err := s.db.Query(`SELECT id,run_id,event_type,COALESCE(idempotency_key,''),payload_json,created_at FROM mission_run_events WHERE run_id=? ORDER BY created_at ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MissionRunEventRecord
	for rows.Next() {
		var record MissionRunEventRecord
		var created string
		if err := rows.Scan(&record.ID, &record.RunID, &record.EventType, &record.IdempotencyKey, &record.PayloadJSON, &created); err != nil {
			return nil, err
		}
		record.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, record)
	}
	return out, rows.Err()
}

func (s *Store) GetMissionRun(id string) (*MissionRunRecord, error) {
	row := s.db.QueryRow(`SELECT id,plan_id,project_id,state,payload_json,lease_owner,lease_token,lease_expires_at,heartbeat_at,created_at,updated_at FROM mission_runs WHERE id=?`, id)
	return scanMissionRun(row)
}

func (s *Store) ListMissionRuns() ([]MissionRunRecord, error) {
	rows, err := s.db.Query(`SELECT id,plan_id,project_id,state,payload_json,lease_owner,lease_token,lease_expires_at,heartbeat_at,created_at,updated_at FROM mission_runs ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MissionRunRecord
	for rows.Next() {
		rec, err := scanMissionRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *rec)
	}
	return out, rows.Err()
}

// ListMissionRunsForProject returns only the newest bounded Mission runs for
// one project. Context consumers must not scan the global run ledger and then
// filter in memory.
func (s *Store) ListMissionRunsForProject(projectID string, limit int) ([]MissionRunRecord, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("project id is required")
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	rows, err := s.db.Query(`SELECT id,plan_id,project_id,state,payload_json,lease_owner,lease_token,lease_expires_at,heartbeat_at,created_at,updated_at
		FROM mission_runs WHERE project_id=? ORDER BY updated_at DESC LIMIT ?`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]MissionRunRecord, 0, limit)
	for rows.Next() {
		rec, err := scanMissionRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *rec)
	}
	return out, rows.Err()
}

type missionRunScanner interface{ Scan(...any) error }

func scanMissionRun(row missionRunScanner) (*MissionRunRecord, error) {
	var rec MissionRunRecord
	var leaseExp, heartbeat sql.NullString
	var created, updated string
	if err := row.Scan(&rec.ID, &rec.PlanID, &rec.ProjectID, &rec.State, &rec.PayloadJSON, &rec.LeaseOwner, &rec.LeaseToken, &leaseExp, &heartbeat, &created, &updated); err != nil {
		return nil, err
	}
	rec.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	rec.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	if leaseExp.Valid {
		t, err := time.Parse(time.RFC3339Nano, leaseExp.String)
		if err == nil {
			rec.LeaseExpiresAt = &t
		}
	}
	if heartbeat.Valid {
		t, err := time.Parse(time.RFC3339Nano, heartbeat.String)
		if err == nil {
			rec.HeartbeatAt = &t
		}
	}
	return &rec, nil
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// AcquireMissionLease is an atomic fencing operation. A live lease owned by a
// different worker cannot be stolen; expired leases are reclaimable after restart.
func (s *Store) AcquireMissionLease(id, owner string, ttl time.Duration) (*MissionRunRecord, error) {
	now := time.Now().UTC()
	expires := now.Add(ttl)
	token := "lease_" + ids.NewRuntimeID()
	res, err := s.db.Exec(`UPDATE mission_runs SET lease_owner=?,lease_token=?,lease_expires_at=?,heartbeat_at=?,updated_at=?
		WHERE id=? AND (lease_owner='' OR lease_owner=? OR lease_expires_at IS NULL OR lease_expires_at<=?)`,
		owner, token, expires.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), id, owner, now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var exists int
		if queryErr := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM mission_runs WHERE id=?)`, id).Scan(&exists); queryErr != nil {
			return nil, queryErr
		}
		if exists == 0 {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("mission run lease held")
	}
	return s.GetMissionRun(id)
}

func (s *Store) RenewMissionLease(id, owner, token string, ttl time.Duration) error {
	now := time.Now().UTC()
	expires := now.Add(ttl)
	res, err := s.db.Exec(`UPDATE mission_runs SET lease_expires_at=?,heartbeat_at=?,updated_at=? WHERE id=? AND lease_owner=? AND lease_token=?`, expires.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), id, owner, token)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("mission run lease fencing mismatch")
	}
	return nil
}
func (s *Store) ReleaseMissionLease(id, owner, token string) error {
	res, err := s.db.Exec(`UPDATE mission_runs SET lease_owner='',lease_token='',lease_expires_at=NULL WHERE id=? AND lease_owner=? AND lease_token=?`, id, owner, token)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("mission run lease fencing mismatch")
	}
	return nil
}

type PromptVersion struct {
	ID           string    `json:"id"`
	PlanID       string    `json:"plan_id"`
	PackageID    string    `json:"package_id"`
	PlanRevision int       `json:"plan_revision"`
	ContentHash  string    `json:"content_hash"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

func (s *Store) CreatePromptVersion(v PromptVersion) (*PromptVersion, error) {
	if v.ID == "" {
		v.ID = "prompt_" + ids.NewRuntimeID()
	}
	v.CreatedAt = time.Now().UTC()
	_, err := s.db.Exec(`INSERT INTO prompt_versions(id,plan_id,package_id,plan_revision,content_hash,content,created_at) VALUES(?,?,?,?,?,?,?)`, v.ID, v.PlanID, v.PackageID, v.PlanRevision, v.ContentHash, v.Content, v.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Store) LatestPromptVersion(planID, packageID string) (*PromptVersion, error) {
	var v PromptVersion
	var created string
	err := s.db.QueryRow(`SELECT id,plan_id,package_id,plan_revision,content_hash,content,created_at FROM prompt_versions WHERE plan_id=? AND package_id=? ORDER BY created_at DESC LIMIT 1`, planID, packageID).Scan(&v.ID, &v.PlanID, &v.PackageID, &v.PlanRevision, &v.ContentHash, &v.Content, &created)
	if err != nil {
		return nil, err
	}
	v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return &v, nil
}
