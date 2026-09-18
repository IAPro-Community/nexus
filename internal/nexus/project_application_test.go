package nexus

import (
	"context"
	"errors"
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
