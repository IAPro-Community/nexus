package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/ids"
)

const (
	ComposerExploring        = "EXPLORING"
	ComposerReadyWithGaps    = "READY_WITH_GAPS"
	ComposerReady            = "READY"
	ComposerFinalized        = "FINALIZED"
	ComposerUser             = "USER"
	ComposerAssistant        = "ASSISTANT"
	ComposerSkillSuggested   = "SUGGESTED"
	ComposerSkillAccepted    = "ACCEPTED"
	ComposerSkillApplied     = "APPLIED"
	ComposerSkillRejected    = "REJECTED"
	ComposerSkillUnavailable = "UNAVAILABLE"
)

// ComposerSession stores one durable, project-scoped prompt elaboration.
type ComposerSession struct {
	ID                 string    `json:"id"`
	ProjectID          string    `json:"project_id"`
	Title              string    `json:"title"`
	State              string    `json:"state"`
	ContextFingerprint string    `json:"context_fingerprint"`
	BriefJSON          string    `json:"brief_json"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
type ComposerTurn struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Sequence  int       `json:"sequence"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
type ComposerSkillProposal struct {
	SessionID     string    `json:"session_id"`
	SkillID       string    `json:"skill_id"`
	State         string    `json:"state"`
	Reason        string    `json:"reason"`
	Applicability string    `json:"applicability"`
	Risk          string    `json:"risk"`
	UpdatedAt     time.Time `json:"updated_at"`
}
type PromptArtifact struct {
	ID           string    `json:"id"`
	SessionID    string    `json:"session_id"`
	Version      int       `json:"version"`
	Content      string    `json:"content"`
	Hash         string    `json:"hash"`
	ContextJSON  string    `json:"context_json"`
	SkillIDsJSON string    `json:"skill_ids_json"`
	CreatedAt    time.Time `json:"created_at"`
}

func (s *Store) CreateComposerSession(in ComposerSession) (*ComposerSession, error) {
	if _, err := s.GetProject(in.ProjectID); err != nil {
		return nil, fmt.Errorf("composer project: %w", err)
	}
	if in.ID == "" {
		in.ID = "cmp_" + ids.NewRuntimeID()
	}
	if in.State == "" {
		in.State = ComposerExploring
	}
	if in.BriefJSON == "" {
		in.BriefJSON = "{}"
	}
	now := time.Now().UTC()
	in.CreatedAt, in.UpdatedAt = now, now
	_, err := s.db.Exec(`INSERT INTO composer_sessions(id,project_id,title,state,context_fingerprint,brief_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, in.ID, in.ProjectID, in.Title, in.State, in.ContextFingerprint, in.BriefJSON, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("create composer session: %w", err)
	}
	return &in, nil
}

func (s *Store) GetComposerSession(id string) (*ComposerSession, error) {
	var out ComposerSession
	var created, updated string
	err := s.db.QueryRow(`SELECT id,project_id,title,state,context_fingerprint,brief_json,created_at,updated_at FROM composer_sessions WHERE id=?`, id).Scan(&out.ID, &out.ProjectID, &out.Title, &out.State, &out.ContextFingerprint, &out.BriefJSON, &created, &updated)
	if err != nil {
		return nil, err
	}
	out.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	out.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &out, nil
}

func (s *Store) ListComposerSessions(projectID string) ([]ComposerSession, error) {
	rows, err := s.db.Query(`SELECT id,project_id,title,state,context_fingerprint,brief_json,created_at,updated_at FROM composer_sessions WHERE project_id=? ORDER BY updated_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ComposerSession{}
	for rows.Next() {
		var item ComposerSession
		var created, updated string
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Title, &item.State, &item.ContextFingerprint, &item.BriefJSON, &created, &updated); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) UpdateComposerSession(in ComposerSession) error {
	in.UpdatedAt = time.Now().UTC()
	res, err := s.db.Exec(`UPDATE composer_sessions SET title=?,state=?,context_fingerprint=?,brief_json=?,updated_at=? WHERE id=?`, in.Title, in.State, in.ContextFingerprint, in.BriefJSON, in.UpdatedAt.Format(time.RFC3339Nano), in.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) AppendComposerTurn(in ComposerTurn) (*ComposerTurn, error) {
	if in.ID == "" {
		in.ID = "ctn_" + ids.NewRuntimeID()
	}
	if in.Role != "USER" && in.Role != "ASSISTANT" {
		return nil, fmt.Errorf("invalid composer turn role")
	}
	in.Content = capComposerText(in.Content)
	if in.Content == "" {
		return nil, fmt.Errorf("composer turn content is required")
	}
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now().UTC()
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := tx.QueryRow(`SELECT COALESCE(MAX(sequence),0)+1 FROM composer_turns WHERE session_id=?`, in.SessionID).Scan(&in.Sequence); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`INSERT INTO composer_turns(id,session_id,sequence,role,content,created_at) VALUES(?,?,?,?,?,?)`, in.ID, in.SessionID, in.Sequence, in.Role, in.Content, in.CreatedAt.Format(time.RFC3339Nano)); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE composer_sessions SET updated_at=? WHERE id=?`, in.CreatedAt.Format(time.RFC3339Nano), in.SessionID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &in, nil
}

func (s *Store) ListComposerTurns(sessionID string, limit int) ([]ComposerTurn, error) {
	if limit <= 0 || limit > 40 {
		limit = 40
	}
	rows, err := s.db.Query(`SELECT id,session_id,sequence,role,content,created_at FROM composer_turns WHERE session_id=? ORDER BY sequence DESC LIMIT ?`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ComposerTurn{}
	for rows.Next() {
		var item ComposerTurn
		var created string
		if err := rows.Scan(&item.ID, &item.SessionID, &item.Sequence, &item.Role, &item.Content, &created); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append([]ComposerTurn{item}, out...)
	}
	return out, rows.Err()
}

func (s *Store) UpsertComposerSkillProposal(in ComposerSkillProposal) (*ComposerSkillProposal, error) {
	if strings.TrimSpace(in.SkillID) == "" {
		return nil, fmt.Errorf("skill id is required")
	}
	in.State = normalizeComposerSkillState(in.State)
	if in.State == "" {
		in.State = ComposerSkillSuggested
	}
	in.UpdatedAt = time.Now().UTC()
	_, err := s.db.Exec(`INSERT INTO composer_skill_proposals(session_id,skill_id,state,reason,applicability,risk,updated_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(session_id,skill_id) DO UPDATE SET state=excluded.state,reason=excluded.reason,applicability=excluded.applicability,risk=excluded.risk,updated_at=excluded.updated_at`, in.SessionID, in.SkillID, in.State, in.Reason, in.Applicability, in.Risk, in.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return &in, nil
}
func (s *Store) ListComposerSkillProposals(sessionID string) ([]ComposerSkillProposal, error) {
	rows, err := s.db.Query(`SELECT session_id,skill_id,state,reason,applicability,risk,updated_at FROM composer_skill_proposals WHERE session_id=? ORDER BY skill_id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ComposerSkillProposal{}
	for rows.Next() {
		var item ComposerSkillProposal
		var updated string
		if err := rows.Scan(&item.SessionID, &item.SkillID, &item.State, &item.Reason, &item.Applicability, &item.Risk, &updated); err != nil {
			return nil, err
		}
		item.State = normalizeComposerSkillState(item.State)
		item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) SetComposerSkillState(sessionID, skillID, state string) error {
	state = normalizeComposerSkillState(state)
	if state == "" {
		return fmt.Errorf("invalid composer skill state")
	}
	res, err := s.db.Exec(`UPDATE composer_skill_proposals SET state=?, updated_at=? WHERE session_id=? AND skill_id=?`, state, time.Now().UTC().Format(time.RFC3339Nano), sessionID, skillID)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CreatePromptArtifact(in PromptArtifact) (*PromptArtifact, error) {
	in.Content = capComposerText(in.Content)
	if in.Content == "" {
		return nil, fmt.Errorf("prompt artifact content is required")
	}
	if in.ID == "" {
		in.ID = "prm_" + ids.NewRuntimeID()
	}
	sum := sha256.Sum256([]byte(in.Content))
	in.Hash = hex.EncodeToString(sum[:])
	if in.ContextJSON == "" {
		in.ContextJSON = "{}"
	}
	if in.SkillIDsJSON == "" {
		in.SkillIDsJSON = "[]"
	}
	in.CreatedAt = time.Now().UTC()
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := tx.QueryRow(`SELECT COALESCE(MAX(version),0)+1 FROM prompt_artifacts WHERE session_id=?`, in.SessionID).Scan(&in.Version); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`INSERT INTO prompt_artifacts(id,session_id,version,content,content_hash,context_json,skill_ids_json,created_at) VALUES(?,?,?,?,?,?,?,?)`, in.ID, in.SessionID, in.Version, in.Content, in.Hash, in.ContextJSON, in.SkillIDsJSON, in.CreatedAt.Format(time.RFC3339Nano)); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &in, nil
}

func (s *Store) ListPromptArtifacts(sessionID string) ([]PromptArtifact, error) {
	rows, err := s.db.Query(`SELECT id,session_id,version,content,content_hash,context_json,skill_ids_json,created_at FROM prompt_artifacts WHERE session_id=? ORDER BY version DESC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PromptArtifact{}
	for rows.Next() {
		var item PromptArtifact
		var created string
		if err := rows.Scan(&item.ID, &item.SessionID, &item.Version, &item.Content, &item.Hash, &item.ContextJSON, &item.SkillIDsJSON, &created); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, item)
	}
	return out, rows.Err()
}

func capComposerText(value string) string {
	value = strings.TrimSpace(value)
	const max = 8 * 1024
	if len(value) > max {
		return value[:max]
	}
	return value
}

func normalizeComposerSkillState(state string) string {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case ComposerSkillSuggested:
		return ComposerSkillSuggested
	case "SELECTED", ComposerSkillAccepted:
		return ComposerSkillAccepted
	case ComposerSkillApplied:
		return ComposerSkillApplied
	case ComposerSkillRejected:
		return ComposerSkillRejected
	case ComposerSkillUnavailable:
		return ComposerSkillUnavailable
	default:
		return ""
	}
}

var _ = sql.ErrNoRows
