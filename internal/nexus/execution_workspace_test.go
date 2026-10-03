package nexus

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func newExecutionWorkspaceTestRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	repo := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "init")
	return repo
}

func TestResolveTaskExecutionWorkspaceDefaultsOff(t *testing.T) {
	n := &Nexus{}
	project := store.Project{ID: "project-1", CanonicalPath: newExecutionWorkspaceTestRepo(t)}
	_, err := n.ResolveTaskExecutionWorkspace(context.Background(), project, ExecutionWorkspaceRequest{ID: "task-1"})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("v1.1 workspaces must fail closed while flag is off, got %v", err)
	}
}

func TestResolveTaskExecutionWorkspaceCreatesIndependentWorktrees(t *testing.T) {
	repo := newExecutionWorkspaceTestRepo(t)
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("NEXUS_ENABLE_EXECUTION_WORKSPACES_V2", "1")

	n := &Nexus{}
	project := store.Project{ID: "project-1", CanonicalPath: repo}
	first, err := n.ResolveTaskExecutionWorkspace(context.Background(), project, ExecutionWorkspaceRequest{ID: "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := n.ResolveTaskExecutionWorkspace(context.Background(), project, ExecutionWorkspaceRequest{ID: "candidate-b"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Path == second.Path || first.Path == repo || second.Path == repo {
		t.Fatalf("task workspaces must be isolated: first=%q second=%q repo=%q", first.Path, second.Path, repo)
	}
	root := executionWorkspaceManagedRoot(dataDir, project.ID)
	if !pathWithinRoot(first.Path, root) || !pathWithinRoot(second.Path, root) {
		t.Fatalf("workspaces escaped managed root %q", root)
	}
	if first.Branch != "iapro/workspace/candidate-a" || second.Branch != "iapro/workspace/candidate-b" {
		t.Fatalf("unexpected branches: %q %q", first.Branch, second.Branch)
	}
	reused, err := n.ResolveTaskExecutionWorkspace(context.Background(), project, ExecutionWorkspaceRequest{ID: "candidate-a"})
	if err != nil || reused.Path != first.Path {
		t.Fatalf("workspace reuse path=%q err=%v want=%q", reused.Path, err, first.Path)
	}
}

func TestResolveTaskExecutionWorkspaceRejectsTraversal(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("NEXUS_ENABLE_EXECUTION_WORKSPACES_V2", "1")
	n := &Nexus{}
	project := store.Project{ID: "project-1", CanonicalPath: newExecutionWorkspaceTestRepo(t)}
	_, err := n.ResolveTaskExecutionWorkspace(context.Background(), project, ExecutionWorkspaceRequest{ID: "../escape"})
	if err == nil || !strings.Contains(err.Error(), "unsafe managed") {
		t.Fatalf("unsafe workspace id must fail closed, got %v", err)
	}
}

func TestTaskExecutionWorkspaceIDIsStableAndFilesystemSafe(t *testing.T) {
	first := taskExecutionWorkspaceID("run/unsafe", "pkg/unsafe")
	second := taskExecutionWorkspaceID("run/unsafe", "pkg/unsafe")
	other := taskExecutionWorkspaceID("run/unsafe", "pkg/other")
	if first != second {
		t.Fatalf("workspace id must be stable: %q != %q", first, second)
	}
	if first == other {
		t.Fatal("different package identities must not share a workspace id")
	}
	if strings.ContainsAny(first, `/\\`) {
		t.Fatalf("workspace id must be path-safe: %q", first)
	}
}
