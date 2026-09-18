package store

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/core/config"
)

var (
	// ErrNotFound is returned when a requested row does not exist.
	ErrNotFound = errors.New("not found")
	// ErrDuplicateSlug is returned when a project slug collides.
	ErrDuplicateSlug = errors.New("duplicate project slug")
)

const projectColumns = `id,name,slug,canonical_path,repo_remote,repo_url,default_branch,
	maestro_mode,resource_policy,default_isolation,settings,created_at,updated_at,last_opened_at,
	display_path,identity_kind,identity_key`

// CanonicalPath resolves and validates a project path: absolute, cleaned,
// symlinks resolved, must exist and be a directory.
func CanonicalPath(p string) (string, error) {
	return config.CanonicalExistingWorkspaceDir(p)
}

// Slugify derives a URL/path-safe slug from a name.
func Slugify(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var sb strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			sb.WriteRune(r)
		case r == ' ' || r == '_' || r == '-' || r == '.':
			sb.WriteRune('-')
		}
	}
	slug := strings.Trim(sb.String(), "-")
	if slug == "" {
		slug = "project"
	}
	return slug
}

// CreateProject inserts a project. If slug is empty it is derived from name;
// collisions are disambiguated with a short suffix.
func (s *Store) CreateProject(p Project) (Project, error) {
	if p.ID == "" {
		p.ID = "prj_" + ids.NewRuntimeID()
	}
	if p.CanonicalPath == "" {
		return Project{}, errors.New("canonical_path is required")
	}
	if p.DisplayPath == "" {
		p.DisplayPath = p.CanonicalPath
	}
	if p.IdentityKind == "" {
		if ref, err := config.ResolvePathRef(p.CanonicalPath); err == nil {
			p.IdentityKind = ref.Identity.Kind
			p.IdentityKey = ref.Identity.StableKey
			p.PathRef = ref
		}
	}
	if strings.TrimSpace(p.Name) == "" {
		p.Name = filepath.Base(p.CanonicalPath)
	}
	if p.Slug == "" {
		p.Slug = Slugify(p.Name)
	}
	if p.MaestroMode == "" {
		p.MaestroMode = MaestroOff
	}
	if p.DefaultIsolation == "" {
		p.DefaultIsolation = "project"
	}
	if p.ResourcePolicy == "" {
		p.ResourcePolicy = "{}"
	}
	if p.Settings == "" {
		p.Settings = "{}"
	}
	if p.DefaultBranch == "" {
		p.DefaultBranch = "main"
	}
	now := time.Now().UTC()
	p.CreatedAt, p.UpdatedAt = now, now

	for attempt := 0; attempt < 10; attempt++ {
		slug := p.Slug
		if attempt > 0 {
			slug = fmt.Sprintf("%s-%d", p.Slug, attempt)
		}
		_, err := s.db.Exec(`INSERT INTO projects
			(id,name,slug,canonical_path,repo_remote,repo_url,default_branch,maestro_mode,
			 resource_policy,default_isolation,settings,created_at,updated_at,last_opened_at,
			 display_path,identity_kind,identity_key)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,NULL,?,?,?)`,
			p.ID, p.Name, slug, p.CanonicalPath, p.RepoRemote, p.RepoURL, p.DefaultBranch,
			p.MaestroMode, p.ResourcePolicy, p.DefaultIsolation, p.Settings,
			p.CreatedAt.Format(time.RFC3339Nano), p.UpdatedAt.Format(time.RFC3339Nano),
			p.DisplayPath, p.IdentityKind, p.IdentityKey)
		if err == nil {
			p.Slug = slug
			return p, nil
		}
		if strings.Contains(err.Error(), "UNIQUE") && strings.Contains(err.Error(), "slug") {
			continue
		}
		return Project{}, fmt.Errorf("create project: %w", err)
	}
	return Project{}, ErrDuplicateSlug
}

// GetProject loads a project by ID.
func (s *Store) GetProject(id string) (Project, error) {
	return s.scanProject(s.db.QueryRow(`SELECT `+projectColumns+` FROM projects WHERE id=?`, id))
}

// GetProjectByPath loads a project by canonical path.
func (s *Store) GetProjectByPath(path string) (Project, error) {
	return s.scanProject(s.db.QueryRow(`SELECT `+projectColumns+` FROM projects WHERE canonical_path=?`, path))
}

