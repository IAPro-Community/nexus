package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func openUpgradeParent(t *testing.T, path string) (projectID, planID string) {
	t.Helper()
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	project, err := st.CreateProject(Project{Name: "Upgrade", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(WorkPlan{ProjectID: project.ID, Title: "Flow"})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertMissionRun(MissionRunRecord{ID: "run-1", PlanID: plan.ID, ProjectID: project.ID, State: "EXECUTING", PayloadJSON: `{}`}); err != nil {
		t.Fatal(err)
	}
	return project.ID, plan.ID
}

func ledgerEntry(t *testing.T, st *Store, table string) (copied, skipped int64, note string) {
	t.Helper()
	err := st.DB().QueryRow(`SELECT copied_rows, skipped_rows, note FROM schema_repair_ledger WHERE table_name=? ORDER BY id DESC LIMIT 1`, table).Scan(&copied, &skipped, &note)
	if err != nil {
		t.Fatalf("ledger entry for %s: %v", table, err)
	}
	return copied, skipped, note
}

// TestLegacyCapsulesAreMigratedWithLedger proves pre-0008 capsule rows stay
// queryable after upgrade instead of hiding inside a renamed legacy table.
func TestLegacyCapsulesAreMigratedWithLedger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "upgrade.db")
	openUpgradeParent(t, path)

	fixture, err := os.ReadFile(filepath.Join("testdata", "upgrades", "0008_capsules_legacy.sql"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB().Exec(string(fixture)); err != nil {
		t.Fatal(err)
	}
	seeds := []struct{ id, step, payload string }{
		{"capsule-old-1", "A", `{"legacy":1}`},
		{"capsule-old-2", "B", `{"legacy":2}`},
	}
	for _, s := range seeds {
		if _, err := st.DB().Exec(`INSERT INTO flow_context_capsules(id,run_id,step_id,payload_json,created_at) VALUES(?,?,?,?,'2026-01-01')`, s.id, "run-1", s.step, s.payload); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	for id, want := range map[string]string{"capsule-old-1": `{"legacy":1}`, "capsule-old-2": `{"legacy":2}`} {
		var content string
		if err := st.DB().QueryRow(`SELECT content_json FROM flow_context_capsules WHERE id=?`, id).Scan(&content); err != nil {
			t.Fatalf("capsule %s invisible after upgrade: %v", id, err)
		}
		if content != want {
			t.Fatalf("capsule %s content=%s, want %s", id, content, want)
		}
	}
	copied, skipped, _ := ledgerEntry(t, st, "flow_context_capsules")
	if copied != 2 || skipped != 0 {
		t.Fatalf("ledger copied=%d skipped=%d, want 2/0", copied, skipped)
	}
}

// TestAllLegacyTablesAreMigratedOldestFirst proves accumulated legacy tables
// are all drained, not just the newest one.
func TestAllLegacyTablesAreMigratedOldestFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "multilegacy.db")
	openUpgradeParent(t, path)
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB().Exec(`DROP TABLE flow_work_receipts`); err != nil {
		t.Fatal(err)
	}
	for _, ddl := range []string{
		`CREATE TABLE flow_work_receipts_legacy_20200101000000 (id TEXT PRIMARY KEY, run_id TEXT, step_id TEXT, status TEXT, payload_json TEXT, created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE flow_work_receipts_legacy_20200202000000 (id TEXT PRIMARY KEY, run_id TEXT, step_id TEXT, status TEXT, payload_json TEXT, created_at TEXT, updated_at TEXT)`,
	} {
		if _, err := st.DB().Exec(ddl); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.DB().Exec(`INSERT INTO flow_work_receipts_legacy_20200101000000 VALUES ('old-1','run-1','A','VERIFIED','{"v":1}','2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB().Exec(`INSERT INTO flow_work_receipts_legacy_20200202000000 VALUES ('old-2','run-1','B','VERIFIED','{"v":2}','2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	for id, want := range map[string]string{"old-1": `{"v":1}`, "old-2": `{"v":2}`} {
		var content string
		if err := st.DB().QueryRow(`SELECT content_json FROM flow_work_receipts WHERE id=?`, id).Scan(&content); err != nil {
			t.Fatalf("receipt %s invisible after upgrade: %v", id, err)
		}
		if content != want {
			t.Fatalf("receipt %s content=%s, want %s", id, content, want)
		}
	}
}

// TestIncompatibleAttachmentsTableIsPreservedWithLedger proves a wrong-shape
// attachments table is kept as legacy (plus ledger) instead of dropped.
func TestIncompatibleAttachmentsTableIsPreservedWithLedger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "attshape.db")
	openUpgradeParent(t, path)
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB().Exec(`DROP TABLE conversation_attachments`); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB().Exec(`CREATE TABLE conversation_attachments (id TEXT PRIMARY KEY, session_id TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB().Exec(`INSERT INTO conversation_attachments VALUES ('att_kept','cmp_x')`); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	var kept string
	var legacyName string
	err = st.DB().QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'conversation_attachments_legacy_%'`).Scan(&legacyName)
	if err != nil {
		t.Fatalf("legacy attachments table missing: %v", err)
	}
	if err := st.DB().QueryRow(`SELECT id FROM ` + quoteIdent(legacyName) + ` WHERE id='att_kept'`).Scan(&kept); err != nil {
		t.Fatalf("legacy row not preserved: %v", err)
	}
	copied, skipped, _ := ledgerEntry(t, st, "conversation_attachments")
	if copied != 0 || skipped != 1 {
		t.Fatalf("ledger copied=%d skipped=%d, want 0/1", copied, skipped)
	}
	var cols int
	if err := st.DB().QueryRow(`SELECT COUNT(*) FROM pragma_table_info('conversation_attachments')`).Scan(&cols); err != nil {
		t.Fatal(err)
	}
	if cols != 9 {
		t.Fatalf("canonical attachments table has %d cols, want 9", cols)
	}
}

// TestMissingComposerTurnsRecreated proves a wholly missing turns table is
// rebuilt with the full canonical shape including conversation columns.
func TestMissingComposerTurnsRecreated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "turns.db")
	openUpgradeParent(t, path)
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB().Exec(`DROP TABLE composer_turns`); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cols := map[string]bool{}
	rows, err := st.DB().Query(`SELECT name FROM pragma_table_info('composer_turns')`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		cols[name] = true
	}
	rows.Close()
	for _, c := range []string{"id", "session_id", "sequence", "role", "content", "created_at", "kind", "payload_json"} {
		if !cols[c] {
			t.Fatalf("recreated composer_turns lacks %s", c)
		}
	}
}

