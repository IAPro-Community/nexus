package store

import (
	"database/sql"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// FIX_EXISTING (0021 fundida): exatamente UM arquivo 0021. Antes da fusão
// existiam dois (0021_conversation_events.sql + 0021_capacity_monitor.sql) com
// a mesma versão 21 — o runner marca a versão após o primeiro arquivo e o
// segundo nunca era aplicado (só o repair mascarava a metade conversation).
func TestSingle0021MigrationFile(t *testing.T) {
	entries, err := fs.Glob(migrationsFS, "migrations/0021_*.sql")
	if err != nil {
		t.Fatalf("glob 0021: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("0021 files = %d (%v), want exactly 1 (fusão, não renumerar)", len(entries), entries)
	}
}

// A 0021 fundida deve ser idempotente em SQL puro: aplicada duas vezes sobre
// a mesma base não pode falhar (CREATE IF NOT EXISTS; sem ADD COLUMN nu —
// as colunas kind/payload_json são de responsabilidade do repair idempotente
// repairComposerConversationSchema, que roda a cada Open).
func TestMerged0021AppliesTwice(t *testing.T) {
	entries, err := fs.Glob(migrationsFS, "migrations/0021_*.sql")
	if err != nil {
		t.Fatalf("glob 0021: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("precondition: single 0021 file, got %v", entries)
	}
	body, err := migrationsFS.ReadFile(entries[0])
	if err != nil {
		t.Fatalf("read merged 0021: %v", err)
	}
	// Base mínima: abre o store (traz 0001..0020 + repair) e remove o
	// registro 0021 para re-aplicar o arquivo manualmente duas vezes.
	path := filepath.Join(t.TempDir(), "nexus.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if _, err := s.db.Exec(`DELETE FROM schema_migrations WHERE version = 21`); err != nil {
		t.Fatalf("unrecord 21: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("raw open: %v", err)
	}
	defer raw.Close()
	for i := 0; i < 2; i++ {
		for _, stmt := range splitSQLStatements(string(body)) {
			if _, err := raw.Exec(stmt); err != nil {
				t.Fatalf("apply #%d failed: %v\nstmt: %.120s", i+1, err, stmt)
			}
		}
	}
}

// Migração do zero: fresh Open deve expor ambas as metades (capacity +
// conversation) — hoje passa via repair; após a fusão deve passar via
// migração + repair.
func TestMerged0021FreshCoversBothHalves(t *testing.T) {
	s := openTestStore(t)
	for _, table := range []string{
		"capacity_observations", "capacity_states", "capacity_monitor_lease",
		"capacity_events", "capacity_delivery_receipts", "capacity_recommendations",
		"capacity_preferences", "conversation_attachments",
	} {
		var name string
		if err := s.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name); err != nil {
			t.Fatalf("table %s missing: %v", table, err)
		}
	}
	for _, col := range []string{"kind", "payload_json"} {
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('composer_turns') WHERE name=?`, col).Scan(&n); err != nil || n != 1 {
			t.Fatalf("composer_turns.%s missing: count=%d err=%v", col, n, err)
		}
	}
	v, err := s.SchemaVersion()
	if err != nil {
		t.Fatalf("schema version: %v", err)
	}
	migrationFiles, globErr := fs.Glob(migrationsFS, "migrations/[0-9][0-9][0-9][0-9]_*.sql")
	if globErr != nil {
		t.Fatalf("glob migrations: %v", globErr)
	}
	latest := 0
	for _, name := range migrationFiles {
		base := strings.TrimPrefix(name, "migrations/")
		versionText := strings.SplitN(base, "_", 2)[0]
		version, parseErr := strconv.Atoi(versionText)
		if parseErr != nil {
			t.Fatalf("parse migration version from %q: %v", name, parseErr)
		}
		if version > latest {
			latest = version
		}
	}
	if v != latest {
		t.Fatalf("schema version = %d, want latest embedded migration %d", v, latest)
	}
}

// A partir da metade capacity: sem registro 21 e sem conversation_attachments,
// o Open deve restaurar a metade conversation (migração + repair).
func TestMerged0021FromCapacityHalf(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nexus.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := s.db.Exec(`DELETE FROM schema_migrations WHERE version = 21`); err != nil {
		t.Fatalf("unrecord 21: %v", err)
	}
	if _, err := s.db.Exec(`DROP TABLE IF EXISTS conversation_attachments`); err != nil {
		t.Fatalf("drop conversation half: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen from capacity half: %v", err)
	}
	defer s2.Close()
	var name string
	if err := s2.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='conversation_attachments'`).Scan(&name); err != nil {
		t.Fatalf("conversation half not restored: %v", err)
	}
	var capName string
	if err := s2.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='capacity_observations'`).Scan(&capName); err != nil {
		t.Fatalf("capacity half lost: %v", err)
	}
}

// A partir da metade conversation: sem registro 21 e sem tabelas capacity,
// o Open deve restaurar a metade capacity.
func TestMerged0021FromConversationHalf(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nexus.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := s.db.Exec(`DELETE FROM schema_migrations WHERE version = 21`); err != nil {
		t.Fatalf("unrecord 21: %v", err)
	}
	for _, table := range []string{
		"capacity_observations", "capacity_states", "capacity_monitor_lease",
		"capacity_events", "capacity_delivery_receipts", "capacity_recommendations",
		"capacity_preferences",
	} {
		if _, err := s.db.Exec(`DROP TABLE IF EXISTS ` + table); err != nil {
			t.Fatalf("drop %s: %v", table, err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen from conversation half: %v", err)
	}
	defer s2.Close()
	for _, table := range []string{
		"capacity_observations", "capacity_states", "capacity_monitor_lease",
		"capacity_events", "capacity_delivery_receipts", "capacity_recommendations",
		"capacity_preferences", "conversation_attachments",
	} {
		var name string
		if err := s2.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name); err != nil {
			t.Fatalf("table %s not restored from conversation half: %v", table, err)
		}
	}
	if !strings.Contains(t.Name(), "ConversationHalf") {
		t.Fatal("unreachable")
	}
}
