package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func openGateStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "nexus.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, dir
}

func gateProject(t *testing.T, s *Store) string {
	t.Helper()
	project, err := s.CreateProject(Project{Name: "Gate", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	return project.ID
}

func closeGateManually(t *testing.T, dir string) {
	t.Helper()
	state := `{"schemaVersion":1,"phase":"MIGRATED","migratedAt":"2026-01-01T00:00:00Z","sourceDigestSHA256":"` +
		strings.Repeat("a", 64) + `"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "memory-migration-state.json"), []byte(state), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyMemoryGateBlocksWritersAfterMigration(t *testing.T) {
	s, _ := openGateStore(t)
	projectID := gateProject(t, s)
	if _, err := s.UpsertIntelligenceMemory(IntelligenceMemory{ProjectID: projectID, Key: "quality.testing", Value: "run go test", Source: "explicit"}); err != nil {
		t.Fatal(err)
	}
	closeGateManually(t, s.memoryFenceDir)

	if _, err := s.UpsertIntelligenceMemory(IntelligenceMemory{ProjectID: projectID, Key: "quality.testing", Value: "changed"}); !errors.Is(err, ErrLegacyMemoryWritesDisabled) {
		t.Fatalf("upsert must be disabled after migration, got %v", err)
	}
	if err := s.DeleteIntelligenceMemory(projectID, "whatever"); !errors.Is(err, ErrLegacyMemoryWritesDisabled) {
		t.Fatalf("delete must be disabled after migration, got %v", err)
	}
	if err := s.DeleteIntelligenceMemoryByKey(projectID, "quality.testing"); !errors.Is(err, ErrLegacyMemoryWritesDisabled) {
		t.Fatalf("delete by key must be disabled after migration, got %v", err)
	}
	got, err := s.GetIntelligenceMemoryByKey(projectID, "quality.testing")
	if err != nil || got == nil || got.Value != "run go test" {
		t.Fatalf("reads must stay allowed after migration: %v, %+v", err, got)
	}
	rows, err := s.ListIntelligenceMemories(projectID)
	if err != nil || len(rows) != 1 {
		t.Fatalf("list must stay allowed after migration: %v, %d rows", err, len(rows))
	}
}

func TestLegacyMemoryGateMalformedStateFailsClosed(t *testing.T) {
	s, dir := openGateStore(t)
	projectID := gateProject(t, s)
	if err := os.WriteFile(filepath.Join(dir, "memory-migration-state.json"), []byte("{not json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpsertIntelligenceMemory(IntelligenceMemory{ProjectID: projectID, Key: "k", Value: "v"}); err == nil || errors.Is(err, os.ErrNotExist) {
		t.Fatalf("malformed gate must fail closed, got %v", err)
	}
}

func TestLegacyMemoryGateCloseRequiresActiveJournal(t *testing.T) {
	s, dir := openGateStore(t)
	opID := strings.Repeat("b", 32)
	digest := strings.Repeat("c", 64)

	if err := s.CloseLegacyMemoryGate(opID, digest); err == nil {
		t.Fatal("gate must refuse to close without an active recovery journal lock")
	}
	root := filepath.Join(dir, "memory-recovery-journal")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".active.lock"), []byte(strings.Repeat("d", 32)), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.CloseLegacyMemoryGate(opID, digest); err == nil {
		t.Fatal("gate must refuse to close under a different journal operation")
	}
	if err := os.WriteFile(filepath.Join(root, ".active.lock"), []byte(opID), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.CloseLegacyMemoryGate(opID, digest); err != nil {
		t.Fatalf("gate close with matching lock refused: %v", err)
	}
	if err := s.CloseLegacyMemoryGate(opID, digest); err == nil {
		t.Fatal("gate must refuse a second close")
	}
	if err := s.CloseLegacyMemoryGate("short", digest); err == nil {
		t.Fatal("gate must refuse a malformed operation id")
	}
	if err := s.CloseLegacyMemoryGate(opID, "not-hex"); err == nil {
		t.Fatal("gate must refuse a malformed digest")
	}
}

func TestLegacyMemoryGateReopenRestoresWriters(t *testing.T) {
	s, _ := openGateStore(t)
	projectID := gateProject(t, s)
	if err := s.ReopenLegacyMemoryGate(); err != nil {
		t.Fatalf("reopen without a gate must be a no-op, got %v", err)
	}
	closeGateManually(t, s.memoryFenceDir)
	if err := s.ReopenLegacyMemoryGate(); err != nil {
		t.Fatalf("reopen refused: %v", err)
	}
	if _, err := s.UpsertIntelligenceMemory(IntelligenceMemory{ProjectID: projectID, Key: "quality.testing", Value: "run go test"}); err != nil {
		t.Fatalf("writers must work again after reopen: %v", err)
	}
}

func TestLegacyMemoryGateReopenRefusedWithActiveJournal(t *testing.T) {
	s, dir := openGateStore(t)
	closeGateManually(t, s.memoryFenceDir)
	root := filepath.Join(dir, "memory-recovery-journal")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".active.lock"), []byte(strings.Repeat("e", 32)), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.ReopenLegacyMemoryGate(); err == nil {
		t.Fatal("reopen must be refused while a recovery journal lock exists")
	}
}
