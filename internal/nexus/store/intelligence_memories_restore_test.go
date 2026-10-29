package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func restoreTestStore(t *testing.T) (*Store, string, string) {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "nexus.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	p1, err := s.CreateProject(Project{Name: "Restore A", CanonicalPath: t.TempDir()})
	if err != nil { t.Fatal(err) }
	p2, err := s.CreateProject(Project{Name: "Restore B", CanonicalPath: t.TempDir()})
	if err != nil { t.Fatal(err) }
	return s, p1.ID, p2.ID
}

func TestRestoreProjectIntelligenceMemoriesPreservesExactValuesAndOtherProjects(t *testing.T) {
	s, projectA, projectB := restoreTestStore(t)
	ctx := context.Background()
	original, err := s.UpsertIntelligenceMemory(IntelligenceMemory{
		ProjectID: projectA, Key: "quality.testing", Value: "go test ./...", Source: "composer",
	})
	if err != nil { t.Fatal(err) }
	pending, err := s.UpsertIntelligenceMemory(IntelligenceMemory{
		ProjectID: projectA, Key: "pending.quality.verification", Value: "not promoted", Source: "composer",
	})
	if err != nil { t.Fatal(err) }
	other, err := s.UpsertIntelligenceMemory(IntelligenceMemory{
		ProjectID: projectB, Key: "environment.stack", Value: "Go", Source: "user",
	})
	if err != nil { t.Fatal(err) }
	before, err := s.ListIntelligenceMemories(projectA)
	if err != nil { t.Fatal(err) }
	target := []IntelligenceMemory{*original, *pending}
	target[0].Value = "go test -race ./..."
	target[0].UpdatedAt = time.Date(2024, 1, 2, 3, 4, 5, 123, time.UTC)
	if err := s.ReplaceProjectIntelligenceMemoriesTransactional(ctx, projectA, before, target); err != nil {
		t.Fatal(err)
	}
	restored, err := s.ListIntelligenceMemories(projectA)
	if err != nil { t.Fatal(err) }
	if !sameRestoreRows(restored, target) || len(restored) != 2 {
		t.Fatalf("restoration changed IDs, timestamps or pending state")
	}
	untouched, err := s.ListIntelligenceMemories(projectB)
	if err != nil { t.Fatal(err) }
	if len(untouched) != 1 || untouched[0].ID != other.ID || untouched[0].Value != other.Value {
		t.Fatal("cross-project restoration leak")
	}
}

func TestRestoreProjectIntelligenceCASRejectsStaleAndMalformedInput(t *testing.T) {
	s, projectID, _ := restoreTestStore(t)
	row, err := s.UpsertIntelligenceMemory(IntelligenceMemory{
		ProjectID: projectID, Key: "quality.testing", Value: "original", Source: "user",
	})
	if err != nil { t.Fatal(err) }
	before := []IntelligenceMemory{*row}
	_, err = s.UpsertIntelligenceMemory(IntelligenceMemory{
		ProjectID: projectID, Key: "quality.testing", Value: "updated concurrently", Source: "user",
	})
	if err != nil { t.Fatal(err) }
	for _, test := range []struct {
		name string
		expected, replacement []IntelligenceMemory
	}{
		{"stale snapshot", before, before},
		{"duplicate keys", nil, []IntelligenceMemory{*row, *row}},
		{"foreign project row", nil, []IntelligenceMemory{{ID: "imem_other", ProjectID: "prj_attacker", Key: "quality.testing", Value: "x", CreatedAt: time.Now(), UpdatedAt: time.Now()}}},
		{"zero timestamps", nil, []IntelligenceMemory{{ID: "imem_time", ProjectID: projectID, Key: "quality.testing", Value: "x"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := s.ReplaceProjectIntelligenceMemoriesTransactional(context.Background(), projectID, test.expected, test.replacement); err == nil {
				t.Fatal("invalid or stale restore accepted")
			}
		})
	}
	rows, err := s.ListIntelligenceMemories(projectID)
	if err != nil { t.Fatal(err) }
	if len(rows) != 1 || rows[0].Value != "updated concurrently" {
		t.Fatal("failed CAS or invalid replacement changed source data")
	}
}

func TestRestoreProjectIntelligenceSQLFailureRollsBackWholeTransaction(t *testing.T) {
	s, projectA, projectB := restoreTestStore(t)
	own, err := s.UpsertIntelligenceMemory(IntelligenceMemory{
		ProjectID: projectA, Key: "quality.testing", Value: "keep existing", Source: "user",
	})
	if err != nil { t.Fatal(err) }
	other, err := s.UpsertIntelligenceMemory(IntelligenceMemory{
		ProjectID: projectB, Key: "quality.testing", Value: "not ours", Source: "user",
	})
	if err != nil { t.Fatal(err) }
	before, err := s.ListIntelligenceMemories(projectA)
	if err != nil { t.Fatal(err) }
	// Force a global primary-key collision after the transactional DELETE.
	colliding := IntelligenceMemory{ID: other.ID, ProjectID: projectA, Key: "different.key",
		Value: "invalid collision", Source: "test", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := s.ReplaceProjectIntelligenceMemoriesTransactional(context.Background(), projectA, before, []IntelligenceMemory{colliding}); err == nil {
		t.Fatal("expected SQLite constraint failure")
	}
	rows, err := s.ListIntelligenceMemories(projectA)
	if err != nil { t.Fatal(err) }
	if len(rows) != 1 || rows[0].ID != own.ID || rows[0].Value != own.Value {
		t.Fatal("SQLite rollback did not preserve original project memory")
	}
}

func TestRestoreProjectIntelligenceWithEmptySnapshotOnlyTouchesSelectedProject(t *testing.T) {
	s, projectA, projectB := restoreTestStore(t)
	_, err := s.UpsertIntelligenceMemory(IntelligenceMemory{
		ProjectID: projectB, Key: "quality.testing", Value: "preserved",
	})
	if err != nil { t.Fatal(err) }
	if err := s.ReplaceProjectIntelligenceMemoriesTransactional(context.Background(), projectA, nil, nil); err != nil {
		t.Fatal(err)
	}
	b, err := s.ListIntelligenceMemories(projectB)
	if err != nil || len(b) != 1 || b[0].Value != "preserved" { t.Fatal("other project modified") }
}
