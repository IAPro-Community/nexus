// Package store provides the durable Nexus product state backed by SQLite
// (pure-Go driver, no CGO, single portable binary).
//
// The SQLite store is the product domain store (projects, agents, revisions,
// runtime generations, lineage, layouts). Live low-level runtime state stays in
// internal/control/registry.
package store

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store is the Nexus durable product state handle.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the Nexus SQLite database at path and runs
// all pending schema migrations idempotently.
func Open(path string) (*Store, error) {
	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open nexus store: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping nexus store: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close flushes pending WAL pages and closes the underlying database.
func (s *Store) Close() error {
	// Checkpoint WAL to main database before closing so Windows does not
	// leave a locked -wal file behind.
	_, _ = s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return s.db.Close()
}

// DB exposes the underlying handle for advanced queries.
func (s *Store) DB() *sql.DB { return s.db }

// migrate applies embedded migrations in filename order, recording each
// applied version in schema_migrations. Re-runs are no-ops.
func (s *Store) migrate() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(entries)

	for _, entry := range entries {
		name := filepath.Base(entry)
		var version int
		if _, err := fmt.Sscanf(name, "%d_", &version); err != nil {
			continue
		}
		applied := false
		_ = s.db.QueryRow(`SELECT 1 FROM schema_migrations WHERE version=?`, version).Scan(&applied)
		_ = applied
		var exists int
		err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=?)`, version).Scan(&exists)
		if err != nil {
			return err
		}
		if exists == 1 {
			continue
		}

		body, err := migrationsFS.ReadFile(entry)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		for _, stmt := range splitSQLStatements(string(body)) {
			if _, err := tx.Exec(stmt); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("apply migration %s: %w", name, err)
			}
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations(version) VALUES(?)`, version); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	// Migration numbers alone are insufficient: databases can have a recorded
	// migration with a missing or incompatible table (for example after a
	// partial restore). Validate the structural contract on every open and
	// repair drift idempotently before exposing the store to callers.
	if err := s.repairFlowEvidenceSchema(); err != nil {
		return err
	}
	if err := s.repairComposerConversationSchema(); err != nil {
		return err
	}
	return nil
}

// splitSQLStatements splits a migration file into executable statements.
// modernc.org/sqlite Exec accepts only one statement per call; multi-statement
// files were previously marked applied after running only the first clause.
func splitSQLStatements(body string) []string {
	var cleaned strings.Builder
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		cleaned.WriteString(line)
		cleaned.WriteByte('\n')
	}
	parts := strings.Split(cleaned.String(), ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		stmt := strings.TrimSpace(part)
		if stmt == "" {
			continue
		}
		out = append(out, stmt)
	}
	return out
}

