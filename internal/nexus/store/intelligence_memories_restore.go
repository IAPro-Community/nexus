package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ReplaceProjectIntelligenceMemoriesTransactional is the SQLite-only, exact
// restoration primitive. It is intentionally NOT connected to the CLI or a
// network route: the INSTALL config store needs an independent crash-recovery
// journal and a cross-process writer fence before a full restore is safe.
//
// expectedBefore is a compare-and-swap precondition checked inside the same
// SQLite transaction. Rows in other projects are never selected or deleted.
// Original IDs and timestamps are preserved, including pending.* markers.
func (s *Store) ReplaceProjectIntelligenceMemoriesTransactional(
	ctx context.Context, projectID string,
	expectedBefore, replacement []IntelligenceMemory,
) error {
	if strings.TrimSpace(projectID) == "" || strings.TrimSpace(projectID) != projectID {
		return errors.New("project ID is required")
	}
	if err := validateRestoreRows(projectID, expectedBefore); err != nil {
		return fmt.Errorf("invalid expected memory set: %w", err)
	}
	if err := validateRestoreRows(projectID, replacement); err != nil {
		return fmt.Errorf("invalid restore memory set: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin project memory restore: %w", err)
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM projects WHERE id=?)", projectID).Scan(&exists); err != nil {
		return fmt.Errorf("check restore project existence: %w", err)
	}
	if exists != 1 {
		return ErrNotFound
	}
	current, err := readRestoreProjectRows(ctx, tx, projectID)
	if err != nil {
		return err
	}
	if !sameRestoreRows(current, expectedBefore) {
		return errors.New("project memories changed since restore preview (CAS precondition failed)")
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM intelligence_memories WHERE project_id=?", projectID); err != nil {
		return fmt.Errorf("delete project restore rows: %w", err)
	}
	for _, m := range replacement {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO intelligence_memories(id,project_id,key,value,source,created_at,updated_at)
			 VALUES(?,?,?,?,?,?,?)`,
			m.ID, projectID, m.Key, m.Value, m.Source,
			m.CreatedAt.UTC().Format(time.RFC3339Nano),
			m.UpdatedAt.UTC().Format(time.RFC3339Nano))
		if err != nil {
			return fmt.Errorf("insert project restore row: %w", err)
		}
	}
	verified, err := readRestoreProjectRows(ctx, tx, projectID)
	if err != nil {
		return err
	}
	if !sameRestoreRows(verified, replacement) {
		return errors.New("project memory restore read-back differs from expected snapshot")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit project memory restore: %w", err)
	}
	return nil
}

func validateRestoreRows(projectID string, rows []IntelligenceMemory) error {
	byID := make(map[string]bool, len(rows))
	byKey := make(map[string]bool, len(rows))
	for _, row := range rows {
		if row.ProjectID != projectID || row.ID == "" ||
			strings.TrimSpace(row.Key) == "" ||
			strings.TrimSpace(row.Value) == "" ||
			row.CreatedAt.IsZero() || row.UpdatedAt.IsZero() {
			return errors.New("memory snapshot contains an invalid or cross-project row")
		}
		if byID[row.ID] || byKey[row.Key] {
			return errors.New("memory snapshot contains duplicate IDs or keys")
		}
		byID[row.ID] = true
		byKey[row.Key] = true
	}
	return nil
}

func readRestoreProjectRows(ctx context.Context, tx *sql.Tx, projectID string) ([]IntelligenceMemory, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id,project_id,key,value,source,created_at,updated_at
		FROM intelligence_memories WHERE project_id=? ORDER BY key, id`, projectID)
	if err != nil {
		return nil, fmt.Errorf("read transactional memory snapshot: %w", err)
	}
	defer rows.Close()
	out := make([]IntelligenceMemory, 0)
	for rows.Next() {
		var m IntelligenceMemory
		var created, updated string
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.Key, &m.Value, &m.Source, &created, &updated); err != nil {
			return nil, err
		}
		m.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, errors.New("invalid created_at in project memory store")
		}
		m.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
		if err != nil {
			return nil, errors.New("invalid updated_at in project memory store")
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func sameRestoreRows(a, b []IntelligenceMemory) bool {
	if len(a) != len(b) {
		return false
	}
	key := func(rows []IntelligenceMemory) []string {
		out := make([]string, len(rows))
		for i, v := range rows {
			out[i] = v.ID + "\x00" + v.ProjectID + "\x00" + v.Key + "\x00" +
				v.Value + "\x00" + v.Source + "\x00" +
				v.CreatedAt.UTC().Format(time.RFC3339Nano) + "\x00" +
				v.UpdatedAt.UTC().Format(time.RFC3339Nano)
		}
		sort.Strings(out)
		return out
	}
	left, right := key(a), key(b)
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
