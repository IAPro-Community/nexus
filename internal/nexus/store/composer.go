package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
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
	ID                  string    `json:"id"`
	ProjectID           string    `json:"project_id"`
	Title               string    `json:"title"`
	State               string    `json:"state"`
	ContextFingerprint  string    `json:"context_fingerprint"`
	BriefJSON           string    `json:"brief_json"`
	ContextManifestJSON string    `json:"-"`
	Revision            int       `json:"revision"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
type ComposerTurn struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	Sequence    int       `json:"sequence"`
	Role        string    `json:"role"`
	Content     string    `json:"content"`
	Kind        string    `json:"kind,omitempty"`
	PayloadJSON string    `json:"payload_json,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ConversationAttachment is a durable file reference owned by a composer/session thread.
type ConversationAttachment struct {
	ID           string    `json:"id"`
	SessionID    string    `json:"session_id"`
	Name         string    `json:"name"`
	MimeType     string    `json:"mime_type"`
	Size         int64     `json:"size"`
	Type         string    `json:"type"`
	StorageRef   string    `json:"storage_ref"`
	MetadataJSON string    `json:"metadata_json,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
type ComposerSkillProposal struct {
	SessionID           string    `json:"session_id"`
	SkillID             string    `json:"skill_id"`
	State               string    `json:"state"`
	Reason              string    `json:"reason"`
	Applicability       string    `json:"applicability"`
	Risk                string    `json:"risk"`
	Source              string    `json:"source,omitempty"`
	Version             string    `json:"version,omitempty"`
	Available           bool      `json:"available"`
	RecommendationScore int       `json:"recommendation_score,omitempty"`
	UpdatedAt           time.Time `json:"updated_at"`
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

type PromptVariant struct {
	ID               string    `json:"id"`
	ArtifactID       string    `json:"artifact_id"`
	Variant          string    `json:"variant"`
	Target           string    `json:"target,omitempty"`
	Content          string    `json:"content"`
	Hash             string    `json:"hash"`
	CapabilitiesJSON string    `json:"capabilities_json,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

type ComposerDestinationReceipt struct {
	ID           string    `json:"id"`
	ArtifactID   string    `json:"artifact_id"`
	Destination  string    `json:"destination"`
	Variant      string    `json:"variant"`
	Status       string    `json:"status"`
	MetadataJSON string    `json:"metadata_json,omitempty"`
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
	if in.Revision <= 0 {
		in.Revision = 1
	}
	if in.ContextManifestJSON == "" {
		in.ContextManifestJSON = "{}"
	}
	_, err := s.db.Exec(`INSERT INTO composer_sessions(id,project_id,title,state,context_fingerprint,brief_json,context_manifest_json,revision,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, in.ID, in.ProjectID, in.Title, in.State, in.ContextFingerprint, in.BriefJSON, in.ContextManifestJSON, in.Revision, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("create composer session: %w", err)
	}
	return &in, nil
}

func (s *Store) GetComposerSession(id string) (*ComposerSession, error) {
	var out ComposerSession
	var created, updated string
	err := s.db.QueryRow(`SELECT id,project_id,title,state,context_fingerprint,brief_json,context_manifest_json,revision,created_at,updated_at FROM composer_sessions WHERE id=?`, id).Scan(&out.ID, &out.ProjectID, &out.Title, &out.State, &out.ContextFingerprint, &out.BriefJSON, &out.ContextManifestJSON, &out.Revision, &created, &updated)
	if err != nil {
		return nil, err
	}
	out.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	out.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &out, nil
}

func (s *Store) ListComposerSessions(projectID string) ([]ComposerSession, error) {
	rows, err := s.db.Query(`SELECT id,project_id,title,state,context_fingerprint,brief_json,context_manifest_json,revision,created_at,updated_at FROM composer_sessions WHERE project_id=? ORDER BY updated_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ComposerSession{}
	for rows.Next() {
		var item ComposerSession
		var created, updated string
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Title, &item.State, &item.ContextFingerprint, &item.BriefJSON, &item.ContextManifestJSON, &item.Revision, &created, &updated); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, item)
	}
	return out, rows.Err()
}