func (s *Store) repairComposerConversationSchema() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin composer conversation repair: %w", err)
	}
	defer tx.Rollback()

	cols, exists, err := tableColumns(tx, "composer_turns")
	if err != nil {
		return err
	}
	if exists {
		if !cols["kind"] {
			if _, err := tx.Exec(`ALTER TABLE composer_turns ADD COLUMN kind TEXT NOT NULL DEFAULT ''`); err != nil {
				return fmt.Errorf("add composer_turns.kind: %w", err)
			}
		}
		if !cols["payload_json"] {
			if _, err := tx.Exec(`ALTER TABLE composer_turns ADD COLUMN payload_json TEXT NOT NULL DEFAULT ''`); err != nil {
				return fmt.Errorf("add composer_turns.payload_json: %w", err)
			}
		}
	}

	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS conversation_attachments (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES composer_sessions(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		mime_type TEXT NOT NULL,
		size INTEGER NOT NULL,
		type TEXT NOT NULL,
		storage_ref TEXT NOT NULL,
		metadata_json TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create conversation_attachments: %w", err)
	}
	if err := ensureRepairLedger(tx); err != nil {
		return err
	}
	// Shape repair must precede index creation: the index references
	// created_at, which a drifted pre-existing table may lack.
	if err := repairConversationAttachmentsShape(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_conversation_attachments_session
		ON conversation_attachments(session_id, created_at DESC)`); err != nil {
		return fmt.Errorf("index conversation_attachments: %w", err)
	}
	if err := ensureComposerTurnsTable(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit composer conversation repair: %w", err)
	}
	return nil
}

// repairConversationAttachmentsShape preserves rows when a pre-existing
// conversation_attachments table does not match the canonical shape: the old
// table is kept as *_legacy_* (files on disk are untouched) and every
// decision is recorded in schema_repair_ledger.
func repairConversationAttachmentsShape(tx *sql.Tx) error {
	required := []string{"id", "session_id", "name", "mime_type", "size", "type", "storage_ref", "metadata_json", "created_at"}
	cols, exists, err := tableColumns(tx, "conversation_attachments")
	if err != nil {
		return err
	}
	if !exists || hasColumns(cols, required) {
		return nil
	}
	before := legacyTableCount(tx, "conversation_attachments")
	legacy := "conversation_attachments_legacy_" + time.Now().UTC().Format("20060102150405")
	if _, err := tx.Exec(`ALTER TABLE "conversation_attachments" RENAME TO ` + quoteIdent(legacy)); err != nil {
		return fmt.Errorf("preserve legacy conversation_attachments: %w", err)
	}
	if _, err := tx.Exec(`CREATE TABLE conversation_attachments (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES composer_sessions(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		mime_type TEXT NOT NULL,
		size INTEGER NOT NULL,
		type TEXT NOT NULL,
		storage_ref TEXT NOT NULL,
		metadata_json TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("recreate conversation_attachments: %w", err)
	}
	lcols, _, _ := tableColumns(tx, legacy)
	copied, skipped := int64(0), before
	if hasColumns(lcols, []string{"id", "session_id", "name", "mime_type", "size", "type", "storage_ref", "created_at"}) {
		res, err := tx.Exec(`INSERT OR IGNORE INTO conversation_attachments(id,session_id,name,mime_type,size,type,storage_ref,metadata_json,created_at)
			SELECT id,session_id,name,mime_type,size,type,storage_ref,COALESCE(metadata_json,'{}'),created_at FROM ` + quoteIdent(legacy))
		if err == nil {
			copied, _ = res.RowsAffected()
			skipped = before - copied
		}
	}
	recordRepair(tx, "conversation_attachments", legacy, copied, skipped,
		"pre-existing table had an incompatible shape; rows kept in legacy table, files on disk untouched")
	return nil
}

// ensureComposerTurnsTable recreates a wholly missing composer_turns table
// with the canonical migration shape. A missing table holds no rows, so
// creation loses nothing; a present table is never touched here.
func ensureComposerTurnsTable(tx *sql.Tx) error {
	_, exists, err := tableColumns(tx, "composer_turns")
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if _, err := tx.Exec(`CREATE TABLE composer_turns (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES composer_sessions(id) ON DELETE CASCADE,
		sequence INTEGER NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at TEXT NOT NULL,
		kind TEXT NOT NULL DEFAULT '',
		payload_json TEXT NOT NULL DEFAULT '',
		UNIQUE(session_id, sequence)
	)`); err != nil {
		return fmt.Errorf("recreate missing composer_turns: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_composer_turns_session ON composer_turns(session_id, sequence DESC)`); err != nil {
		return fmt.Errorf("index recreated composer_turns: %w", err)
	}
	recordRepair(tx, "composer_turns", "", 0, 0, "missing table recreated with canonical shape; no rows could have existed")
	return nil
}

// ensureRepairLedger records every destructive-looking schema repair so no
// legitimate row ever disappears silently: either its content is migrated
// into the canonical table or the legacy table name plus row counts stay
// queryable in schema_repair_ledger.
func ensureRepairLedger(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS schema_repair_ledger (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repaired_at TEXT NOT NULL,
		table_name TEXT NOT NULL,
		legacy_table TEXT NOT NULL,
		copied_rows INTEGER NOT NULL DEFAULT 0,
		skipped_rows INTEGER NOT NULL DEFAULT 0,
		note TEXT NOT NULL DEFAULT ''
	)`)
	return err
}

func recordRepair(tx *sql.Tx, table, legacy string, copied, skipped int64, note string) {
	_, _ = tx.Exec(`INSERT INTO schema_repair_ledger(repaired_at,table_name,legacy_table,copied_rows,skipped_rows,note) VALUES(datetime('now'),?,?,?,?,?)`,
		table, legacy, copied, skipped, note)
}

func legacyTableCount(tx *sql.Tx, legacy string) int64 {
	var n int64
	if err := tx.QueryRow(`SELECT COUNT(*) FROM ` + quoteIdent(legacy)).Scan(&n); err != nil {
		return -1
	}
	return n
}

var flowEvidenceColumns = map[string][]string{
	"flow_context_capsules": {"id", "run_id", "step_id", "flow_revision", "content_json", "created_at", "updated_at"},
	"flow_work_receipts":    {"id", "run_id", "step_id", "status", "content_json", "created_at", "updated_at"},
}

func (s *Store) repairFlowEvidenceSchema() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin schema repair: %w", err)
	}
	defer tx.Rollback()
	if err := ensureRepairLedger(tx); err != nil {
		return err
	}
	for table, required := range flowEvidenceColumns {
		cols, exists, err := tableColumns(tx, table)
		if err != nil {
			return err
		}
		if exists && hasColumns(cols, required) {
			continue
		}
		if exists {
			legacy := table + "_legacy_" + time.Now().UTC().Format("20060102150405")
			if _, err := tx.Exec(`ALTER TABLE ` + quoteIdent(table) + ` RENAME TO ` + quoteIdent(legacy)); err != nil {
				return fmt.Errorf("preserve legacy %s: %w", table, err)
			}
		}
		if err := createFlowEvidenceTable(tx, table); err != nil {
			return err
		}
		// Migrate content back from EVERY legacy table, oldest first, also
		// when the canonical table was wholly missing: accumulated legacy
		// rows must never become query-invisible. Each migration is recorded
		// in schema_repair_ledger; skipped rows stay in their legacy table.
		if err := migrateAllLegacyFlowEvidence(tx, table); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_flow_context_capsules_run ON flow_context_capsules(run_id, step_id)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_flow_work_receipts_run ON flow_work_receipts(run_id, step_id)`); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit schema repair: %w", err)
	}
	return nil
}

func tableColumns(tx *sql.Tx, table string) (map[string]bool, bool, error) {
	rows, err := tx.Query(`PRAGMA table_info(` + quoteIdent(table) + `)`)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	cols := map[string]bool{}
	var cid int
	var name, typ string
	var notnull, pk int
	var dflt any
	for rows.Next() {
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return nil, false, err
		}
		cols[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return cols, len(cols) > 0, nil
}

func hasColumns(cols map[string]bool, required []string) bool {
	for _, c := range required {
		if !cols[c] {
			return false
		}
	}
	return true
}

func quoteIdent(v string) string { return `"` + strings.ReplaceAll(v, `"`, `""`) + `"` }
func createFlowEvidenceTable(tx *sql.Tx, table string) error {
	stmt := `CREATE TABLE IF NOT EXISTS ` + quoteIdent(table) + ` (id TEXT PRIMARY KEY, run_id TEXT NOT NULL REFERENCES mission_runs(id) ON DELETE CASCADE, step_id TEXT NOT NULL, `
	if table == "flow_context_capsules" {
		stmt += `flow_revision INTEGER NOT NULL, `
	} else {
		stmt += `status TEXT NOT NULL, `
	}
	stmt += `content_json TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE(run_id, step_id))`
	_, err := tx.Exec(stmt)
	return err
}

// migrateAllLegacyFlowEvidence copies rows from every legacy table of the
// given canonical table (oldest first) back into the canonical shape.
// Tolerant column mapping covers historical variants (payload_json,
// receipt_json, capsule_json); fabricated defaults are limited to metadata
// the canonical schema requires, and every run is ledger-recorded.
func migrateAllLegacyFlowEvidence(tx *sql.Tx, table string) error {
	rows, err := tx.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name LIKE ? ORDER BY name ASC`, table+"_legacy_%")
	if err != nil {
		return err
	}
	var legacies []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		legacies = append(legacies, name)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, legacy := range legacies {
		copied, skipped, note := copyLegacyFlowEvidence(tx, table, legacy)
		recordRepair(tx, table, legacy, copied, skipped, note)
	}
	return nil
}

func copyLegacyFlowEvidence(tx *sql.Tx, table, legacy string) (copied, skipped int64, note string) {
	before := legacyTableCount(tx, legacy)
	lcols, _, _ := tableColumns(tx, legacy)
	if !hasColumns(lcols, []string{"id", "run_id", "step_id"}) {
		return 0, before, "legacy table lacks id/run_id/step_id; kept as-is for manual inspection"
	}
	content := "''"
	for _, c := range []string{"content_json", "payload_json", "receipt_json", "capsule_json"} {
		if lcols[c] {
			content = quoteIdent(c)
			break
		}
	}
	created := "datetime('now')"
	if lcols["created_at"] {
		created = quoteIdent("created_at")
	}
	updated := created
	if lcols["updated_at"] {
		updated = quoteIdent("updated_at")
	}
	var res sql.Result
	var err error
	if table == "flow_context_capsules" {
		revision := "0"
		if lcols["flow_revision"] {
			revision = quoteIdent("flow_revision")
		}
		res, err = tx.Exec(`INSERT OR IGNORE INTO flow_context_capsules(id,run_id,step_id,flow_revision,content_json,created_at,updated_at) SELECT id,run_id,step_id,` + revision + `,` + content + `,` + created + `,` + updated + ` FROM ` + quoteIdent(legacy))
	} else {
		status := "'UNKNOWN'"
		if lcols["status"] {
			status = quoteIdent("status")
		}
		res, err = tx.Exec(`INSERT OR IGNORE INTO flow_work_receipts(id,run_id,step_id,status,content_json,created_at,updated_at) SELECT id,run_id,step_id,` + status + `,` + content + `,` + created + `,` + updated + ` FROM ` + quoteIdent(legacy))
	}
	if err != nil {
		return 0, before, "copy failed (" + err.Error() + "); rows kept in legacy table"
	}
	copied, _ = res.RowsAffected()
	skipped = before - copied
	note = "migrated with tolerant column mapping"
	if skipped > 0 {
		note += "; skipped rows remain in legacy table (id collision or constraint)"
	}
	return copied, skipped, note
}

// SchemaVersion returns the highest applied migration version.
func (s *Store) SchemaVersion() (int, error) {
	var v int
	err := s.db.QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&v)
	return v, err
}
