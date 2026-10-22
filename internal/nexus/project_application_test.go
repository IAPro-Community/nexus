package nexus

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectApplicationServiceCRUDPreservesCoreRules(t *testing.T) {
	n := openTestNexus(t)
	service := NewProjectApplicationService(n)
	project, err := service.Create(context.Background(), "Demo", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	name := "Renamed"
	updated, err := service.Update(context.Background(), project.ID, ProjectPatch{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != name || updated.Slug != "renamed" {
		t.Fatalf("project patch was not normalized: %#v", updated)
	}

	listed, err := service.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != project.ID {
		t.Fatalf("unexpected project list: %#v", listed)
	}
}

func TestProjectApplicationServiceRejectsCanceledContext(t *testing.T) {
	n := openTestNexus(t)
	service := NewProjectApplicationService(n)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := service.List(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("list error = %v, want context canceled", err)
	}
}

func TestProjectApplicationServiceRejectsInvalidDirectories(t *testing.T) {
	n := openTestNexus(t)
	service := NewProjectApplicationService(n)
	root := t.TempDir()
	missing := filepath.Join(root, "not-created")
	if _, err := service.Create(context.Background(), "Missing", missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing directory error = %v, want os.ErrNotExist", err)
	}
	file := filepath.Join(root, "regular-file.txt")
	if err := os.WriteFile(file, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(context.Background(), "File", file); err == nil {
		t.Fatal("project registration accepted a regular file as a directory")
	}
	projects, err := service.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 0 {
		t.Fatalf("invalid paths persisted project records: %+v", projects)
	}
}
