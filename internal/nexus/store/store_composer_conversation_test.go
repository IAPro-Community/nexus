package store

import (
	"path/filepath"
	"testing"
)

func TestRepairComposerConversationSchemaAddsMissingColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "drift-composer.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// Simulate the broken state: version recorded, columns stripped / never applied.
	if _, err := s.DB().Exec(`CREATE TABLE composer_turns_legacy AS SELECT id,session_id,sequence,role,content,created_at FROM composer_turns`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB().Exec(`DROP TABLE composer_turns`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB().Exec(`ALTER TABLE composer_turns_legacy RENAME TO composer_turns`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB().Exec(`DROP TABLE IF EXISTS conversation_attachments`); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen after drift: %v", err)
	}
	defer s2.Close()
	var kindCount, payloadCount, tableCount int
	if err := s2.DB().QueryRow(`SELECT COUNT(*) FROM pragma_table_info('composer_turns') WHERE name='kind'`).Scan(&kindCount); err != nil {
		t.Fatal(err)
	}
	if err := s2.DB().QueryRow(`SELECT COUNT(*) FROM pragma_table_info('composer_turns') WHERE name='payload_json'`).Scan(&payloadCount); err != nil {
		t.Fatal(err)
	}
	if err := s2.DB().QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='conversation_attachments'`).Scan(&tableCount); err != nil {
		t.Fatal(err)
	}
	if kindCount != 1 || payloadCount != 1 || tableCount != 1 {
		t.Fatalf("repair incomplete: kind=%d payload=%d attachments=%d", kindCount, payloadCount, tableCount)
	}
}

func TestSplitSQLStatementsRunsAllClauses(t *testing.T) {
	stmts := splitSQLStatements("ALTER TABLE t ADD COLUMN a TEXT;\nALTER TABLE t ADD COLUMN b TEXT;\n")
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %#v", stmts)
	}
}
