package web

import (
    "context"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"
)

func TestParseGitWorktreesPorcelain(t *testing.T) {
    data := []byte("worktree /tmp/project root\x00HEAD aaa\x00branch refs/heads/main\x00\x00" +
        "worktree /tmp/linked\ncheckout\x00HEAD bbb\x00detached\x00locked active runtime\x00\x00" +
        "worktree /tmp/prunable\x00HEAD ccc\x00branch refs/heads/feature/isolated\x00prunable missing\x00\x00")
    got := parseGitWorktreesPorcelain(data)
    if len(got) != 3 {
        t.Fatalf("got %d worktrees, want 3", len(got))
    }
    if got[0].Branch != "main" || !got[0].IsPrimary || got[0].Path != filepath.Clean("/tmp/project root") {
        t.Fatalf("invalid primary: %+v", got[0])
    }
    if !got[1].Detached || !got[1].Locked || got[1].IsPrimary || !strings.Contains(got[1].Path, "\n") {
        t.Fatalf("invalid detached worktree: %+v", got[1])
    }
    if got[2].Branch != "feature/isolated" || !got[2].Prunable {
        t.Fatalf("invalid prunable worktree: %+v", got[2])
    }
    if got := parseGitWorktreesPorcelain(nil); len(got) != 0 {
        t.Fatalf("empty output returned %d worktrees", len(got))
    }
}

func TestListProjectGitWorktreesReadsLinkedCheckouts(t *testing.T) {
    if _, err := exec.LookPath("git"); err != nil {
        t.Skip("git not installed")
    }
    root := t.TempDir()
    repo := filepath.Join(root, "main checkout")
    linked := filepath.Join(root, "feature checkout")
    if err := os.MkdirAll(repo, 0o700); err != nil {
        t.Fatal(err)
    }
    run := func(args ...string) {
        t.Helper()
        cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
        cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
            "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
        if out, err := cmd.CombinedOutput(); err != nil {
            t.Fatalf("git %v: %v: %s", args, err, out)
        }
    }
    run("init", "-q")
    run("commit", "--allow-empty", "-m", "initial")
    run("worktree", "add", "-b", "feature/parallel", linked)
    worktrees, err := listProjectGitWorktrees(context.Background(), repo)
    if err != nil {
        t.Fatal(err)
    }
    if len(worktrees) != 2 || !worktrees[0].IsPrimary || worktrees[1].IsPrimary {
        t.Fatalf("unexpected worktree list: %+v", worktrees)
    }
    if worktrees[1].Branch != "feature/parallel" || worktrees[1].Path != linked {
        t.Fatalf("linked worktree metadata mismatch: %+v", worktrees[1])
    }
}