// ListProjects returns all projects ordered by most recently opened first.
func (s *Store) ListProjects() ([]Project, error) {
	rows, err := s.db.Query(`SELECT ` + projectColumns + ` FROM projects ORDER BY last_opened_at IS NULL, last_opened_at DESC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out = []Project{}
	for rows.Next() {
		p, err := scanProjectRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpdateProject patches mutable project fields by ID.
func (s *Store) UpdateProject(p Project) error {
	if p.ID == "" {
		return errors.New("project id is required")
	}
	p.UpdatedAt = time.Now().UTC()
	_, err := s.db.Exec(`UPDATE projects SET
		name=?,slug=?,canonical_path=?,repo_remote=?,repo_url=?,default_branch=?,
		maestro_mode=?,resource_policy=?,default_isolation=?,settings=?,updated_at=?
		,display_path=?,identity_kind=?,identity_key=?
		WHERE id=?`,
		p.Name, p.Slug, p.CanonicalPath, p.RepoRemote, p.RepoURL, p.DefaultBranch,
		p.MaestroMode, p.ResourcePolicy, p.DefaultIsolation, p.Settings,
		p.UpdatedAt.Format(time.RFC3339Nano), p.DisplayPath, p.IdentityKind, p.IdentityKey, p.ID)
	return err
}

// TouchProject updates last_opened_at (used for MRU ordering).
func (s *Store) TouchProject(id string) error {
	_, err := s.db.Exec(`UPDATE projects SET last_opened_at=?, updated_at=? WHERE id=?`,
		time.Now().UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}

// DeleteProject removes a project and cascades agents/revisions/generations.
func (s *Store) DeleteProject(id string) error {
	_, err := s.db.Exec(`DELETE FROM projects WHERE id=?`, id)
	return err
}

// ErrRevisionConflict is returned when an optimistic revision check fails.
var ErrRevisionConflict = errors.New("layout revision conflict")

// ProjectLayoutRecord represents a durable layout record with monotonic revision.
type ProjectLayoutRecord struct {
	ProjectID string `json:"project_id"`
	Layout    string `json:"layout"`
	Revision  int64  `json:"revision"`
	UpdatedAt string `json:"updated_at"`
}

// SaveLayout persists a project cockpit layout unconditionally (advancing revision).
func (s *Store) SaveLayout(projectID, layout string) error {
	_, err := s.SaveLayoutWithRevision(projectID, layout, 0)
	return err
}

// SaveLayoutWithRevision atomically saves layout and increments revision if expectedRevision matches.
// If expectedRevision is 0, it saves unconditionally advancing the revision.
func (s *Store) SaveLayoutWithRevision(projectID, layout string, expectedRevision int64) (ProjectLayoutRecord, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := s.db.Begin()
	if err != nil {
		return ProjectLayoutRecord{}, err
	}
	defer tx.Rollback()

	var curRev int64
	err = tx.QueryRow(`SELECT revision FROM project_layouts WHERE project_id=?`, projectID).Scan(&curRev)
	if err != nil && err != sql.ErrNoRows {
		return ProjectLayoutRecord{}, err
	}

	exists := (err == nil)
	if expectedRevision > 0 && exists && curRev != expectedRevision {
		return ProjectLayoutRecord{}, ErrRevisionConflict
	}

	nextRev := curRev + 1
	if !exists {
		nextRev = 1
		_, err = tx.Exec(`INSERT INTO project_layouts(project_id, layout, revision, updated_at) VALUES(?,?,?,?)`,
			projectID, layout, nextRev, now)
	} else {
		_, err = tx.Exec(`UPDATE project_layouts SET layout=?, revision=?, updated_at=? WHERE project_id=?`,
			layout, nextRev, now, projectID)
	}
	if err != nil {
		return ProjectLayoutRecord{}, err
	}

	if err := tx.Commit(); err != nil {
		return ProjectLayoutRecord{}, err
	}

	return ProjectLayoutRecord{
		ProjectID: projectID,
		Layout:    layout,
		Revision:  nextRev,
		UpdatedAt: now,
	}, nil
}

// GetLayout loads a project cockpit layout string (backward compatibility).
func (s *Store) GetLayout(projectID string) (string, error) {
	rec, err := s.GetLayoutRecord(projectID)
	if err != nil {
		return "{}", err
	}
	return rec.Layout, nil
}

// GetLayoutRecord loads the full project layout record including monotonic revision.
func (s *Store) GetLayoutRecord(projectID string) (ProjectLayoutRecord, error) {
	var rec ProjectLayoutRecord
	rec.ProjectID = projectID
	err := s.db.QueryRow(`SELECT layout, revision, updated_at FROM project_layouts WHERE project_id=?`, projectID).
		Scan(&rec.Layout, &rec.Revision, &rec.UpdatedAt)
	if err == sql.ErrNoRows {
		return ProjectLayoutRecord{
			ProjectID: projectID,
			Layout:    "{}",
			Revision:  0,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		}, nil
	}
	return rec, err
}

func (s *Store) scanProject(row *sql.Row) (Project, error) {
	return scanProjectRows(row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProjectRows(row rowScanner) (Project, error) {
	var p Project
	var createdAt, updatedAt string
	var lastOpenedAt sql.NullString
	err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.CanonicalPath, &p.RepoRemote, &p.RepoURL,
		&p.DefaultBranch, &p.MaestroMode, &p.ResourcePolicy, &p.DefaultIsolation, &p.Settings,
		&createdAt, &updatedAt, &lastOpenedAt, &p.DisplayPath, &p.IdentityKind, &p.IdentityKey)
	if err == sql.ErrNoRows {
		return Project{}, ErrNotFound
	}
	if err != nil {
		return Project{}, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	if lastOpenedAt.Valid {
		if t, err := time.Parse(time.RFC3339Nano, lastOpenedAt.String); err == nil {
			p.LastOpenedAt = &t
		}
	}
	if p.DisplayPath == "" {
		p.DisplayPath = p.CanonicalPath
	}
	p.PathRef = config.PathRef{
		DisplayPath:   p.DisplayPath,
		CanonicalPath: p.CanonicalPath,
		Identity:      config.FilesystemIdentity{Kind: p.IdentityKind, StableKey: p.IdentityKey, Available: p.IdentityKind != "" && p.IdentityKey != ""},
	}
	return p, nil
}
