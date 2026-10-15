package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ExternalClaudeMessage is token-count metadata from a user-consented local
// transcript scan. Messages remain outside the canonical CostObservation ledger.
type ExternalClaudeMessage struct {
	MessageHash      string
	Model            string
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	ObservedAt       time.Time
}

type ExternalClaudeSummary struct {
	Source           string    `json:"source"`
	Unattributed     bool      `json:"unattributed"`
	Messages         int64     `json:"messages"`
	InputTokens      int64     `json:"input_tokens"`
	OutputTokens     int64     `json:"output_tokens"`
	CacheReadTokens  int64     `json:"cache_read_tokens"`
	CacheWriteTokens int64     `json:"cache_write_tokens"`
	LastObservedAt   *time.Time `json:"last_observed_at,omitempty"`
}

// ExternalClaudeOffset returns only metadata: never raw logs or file paths.
func (s *Store) ExternalClaudeOffset(ctx context.Context, scope, fileHash string) (int64, error) {
	if s == nil || s.db == nil { return 0, errors.New("external usage store unavailable") }
	var offset int64
	err := s.db.QueryRowContext(ctx,
		"SELECT offset_bytes FROM external_claude_offsets WHERE source_scope=? AND file_hash=?",
		scope, fileHash).Scan(&offset)
	if errors.Is(err, sql.ErrNoRows) { return 0, nil }
	return offset, err
}

// SaveExternalClaudeBatch atomically persists message usage and the last fully
// parsed newline offset. A failed commit cannot skip data on a future scan.
func (s *Store) SaveExternalClaudeBatch(ctx context.Context, scope, fileHash string, offset int64, messages []ExternalClaudeMessage) error {
	if s == nil || s.db == nil { return errors.New("external usage store unavailable") }
	if scope != "LOCAL_DEFAULT_UNVERIFIED" || fileHash == "" || offset < 0 {
		return errors.New("invalid external usage source")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()
	for _, m := range messages {
		if m.MessageHash == "" || m.ObservedAt.IsZero() || m.InputTokens < 0 ||
			m.OutputTokens < 0 || m.CacheReadTokens < 0 || m.CacheWriteTokens < 0 {
			return errors.New("invalid external transcript usage")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO external_claude_messages (
			source_scope,message_hash,model,input_tokens,output_tokens,
			cache_read_tokens,cache_write_tokens,observed_at
		) VALUES (?,?,?,?,?,?,?,?)
		ON CONFLICT(source_scope,message_hash) DO UPDATE SET
			model=excluded.model,
			input_tokens=excluded.input_tokens,
			output_tokens=excluded.output_tokens,
			cache_read_tokens=excluded.cache_read_tokens,
			cache_write_tokens=excluded.cache_write_tokens,
			observed_at=excluded.observed_at`,
			scope, m.MessageHash, m.Model, m.InputTokens, m.OutputTokens,
			m.CacheReadTokens, m.CacheWriteTokens, formatStoreTime(m.ObservedAt)); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO external_claude_offsets (
		source_scope,file_hash,offset_bytes,updated_at
	) VALUES(?,?,?,?) ON CONFLICT(source_scope,file_hash) DO UPDATE SET
		offset_bytes=excluded.offset_bytes,updated_at=excluded.updated_at`,
		scope, fileHash, offset, formatStoreTime(time.Now().UTC())); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) SummarizeExternalClaude(ctx context.Context, days int) (ExternalClaudeSummary, error) {
	out := ExternalClaudeSummary{Source: "LOCAL_DEFAULT_UNVERIFIED", Unattributed: true}
	if s == nil || s.db == nil { return out, errors.New("external usage store unavailable") }
	if days < 1 || days > 365 { return out, errors.New("days must be between 1 and 365") }
	var last sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(input_tokens),0),
		COALESCE(SUM(output_tokens),0),COALESCE(SUM(cache_read_tokens),0),
		COALESCE(SUM(cache_write_tokens),0),MAX(observed_at)
		FROM external_claude_messages WHERE source_scope=? AND observed_at>=?`,
		out.Source, formatStoreTime(time.Now().UTC().Add(-time.Duration(days)*24*time.Hour)),
	).Scan(&out.Messages,&out.InputTokens,&out.OutputTokens,&out.CacheReadTokens,&out.CacheWriteTokens,&last)
	if err != nil { return out, err }
	if last.Valid {
		parsed, parseErr := parseStoreTime(last.String)
		if parseErr != nil { return out, parseErr }
		out.LastObservedAt = &parsed
	}
	return out, err
}

/* ClearExternalClaude allows consent revocation without touching canonical
   costs or quota telemetry. Offsets must also be deleted so a future
   opt-in scan can re-read all available files. */
func (s *Store) ClearExternalClaude(ctx context.Context) error {
	if s == nil || s.db == nil { return errors.New("external usage store unavailable") }
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM external_claude_messages WHERE source_scope=?", "LOCAL_DEFAULT_UNVERIFIED"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM external_claude_offsets WHERE source_scope=?", "LOCAL_DEFAULT_UNVERIFIED"); err != nil {
		return err
	}
	return tx.Commit()
}