// ErrComposerDeleteProtected retains the provenance of plans and deliveries
// already derived from a Composer session.
var ErrComposerDeleteProtected = fmt.Errorf("elaboração vinculada a um Flow ou entrega; exclua apenas planejamentos sem dependências")

// DeleteComposerSessionExpected deletes one unreferenced elaboration and its
// turns, revisions, skill proposals, attachments, artifact variants and drafts.
// The checks and delete share one SQLite transaction; no unrelated project is
// affected. A linked WorkPlan must be deleted safely first.
func (s *Store) DeleteComposerSessionExpected(id string, expectedRevision int) error {
	tx, err := s.db.Begin()
	if err != nil { return err }
	defer tx.Rollback()
	var revision int
	if err := tx.QueryRow(`SELECT revision FROM composer_sessions WHERE id=?`, id).Scan(&revision); err != nil {
		return err
	}
	if expectedRevision > 0 && revision != expectedRevision {
		return ErrPlanRevisionConflict
	}
	artifacts := map[string]struct{}{}
	rows, err := tx.Query(`SELECT id FROM prompt_artifacts WHERE session_id=?`, id)
	if err != nil { return err }
	for rows.Next() {
		var artifactID string
		if err := rows.Scan(&artifactID); err != nil { rows.Close(); return err }
		artifacts[artifactID] = struct{}{}
	}
	if err := rows.Err(); err != nil { rows.Close(); return err }
	rows.Close()
	// Any existing delivery receipt means this planning record was used outside
	// the current drafting surface. Keep its history rather than deleting proof.
	var delivered int
	if err := tx.QueryRow(`SELECT EXISTS(
		SELECT 1 FROM composer_destination_receipts r
		JOIN prompt_artifacts a ON a.id=r.artifact_id
		WHERE a.session_id=?
	)`, id).Scan(&delivered); err != nil { return err }
	if delivered != 0 { return ErrComposerDeleteProtected }

	// Cross-project plans can retain imported artifact lineage as well.
	plans, err := tx.Query(`SELECT facts_json FROM work_plans`)
	if err != nil { return err }
	for plans.Next() {
		var raw string
		if err := plans.Scan(&raw); err != nil { plans.Close(); return err }
		var facts map[string]string
		if err := json.Unmarshal([]byte(raw), &facts); err != nil { plans.Close(); return fmt.Errorf("cannot verify WorkPlan lineage before Composer deletion: %w", err) }
		if facts["nexus.source_composer_session_id"] == id {
			plans.Close()
			return ErrComposerDeleteProtected
		}
		if _, linked := artifacts[facts["nexus.source_artifact_id"]]; linked {
			plans.Close()
			return ErrComposerDeleteProtected
		}
		// Imported/cloned plans may carry only the versioned Composer handoff,
		// without older flattened nexus.source_artifact_id fields.
		if raw := strings.TrimSpace(facts["composer.handoff.v1"]); raw != "" {
			var handoff struct {
				SessionID string `json:"session_id"`
				ArtifactID string `json:"artifact_id"`
			}
			if err := json.Unmarshal([]byte(raw), &handoff); err != nil {
				plans.Close()
				return fmt.Errorf("cannot verify imported Composer handoff before deletion: %w", err)
			}
			if handoff.SessionID == id {
				plans.Close()
				return ErrComposerDeleteProtected
			}
			if _, linked := artifacts[handoff.ArtifactID]; linked {
				plans.Close()
				return ErrComposerDeleteProtected
			}
		}
	}
	if err := plans.Err(); err != nil { plans.Close(); return err }
	plans.Close()
	res, err := tx.Exec(`DELETE FROM composer_sessions WHERE id=? AND revision=?`, id, revision)
	if err != nil { return err }
	if count, err := res.RowsAffected(); err != nil { return err } else if count != 1 {
		return ErrPlanRevisionConflict
	}
	return tx.Commit()
}

