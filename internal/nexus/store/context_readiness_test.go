package store

import (
	"path/filepath"
	"testing"
)

func TestContextReadinessPersistsAndReloads(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nexus.db")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.CreateProject(Project{ID: "project-context", Name: "Context", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	fingerprint := map[string]string{"project_id": p.ID, "branch": "main", "head": "abc"}
	if _, err := s.SaveContextReadiness(p.ID, "READY", fingerprint, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	record, err := s.GetContextReadiness(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.State != "READY" || record.ProjectID != p.ID {
		t.Fatalf("reloaded record=%#v", record)
	}
}
