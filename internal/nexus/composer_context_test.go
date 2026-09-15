package nexus

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestAttachComposerContextFileWithinProject(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Title\nHello context\n"), 0600); err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Ctx", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	view, err := n.CreateComposerSession(context.Background(), project.ID, "Add a feature to the dashboard")
	if err != nil {
		t.Fatal(err)
	}

	updated, err := n.AttachComposerContextFile(context.Background(), view.Session.ID, "README.md", view.Session.Revision)
	if err != nil {
		t.Fatalf("attach within project failed: %v", err)
	}
	refs := updated.Brief.Context.ReferencedFiles
	if len(refs) != 1 || refs[0].Path != "README.md" {
		t.Fatalf("expected README.md referenced file, got %+v", refs)
	}
	if refs[0].Hash == "" || !strings.Contains(refs[0].Excerpt, "Hello context") {
		t.Fatalf("expected hash + excerpt, got %+v", refs[0])
	}
	if !strings.Contains(updated.Brief.Preview, "Context — Referenced files") ||
		!strings.Contains(updated.Brief.Preview, "README.md") {
		t.Fatalf("expected preview to list referenced file, got:\n%s", updated.Brief.Preview)
	}
}

func TestAttachComposerContextFileRejectsEscape(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	project, err := st.CreateProject(store.Project{Name: "Ctx escape", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	// Secret lives outside the project root.
	outside := filepath.Join(filepath.Dir(root), "secret.txt")
	if err := os.WriteFile(outside, []byte("top secret"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(outside) })

	view, err := n.CreateComposerSession(context.Background(), project.ID, "Explore secrets")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := n.AttachComposerContextFile(context.Background(), view.Session.ID, "../secret.txt", view.Session.Revision); err == nil {
		t.Fatal("expected path escape to be rejected")
	}
}

func TestDetachComposerContextFile(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("keep this"), 0600); err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Ctx detach", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	view, err := n.CreateComposerSession(context.Background(), project.ID, "Attach then detach")
	if err != nil {
		t.Fatal(err)
	}
	attached, err := n.AttachComposerContextFile(context.Background(), view.Session.ID, "notes.txt", view.Session.Revision)
	if err != nil {
		t.Fatal(err)
	}
	detached, err := n.DetachComposerContextFile(context.Background(), view.Session.ID, "notes.txt", attached.Session.Revision)
	if err != nil {
		t.Fatalf("detach failed: %v", err)
	}
	if len(detached.Brief.Context.ReferencedFiles) != 0 {
		t.Fatalf("expected no referenced files after detach, got %+v", detached.Brief.Context.ReferencedFiles)
	}
}
