package nexus

import (
	"context"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestComposerApplicationServiceSessionWorkflow(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Composer App", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	service := NewComposerApplicationService(n)

	created, err := service.Create(context.Background(), project.ID, "Plan a feature", "Use the existing prompt", "EXISTING_PROMPT")
	if err != nil {
		t.Fatal(err)
	}
	if created.Session.ProjectID != project.ID {
		t.Fatalf("created session project = %q, want %q", created.Session.ProjectID, project.ID)
	}

	listed, err := service.List(context.Background(), project.ID)
	if err != nil || len(listed) != 1 {
		t.Fatalf("listed sessions = %#v, err = %v", listed, err)
	}
	fetched, err := service.Get(context.Background(), created.Session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.Session.ID != created.Session.ID {
		t.Fatalf("fetched session = %q, want %q", fetched.Session.ID, created.Session.ID)
	}
}
