package nexus

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestResolveExecutionWorkspaceDeveloperMapsToProject(t *testing.T) {
	n := &Nexus{}
	dir := t.TempDir()
	project := store.Project{ID: "p", CanonicalPath: dir, DefaultIsolation: "developer"}
	agent := store.Agent{ID: "a", ProjectID: "p"}
	got, err := n.resolveExecutionWorkspace(context.Background(), project, agent, AgentConfig{})
	if err != nil || got != dir {
		t.Fatalf("workspace=%q err=%v want %q", got, err, dir)
	}
}

func TestResolveExecutionWorkspaceAgentProjectOverridesWorktreeDefault(t *testing.T) {
	n := &Nexus{}
	dir := t.TempDir()
	project := store.Project{ID: "p", CanonicalPath: dir, DefaultIsolation: "worktree"}
	agent := store.Agent{ID: "a", ProjectID: "p"}
	got, err := n.resolveExecutionWorkspace(context.Background(), project, agent, AgentConfig{Isolation: "project"})
	if err != nil || got != dir {
		t.Fatalf("workspace=%q err=%v want %q", got, err, dir)
	}
}

func TestResolveAutonomousExecutionWorkspaceRejectsAgentProjectOverride(t *testing.T) {
	n := &Nexus{}
	project := store.Project{ID: "p", CanonicalPath: t.TempDir(), DefaultIsolation: "worktree"}
	agent := store.Agent{ID: "a", ProjectID: "p"}

	_, err := n.resolveAutonomousExecutionWorkspace(context.Background(), project, agent, AgentConfig{Isolation: "project"})
	if err == nil {
		t.Fatal("autonomous execution must reject Agent isolation that overrides a worktree project default")
	}
	if !strings.Contains(err.Error(), "worktree") || !strings.Contains(err.Error(), "project") {
		t.Fatalf("error=%q, want effective isolation guidance", err)
	}
}

func TestResolveAutonomousExecutionWorkspaceFailsClosedWithoutWorktree(t *testing.T) {
	n := &Nexus{}
	project := store.Project{ID: "p", CanonicalPath: t.TempDir(), DefaultIsolation: "project"}
	agent := store.Agent{ID: "a", ProjectID: "p"}

	_, err := n.resolveAutonomousExecutionWorkspace(context.Background(), project, agent, AgentConfig{})
	if err == nil {
		t.Fatal("autonomous execution must reject canonical project isolation")
	}
	if !strings.Contains(err.Error(), "Isolation") || !strings.Contains(err.Error(), "worktree") {
		t.Fatalf("error=%q, want actionable worktree isolation guidance", err)
	}
}

func TestResolveExecutionWorkspaceWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	repo := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "init")

	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	n := &Nexus{}
	project := store.Project{ID: "p", CanonicalPath: repo, DefaultIsolation: "worktree"}
	agent := store.Agent{ID: "agent-1", ProjectID: "p"}
	got, err := n.resolveExecutionWorkspace(context.Background(), project, agent, AgentConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if got == repo {
		t.Fatal("worktree isolation reused canonical project path")
	}
	if _, err := os.Stat(filepath.Join(got, "README.md")); err != nil {
		t.Fatalf("worktree missing checkout: %v", err)
	}
	got2, err := n.resolveExecutionWorkspace(context.Background(), project, agent, AgentConfig{})
	if err != nil || got2 != got {
		t.Fatalf("worktree reuse=%q err=%v want %q", got2, err, got)
	}

	projectDefaultCheckout := store.Project{ID: "p-explicit", CanonicalPath: repo, DefaultIsolation: "project"}
	explicitAgent := store.Agent{ID: "agent-explicit", ProjectID: projectDefaultCheckout.ID}
	explicitWorktree, err := n.resolveAutonomousExecutionWorkspace(context.Background(), projectDefaultCheckout, explicitAgent, AgentConfig{Isolation: "worktree"})
	if err != nil {
		t.Fatalf("explicit Agent worktree isolation must override project default safely: %v", err)
	}
	if explicitWorktree == repo {
		t.Fatal("autonomous explicit worktree isolation reused canonical project path")
	}
}

func TestEnsureAgentWorktreeRejectsManagedPathTraversal(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	repo := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
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

	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)

	tests := []struct {
		name    string
		project store.Project
		agent   store.Agent
	}{
		{
			name:    "project traversal",
			project: store.Project{ID: "../escape-project", CanonicalPath: repo, DefaultIsolation: "worktree"},
			agent:   store.Agent{ID: "agent-1", ProjectID: "../escape-project"},
		},
		{
			name:    "agent traversal",
			project: store.Project{ID: "project-1", CanonicalPath: repo, DefaultIsolation: "worktree"},
			agent:   store.Agent{ID: "../escape-agent", ProjectID: "project-1"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ensureAgentWorktree(context.Background(), tc.project, tc.agent); err == nil || !strings.Contains(err.Error(), "unsafe managed") {
				t.Fatalf("managed path traversal must fail closed, got %v", err)
			}
		})
	}
}

func TestEnsureAgentWorktreeRejectsSymlinkRedirect(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require Windows developer/admin privileges")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}

	repo := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
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

	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	project := store.Project{ID: "project-1", CanonicalPath: repo, DefaultIsolation: "worktree"}
	agent := store.Agent{ID: "agent-1", ProjectID: project.ID}

	projectRoot := filepath.Join(dataDir, "worktrees", project.ID)
	if err := os.MkdirAll(projectRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	redirect := filepath.Join(projectRoot, agent.ID)
	if err := os.Symlink(outside, redirect); err != nil {
		t.Fatal(err)
	}

	if _, err := ensureAgentWorktree(context.Background(), project, agent); err == nil || !strings.Contains(err.Error(), "must not be a symlink") {
		t.Fatalf("managed worktree symlink redirect must fail closed, got %v", err)
	}
}
