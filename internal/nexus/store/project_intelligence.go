package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/nexus/contextsnapshot"
)

const (
	ScanQueued    = "QUEUED"
	ScanRunning   = "RUNNING"
	ScanSucceeded = "SUCCEEDED"
	ScanFailed    = "FAILED"
	ScanCanceled  = "CANCELED"
)

type ProjectIntelligenceScan struct {
	ID             string     `json:"id"`
	ProjectID      string     `json:"project_id"`
	IdentityDigest string     `json:"identity_digest"`
	State          string     `json:"state"`
	ScannerVersion string     `json:"scanner_version"`
	RequestedAt    time.Time  `json:"requested_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
	LeaseOwner     string     `json:"lease_owner,omitempty"`
	LeaseExpiresAt *time.Time `json:"lease_expires_at,omitempty"`
	Error          string     `json:"error,omitempty"`
}

func parseOptionalTime(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return nil
	}
	return &parsed
}

func scanIntelligenceScan(scanner interface{ Scan(...any) error }) (ProjectIntelligenceScan, error) {
	var result ProjectIntelligenceScan
	var requested string
	var started, finished, lease sql.NullString
	err := scanner.Scan(&result.ID, &result.ProjectID, &result.IdentityDigest, &result.State, &result.ScannerVersion, &requested, &started, &finished, &result.LeaseOwner, &lease, &result.Error)
	if err != nil {
		return result, err
	}
	result.RequestedAt, _ = time.Parse(time.RFC3339Nano, requested)
	result.StartedAt, result.FinishedAt, result.LeaseExpiresAt = parseOptionalTime(started), parseOptionalTime(finished), parseOptionalTime(lease)
	return result, nil
}

const intelligenceScanColumns = `id,project_id,identity_digest,state,scanner_version,requested_at,started_at,finished_at,lease_owner,lease_expires_at,error`

func (s *Store) CreateProjectIntelligenceScan(scan ProjectIntelligenceScan) (ProjectIntelligenceScan, error) {
	if scan.ID == "" {
		scan.ID = "pis_" + ids.NewRuntimeID()
	}
	if scan.State == "" {
		scan.State = ScanQueued
	}
	if scan.ScannerVersion == "" {
		scan.ScannerVersion = contextsnapshot.ScannerVersion
	}
	if scan.RequestedAt.IsZero() {
		scan.RequestedAt = time.Now().UTC()
	}
	_, err := s.db.Exec(`INSERT INTO project_intelligence_scans(id,project_id,identity_digest,state,scanner_version,requested_at,error) VALUES(?,?,?,?,?,?,?)`, scan.ID, scan.ProjectID, scan.IdentityDigest, scan.State, scan.ScannerVersion, scan.RequestedAt.Format(time.RFC3339Nano), scan.Error)
	if err != nil {
		return ProjectIntelligenceScan{}, fmt.Errorf("create project intelligence scan: %w", err)
	}
	return s.GetProjectIntelligenceScan(scan.ID)
}

func (s *Store) GetProjectIntelligenceScan(id string) (ProjectIntelligenceScan, error) {
	return scanIntelligenceScan(s.db.QueryRow(`SELECT `+intelligenceScanColumns+` FROM project_intelligence_scans WHERE id=?`, id))
}

func (s *Store) ListProjectIntelligenceScans(projectID string) ([]ProjectIntelligenceScan, error) {
	rows, err := s.db.Query(`SELECT `+intelligenceScanColumns+` FROM project_intelligence_scans WHERE project_id=? ORDER BY requested_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ProjectIntelligenceScan{}
	for rows.Next() {
		scan, scanErr := scanIntelligenceScan(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, scan)
	}
	return result, rows.Err()
}

func (s *Store) UpdateProjectIntelligenceScan(scan ProjectIntelligenceScan) error {
	if scan.ID == "" {
		return errors.New("scan id is required")
	}
	_, err := s.db.Exec(`UPDATE project_intelligence_scans SET state=?,started_at=?,finished_at=?,lease_owner=?,lease_expires_at=?,error=? WHERE id=?`, scan.State, nullableTime(scan.StartedAt), nullableTime(scan.FinishedAt), scan.LeaseOwner, nullableTime(scan.LeaseExpiresAt), scan.Error, scan.ID)
	return err
}

func (s *Store) LatestProjectContextSnapshot(projectID string) (*contextsnapshot.ProjectContextSnapshot, error) {
	var raw string
	err := s.db.QueryRow(`SELECT snapshot_json FROM project_context_snapshots WHERE project_id=? ORDER BY observed_at DESC LIMIT 1`, projectID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var snapshot contextsnapshot.ProjectContextSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return nil, fmt.Errorf("decode project context snapshot: %w", err)
	}
	return &snapshot, nil
}