func (s *Store) UpdateComposerSession(in ComposerSession) error {
	in.UpdatedAt = time.Now().UTC()
	if in.ContextManifestJSON == "" {
		in.ContextManifestJSON = "{}"
	}
	res, err := s.db.Exec(`UPDATE composer_sessions SET title=?,state=?,context_fingerprint=?,brief_json=?,context_manifest_json=?,revision=revision+1,updated_at=? WHERE id=?`, in.Title, in.State, in.ContextFingerprint, in.BriefJSON, in.ContextManifestJSON, in.UpdatedAt.Format(time.RFC3339Nano), in.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateComposerSessionExpected(in ComposerSession, expectedRevision int) error {
	if expectedRevision <= 0 {
		return s.UpdateComposerSession(in)
	}
	in.UpdatedAt = time.Now().UTC()
	if in.ContextManifestJSON == "" {
		in.ContextManifestJSON = "{}"
	}
	res, err := s.db.Exec(`UPDATE composer_sessions SET title=?,state=?,context_fingerprint=?,brief_json=?,context_manifest_json=?,revision=revision+1,updated_at=? WHERE id=? AND revision=?`, in.Title, in.State, in.ContextFingerprint, in.BriefJSON, in.ContextManifestJSON, in.UpdatedAt.Format(time.RFC3339Nano), in.ID, expectedRevision)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("composer revision conflict for %s", in.ID)
	}
	return nil
}

const maxComposerTurnBytes = 256 * 1024

func (s *Store) AppendComposerTurn(in ComposerTurn) (*ComposerTurn, error) {
	if in.ID == "" {
		in.ID = "ctn_" + ids.NewRuntimeID()
	}
	if in.Role != "USER" && in.Role != "ASSISTANT" {
		return nil, fmt.Errorf("invalid composer turn role")
	}
	in.Content = strings.TrimSpace(in.Content)
	if len(in.Content) > maxComposerTurnBytes {
		return nil, fmt.Errorf("composer turn content exceeds %d-byte limit", maxComposerTurnBytes)
	}
	if in.Content == "" && strings.TrimSpace(in.PayloadJSON) == "" {
		return nil, fmt.Errorf("composer turn content is required")
	}
	if in.Content == "" {
		in.Content = " " // placeholder so legacy readers still see a non-empty body
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
	if _, err := tx.Exec(`INSERT INTO composer_turns(id,session_id,sequence,role,content,kind,payload_json,created_at) VALUES(?,?,?,?,?,?,?,?)`, in.ID, in.SessionID, in.Sequence, in.Role, in.Content, in.Kind, in.PayloadJSON, in.CreatedAt.Format(time.RFC3339Nano)); err != nil {
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
	rows, err := s.db.Query(`SELECT id,session_id,sequence,role,content,COALESCE(kind,''),COALESCE(payload_json,''),created_at FROM composer_turns WHERE session_id=? ORDER BY sequence DESC LIMIT ?`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ComposerTurn{}
	for rows.Next() {
		var item ComposerTurn
		var created string
		if err := rows.Scan(&item.ID, &item.SessionID, &item.Sequence, &item.Role, &item.Content, &item.Kind, &item.PayloadJSON, &created); err != nil {
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
	if in.Source == "" {
		in.Source = "Nexus"
	}
	_, err := s.db.Exec(`INSERT INTO composer_skill_proposals(session_id,skill_id,state,reason,applicability,risk,source,version,available,recommendation_score,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(session_id,skill_id) DO UPDATE SET state=excluded.state,reason=excluded.reason,applicability=excluded.applicability,risk=excluded.risk,source=excluded.source,version=excluded.version,available=excluded.available,recommendation_score=excluded.recommendation_score,updated_at=excluded.updated_at`, in.SessionID, in.SkillID, in.State, in.Reason, in.Applicability, in.Risk, in.Source, in.Version, boolToInt(in.Available), in.RecommendationScore, in.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return &in, nil
}
func (s *Store) ListComposerSkillProposals(sessionID string) ([]ComposerSkillProposal, error) {
	rows, err := s.db.Query(`SELECT session_id,skill_id,state,reason,applicability,risk,source,version,available,recommendation_score,updated_at FROM composer_skill_proposals WHERE session_id=? ORDER BY recommendation_score DESC, skill_id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ComposerSkillProposal{}
	for rows.Next() {
		var item ComposerSkillProposal
		var updated string
		var available int
		if err := rows.Scan(&item.SessionID, &item.SkillID, &item.State, &item.Reason, &item.Applicability, &item.Risk, &item.Source, &item.Version, &available, &item.RecommendationScore, &updated); err != nil {
			return nil, err
		}
		item.Available = available != 0
		item.State = normalizeComposerSkillState(item.State)
		item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) DeleteSuggestedComposerSkillProposal(sessionID, skillID string) error {
	res, err := s.db.Exec(`DELETE FROM composer_skill_proposals WHERE session_id=? AND skill_id=? AND state=?`, sessionID, skillID, ComposerSkillSuggested)
	if err != nil {
		return err
	}
	_, _ = res.RowsAffected()
	return nil
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

const maxPromptArtifactBytes = 256 * 1024

func (s *Store) CreatePromptArtifact(in PromptArtifact) (*PromptArtifact, error) {
	in.Content = strings.TrimSpace(in.Content)
	if in.Content == "" {
		return nil, fmt.Errorf("prompt artifact content is required")
	}
	if len(in.Content) > maxPromptArtifactBytes {
		return nil, fmt.Errorf("prompt artifact content exceeds %d-byte limit", maxPromptArtifactBytes)
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

func (s *Store) GetPromptArtifact(id string) (*PromptArtifact, error) {
	var item PromptArtifact
	var created string
	err := s.db.QueryRow(`SELECT id,session_id,version,content,content_hash,context_json,skill_ids_json,created_at FROM prompt_artifacts WHERE id=?`, id).
		Scan(&item.ID, &item.SessionID, &item.Version, &item.Content, &item.Hash, &item.ContextJSON, &item.SkillIDsJSON, &created)
	if err != nil {
		return nil, err
	}
	item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return &item, nil
}

func (s *Store) CreatePromptVariant(in PromptVariant) (*PromptVariant, error) {
	if strings.TrimSpace(in.ArtifactID) == "" || strings.TrimSpace(in.Variant) == "" || strings.TrimSpace(in.Content) == "" {
		return nil, fmt.Errorf("prompt variant requires artifact, variant and content")
	}
	if in.ID == "" {
		in.ID = "pvr_" + ids.NewRuntimeID()
	}
	sum := sha256.Sum256([]byte(in.Content))
	in.Hash = hex.EncodeToString(sum[:])
	if in.CapabilitiesJSON == "" {
		in.CapabilitiesJSON = "{}"
	}
	in.CreatedAt = time.Now().UTC()
	_, err := s.db.Exec(`INSERT INTO composer_prompt_variants(id,artifact_id,variant,target,content,content_hash,capabilities_json,created_at) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(artifact_id,variant) DO UPDATE SET target=excluded.target,content=excluded.content,content_hash=excluded.content_hash,capabilities_json=excluded.capabilities_json`, in.ID, in.ArtifactID, in.Variant, in.Target, in.Content, in.Hash, in.CapabilitiesJSON, in.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return &in, nil
}

func (s *Store) ListPromptVariants(artifactID string) ([]PromptVariant, error) {
	rows, err := s.db.Query(`SELECT id,artifact_id,variant,target,content,content_hash,capabilities_json,created_at FROM composer_prompt_variants WHERE artifact_id=? ORDER BY variant`, artifactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PromptVariant{}
	for rows.Next() {
		var v PromptVariant
		var created string
		if err := rows.Scan(&v.ID, &v.ArtifactID, &v.Variant, &v.Target, &v.Content, &v.Hash, &v.CapabilitiesJSON, &created); err != nil {
			return nil, err
		}
		v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) RecordComposerDestinationReceipt(in ComposerDestinationReceipt) (*ComposerDestinationReceipt, error) {
	if in.ID == "" {
		in.ID = "cdr_" + ids.NewRuntimeID()
	}
	if in.MetadataJSON == "" {
		in.MetadataJSON = "{}"
	}
	in.CreatedAt = time.Now().UTC()
	_, err := s.db.Exec(`INSERT INTO composer_destination_receipts(id,artifact_id,destination,variant,status,metadata_json,created_at) VALUES(?,?,?,?,?,?,?)`, in.ID, in.ArtifactID, in.Destination, in.Variant, in.Status, in.MetadataJSON, in.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return &in, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
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

func (s *Store) CreateConversationAttachment(in ConversationAttachment) (*ConversationAttachment, error) {
	if in.ID == "" {
		in.ID = "att_" + ids.NewRuntimeID()
	}
	if strings.TrimSpace(in.SessionID) == "" || strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("session_id and name are required")
	}
	if in.Type == "" {
		in.Type = "image"
	}
	if in.MetadataJSON == "" {
		in.MetadataJSON = "{}"
	}
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.Exec(
		`INSERT INTO conversation_attachments(id,session_id,name,mime_type,size,type,storage_ref,metadata_json,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		in.ID, in.SessionID, in.Name, in.MimeType, in.Size, in.Type, in.StorageRef, in.MetadataJSON, in.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, err
	}
	return &in, nil
}

func (s *Store) GetConversationAttachment(id string) (*ConversationAttachment, error) {
	var out ConversationAttachment
	var created string
	err := s.db.QueryRow(
		`SELECT id,session_id,name,mime_type,size,type,storage_ref,COALESCE(metadata_json,'{}'),created_at FROM conversation_attachments WHERE id=?`,
		id,
	).Scan(&out.ID, &out.SessionID, &out.Name, &out.MimeType, &out.Size, &out.Type, &out.StorageRef, &out.MetadataJSON, &created)
	if err != nil {
		return nil, err
	}
	out.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return &out, nil
}

func (s *Store) ListConversationAttachments(sessionID string) ([]ConversationAttachment, error) {
	rows, err := s.db.Query(
		`SELECT id,session_id,name,mime_type,size,type,storage_ref,COALESCE(metadata_json,'{}'),created_at FROM conversation_attachments WHERE session_id=? ORDER BY created_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ConversationAttachment{}
	for rows.Next() {
		var item ConversationAttachment
		var created string
		if err := rows.Scan(&item.ID, &item.SessionID, &item.Name, &item.MimeType, &item.Size, &item.Type, &item.StorageRef, &item.MetadataJSON, &created); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) UpdateConversationAttachment(in ConversationAttachment) error {
	if strings.TrimSpace(in.ID) == "" {
		return fmt.Errorf("attachment id is required")
	}
	_, err := s.db.Exec(
		`UPDATE conversation_attachments SET name=?,mime_type=?,size=?,type=?,storage_ref=?,metadata_json=? WHERE id=?`,
		in.Name, in.MimeType, in.Size, in.Type, in.StorageRef, in.MetadataJSON, in.ID,
	)
	return err
}

func (s *Store) DeleteConversationAttachment(id string) error {
	_, err := s.db.Exec(`DELETE FROM conversation_attachments WHERE id=?`, id)
	return err
}

var _ = sql.ErrNoRows
