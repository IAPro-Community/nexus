package web

import (
    "bytes"
    "context"
    "fmt"
    "path/filepath"
    "strings"
    "time"

    "github.com/IAPro-Community/nexus/internal/executil"
)

// ProjectGitWorktree is the read-only Git view of a checkout. It does not
// imply ownership by Nexus or authorization to remove that worktree.
type ProjectGitWorktree struct {
    Path      string `json:"path"`
    Head      string `json:"head"`
    Branch    string `json:"branch"`
    IsPrimary bool   `json:"is_primary"`
    Detached  bool   `json:"detached"`
    Locked    bool   `json:"locked"`
    Prunable  bool   `json:"prunable"`
}

// listProjectGitWorktrees reads Git's NUL-delimited porcelain format so paths
// containing whitespace or newlines cannot corrupt record boundaries.
// This endpoint does not call prune, remove or mutate worktrees.
func listProjectGitWorktrees(parent context.Context, dir string) ([]ProjectGitWorktree, error) {
    ctx, cancel := context.WithTimeout(parent, 3*time.Second)
    defer cancel()

    output, err := executil.CommandContext(ctx, "git", "-C", dir, "worktree", "list", "--porcelain", "-z").Output()
    if err != nil {
        return nil, fmt.Errorf("list git worktrees: %w", err)
    }
    if len(output) > 512*1024 {
        return nil, fmt.Errorf("git worktree list output exceeds allowed limit")
    }
    return parseGitWorktreesPorcelain(output), nil
}

func parseGitWorktreesPorcelain(output []byte) []ProjectGitWorktree {
    worktrees := make([]ProjectGitWorktree, 0)
    for _, record := range bytes.Split(output, []byte{0, 0}) {
        if len(record) == 0 {
            continue
        }
        var worktree ProjectGitWorktree
        for _, lineBytes := range bytes.Split(record, []byte{0}) {
            line := string(lineBytes)
            switch {
            case strings.HasPrefix(line, "worktree "):
                worktree.Path = filepath.Clean(strings.TrimPrefix(line, "worktree "))
            case strings.HasPrefix(line, "HEAD "):
                worktree.Head = strings.TrimPrefix(line, "HEAD ")
            case strings.HasPrefix(line, "branch refs/heads/"):
                worktree.Branch = strings.TrimPrefix(line, "branch refs/heads/")
            case line == "detached":
                worktree.Detached = true
            case line == "locked" || strings.HasPrefix(line, "locked "):
                worktree.Locked = true
            case line == "prunable" || strings.HasPrefix(line, "prunable "):
                worktree.Prunable = true
            }
        }
        if worktree.Path == "" || worktree.Path == "." {
            continue
        }
        worktree.IsPrimary = len(worktrees) == 0
        worktrees = append(worktrees, worktree)
    }
    return worktrees
}