// ProjectContextSnapshotByIdentity returns the immutable snapshot for the
// exact source identity. A snapshot for another identity is never treated as
// current by callers.
func (s *Store) ProjectContextSnapshotByIdentity(projectID, identityDigest string) (*contextsnapshot.ProjectContextSnapshot, error) {
	var raw string
	err := s.db.QueryRow(`SELECT snapshot_json FROM project_context_snapshots WHERE project_id=? AND identity_digest=? ORDER BY observed_at DESC LIMIT 1`, projectID, identityDigest).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var snapshot contextsnapshot.ProjectContextSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return nil, fmt.Errorf("decode project context snapshot: %w", err)
	}
	return &snapshot, nil
}

func (s *Store) SaveProjectContextSnapshot(snapshot contextsnapshot.ProjectContextSnapshot, scanID string) (contextsnapshot.ProjectContextSnapshot, error) {
	if snapshot.ProjectID == "" {
		return contextsnapshot.ProjectContextSnapshot{}, errors.New("project id is required")
	}
	if snapshot.ScannerVersion == "" {
		snapshot.ScannerVersion = contextsnapshot.ScannerVersion
	}
	if snapshot.ObservedAt.IsZero() {
		snapshot.ObservedAt = time.Now().UTC()
	}
	if snapshot.ID == "" {
		snapshot.ID = "pcs_" + ids.NewRuntimeID()
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return contextsnapshot.ProjectContextSnapshot{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return contextsnapshot.ProjectContextSnapshot{}, err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`INSERT OR IGNORE INTO project_context_snapshots(id,project_id,identity_digest,scanner_version,completeness,snapshot_json,observed_at,scan_id) VALUES(?,?,?,?,?,?,?,?)`, snapshot.ID, snapshot.ProjectID, snapshot.Identity.IdentityDigest, snapshot.ScannerVersion, snapshot.Completeness, raw, snapshot.ObservedAt.Format(time.RFC3339Nano), nullableString(scanID))
	if err != nil {
		return contextsnapshot.ProjectContextSnapshot{}, fmt.Errorf("save project context snapshot: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return contextsnapshot.ProjectContextSnapshot{}, err
	}
	var existingRaw string
	if err := tx.QueryRow(`SELECT id,snapshot_json FROM project_context_snapshots WHERE project_id=? AND identity_digest=? AND scanner_version=?`, snapshot.ProjectID, snapshot.Identity.IdentityDigest, snapshot.ScannerVersion).Scan(&snapshot.ID, &existingRaw); err != nil {
		return contextsnapshot.ProjectContextSnapshot{}, err
	}
	if existingRaw != "" {
		if err := json.Unmarshal([]byte(existingRaw), &snapshot); err != nil {
			return contextsnapshot.ProjectContextSnapshot{}, err
		}
	}
	if inserted == 0 {
		if err := tx.Commit(); err != nil {
			return contextsnapshot.ProjectContextSnapshot{}, err
		}
		return snapshot, nil
	}
	for _, fact := range snapshot.Facts {
		factID := "pcf_" + ids.NewRuntimeID()
		value, marshalErr := json.Marshal(fact.Value)
		if marshalErr != nil {
			return contextsnapshot.ProjectContextSnapshot{}, marshalErr
		}
		_, err = tx.Exec(`INSERT OR IGNORE INTO project_context_facts(id,snapshot_id,category,fact_key,value_type,value_json,basis,confidence,observed_at) VALUES(?,?,?,?,?,?,?,?,?)`, factID, snapshot.ID, fact.Category, fact.Key, fact.ValueType, value, fact.Basis, fact.Confidence, fact.ObservedAt.Format(time.RFC3339Nano))
		if err != nil {
			return contextsnapshot.ProjectContextSnapshot{}, err
		}
		var persistedFactID string
		if err := tx.QueryRow(`SELECT id FROM project_context_facts WHERE snapshot_id=? AND category=? AND fact_key=?`, snapshot.ID, fact.Category, fact.Key).Scan(&persistedFactID); err != nil {
			return contextsnapshot.ProjectContextSnapshot{}, err
		}
		for _, provenance := range fact.Provenance {
			_, err = tx.Exec(`INSERT INTO project_context_provenance(id,fact_id,source_path,locator,extractor,content_digest,observed_at) VALUES(?,?,?,?,?,?,?)`, "pcp_"+ids.NewRuntimeID(), persistedFactID, provenance.SourcePath, provenance.Locator, provenance.Extractor, provenance.Digest, provenance.ObservedAt.Format(time.RFC3339Nano))
			if err != nil {
				return contextsnapshot.ProjectContextSnapshot{}, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return contextsnapshot.ProjectContextSnapshot{}, err
	}
	return snapshot, nil
}