// TestTokenBarUsageSchemaRecoversMissingTables covers upgrades from a partial
// backup that retained version 30 but accidentally omitted new read-only tables.
// Reopen repairs those structures without resetting canonical cost history.
func TestTokenBarUsageSchemaRecoversMissingTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "partial-tokenbar.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	msg := ExternalClaudeMessage{
		MessageHash: "preserved-usage",
		Model:       "claude-test",
		InputTokens: 42,
		ObservedAt:  time.Now().UTC(),
	}
	if err := st.SaveExternalClaudeBatch(ctx, "LOCAL_DEFAULT_UNVERIFIED", "old-file", 120,
		[]ExternalClaudeMessage{msg}); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"capacity_pace_samples", "external_claude_offsets"} {
		if _, err := st.DB().Exec("DROP TABLE " + table); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = Open(path)
	if err != nil {
		t.Fatalf("reopen after partial restore: %v", err)
	}
	defer st.Close()
	version, err := st.SchemaVersion()
	if err != nil || version != 30 {
		t.Fatalf("schema version = %d, %v, want 30", version, err)
	}
	for _, table := range []string{"capacity_pace_samples", "external_claude_messages", "external_claude_offsets"} {
		var count int
		if err := st.DB().QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatalf("repaired table %s unavailable: %v", table, err)
		}
	}
	for _, index := range []string{"idx_capacity_pace_recent", "idx_external_claude_time"} {
		var count int
		if err := st.DB().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", index).Scan(&count); err != nil || count != 1 {
			t.Fatalf("restored index %s: count=%d err=%v", index, count, err)
		}
	}
	summary, err := st.SummarizeExternalClaude(ctx, 7)
	if err != nil || summary.Messages != 1 || summary.InputTokens != 42 {
		t.Fatalf("existing user-consented usage lost during repair: %+v, %v", summary, err)
	}
	if err := st.SaveExternalClaudeBatch(ctx, "LOCAL_DEFAULT_UNVERIFIED", "new-file", 10, nil); err != nil {
		t.Fatalf("restored offsets unusable: %v", err)
	}
}
