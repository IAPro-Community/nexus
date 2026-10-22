package store

import "fmt"

// repairTokenBarUsageSchema restores objects from migrations 0029/0030 when
// a partial database restore retained schema_migrations but lost a table or
// index. CREATE IF NOT EXISTS preserves existing observations and offsets.
// Incompatible table shapes are rejected, never dropped or silently replaced.
func (s *Store) repairTokenBarUsageSchema() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tokenbar usage repair: %w", err)
	}
	defer tx.Rollback()

	for _, name := range []string{
		"migrations/0029_capacity_pace_samples.sql",
		"migrations/0030_external_claude_usage.sql",
	} {
		body, err := migrationsFS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read tokenbar migration %s: %w", name, err)
		}
		for _, stmt := range splitSQLStatements(string(body)) {
			if _, err := tx.Exec(stmt); err != nil {
				return fmt.Errorf("repair tokenbar migration %s: %w", name, err)
			}
		}
	}

	for _, shape := range []struct {
		table   string
		columns []string
	}{
		{"capacity_pace_samples", []string{"provider", "profile", "account_scope", "quota_group", "window_kind", "reset_at", "observed_at", "used_percent", "source"}},
		{"external_claude_messages", []string{"source_scope", "message_hash", "model", "input_tokens", "output_tokens", "cache_read_tokens", "cache_write_tokens", "observed_at"}},
		{"external_claude_offsets", []string{"source_scope", "file_hash", "offset_bytes", "updated_at"}},
	} {
		cols, exists, err := tableColumns(tx, shape.table)
		if err != nil {
			return fmt.Errorf("inspect %s schema: %w", shape.table, err)
		}
		if !exists || !hasColumns(cols, shape.columns) {
			return fmt.Errorf("incompatible %s schema; restore from a verified backup", shape.table)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tokenbar usage repair: %w", err)
	}
	return nil
}
