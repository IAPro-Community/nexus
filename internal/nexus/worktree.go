package nexus

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/executil"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

var unsafeBranchChars = regexp.MustCompile(`[^A-Za-z0-9._/-]+`)

// resolveExecutionWorkspace returns the concrete working directory for an
// Agent. "project" uses the canonical checkout; "worktree" creates/reuses a
// stable Agent-owned git worktree outside the source repository. Historical
// credential-isolation values (developer/strict/compat) are treated as project
// workspace isolation because provider credential sandboxing remains owned by
// the ai-cli profile layer.
func (n *Nexus) resolveExecutionWorkspace(ctx context.Context, project store.Project, agent store.Agent, cfg AgentConfig) (string, error) {
	return n.resolveExecutionWorkspaceWithMode(ctx, project, agent, cfg, false)
}

func autonomousAgentUsesWorktree(project store.Project, cfg AgentConfig) bool {
	if strings.TrimSpace(cfg.Workspace) != "" {
		return false
	}
	isolation := strings.ToLower(strings.TrimSpace(cfg.Isolation))
	if isolation == "" {
		isolation = strings.ToLower(strings.TrimSpace(project.DefaultIsolation))
	}
	return isolation == "worktree"
}

func (n *Nexus) resolveAutonomousExecutionWorkspace(ctx context.Context, project store.Project, agent store.Agent, cfg AgentConfig) (string, error) {
	if !autonomousAgentUsesWorktree(project, cfg) {
		effectiveIsolation := strings.ToLower(strings.TrimSpace(cfg.Isolation))
		if effectiveIsolation == "" {
			effectiveIsolation = strings.ToLower(strings.TrimSpace(project.DefaultIsolation))
		}
		if effectiveIsolation == "" {
			effectiveIsolation = "project"
		}
		return "", fmt.Errorf("autonomous execution requires effective Isolation=worktree; got %q", effectiveIsolation)
	}
	return n.resolveExecutionWorkspaceWithMode(ctx, project, agent, cfg, true)
}

func (n *Nexus) resolveExecutionWorkspaceWithMode(ctx context.Context, project store.Project, agent store.Agent, cfg AgentConfig, autonomous bool) (string, error) {
	if strings.TrimSpace(cfg.Workspace) != "" {
		return store.CanonicalPath(cfg.Workspace)
	}
	isolation := strings.ToLower(strings.TrimSpace(cfg.Isolation))
	if isolation == "" {
		isolation = strings.ToLower(strings.TrimSpace(project.DefaultIsolation))
	}
	switch isolation {
	case "", "project", "branch", "none", "developer", "strict", "compat":
		return project.CanonicalPath, nil
	case "worktree":
		return ensureAgentWorktree(ctx, project, agent)
	default:
		if autonomous {
			return "", fmt.Errorf("autonomous execution requires AgentConfig Isolation=worktree; got %q", isolation)
		}
		return "", fmt.Errorf("unsupported agent isolation %q", isolation)
	}
}

func validateWorktreeBase(ctx context.Context, project store.Project) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("worktree isolation requires git: %w", err)
	}
	if out, err := executil.CommandContext(ctx, "git", "-C", project.CanonicalPath, "rev-parse", "--is-inside-work-tree").CombinedOutput(); err != nil || strings.TrimSpace(string(out)) != "true" {
		return fmt.Errorf("worktree isolation requires a git repository at %s", project.CanonicalPath)
	}
	return nil
}

func ensureAgentWorktree(ctx context.Context, project store.Project, agent store.Agent) (string, error) {
	return ensureManagedWorktree(ctx, project, managedWorktreeRequest{
		RootName:     "worktrees",
		EntityLabel:  "agent id",
		EntityID:     agent.ID,
		BranchPrefix: "iapro/agent/",
		BaseRef:      "HEAD",
	})
}

type managedWorktreeRequest struct {
	RootName     string
	EntityLabel  string
	EntityID     string
	BranchPrefix string
	BaseRef      string
}

// ensureManagedWorktree is the single filesystem/git primitive used by both
// the v1 Agent-owned worktree adapter and v1.1 task-scoped execution
// workspaces. Keeping path/symlink validation here prevents the new model from
// weakening the v1 isolation boundary.
func ensureManagedWorktree(ctx context.Context, project store.Project, req managedWorktreeRequest) (string, error) {
	if err := validateWorktreeBase(ctx, project); err != nil {
		return "", err
	}
	if err := validateManagedPathSegment("project id", project.ID); err != nil {
		return "", err
	}
	if err := validateManagedPathSegment(req.EntityLabel, req.EntityID); err != nil {
		return "", err
	}
	if err := validateManagedPathSegment("managed worktree root", req.RootName); err != nil {
		return "", err
	}

	dataDir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	worktreeRoot := filepath.Join(dataDir, req.RootName)
	if err := os.MkdirAll(worktreeRoot, 0o700); err != nil {
		return "", fmt.Errorf("create managed worktree root: %w", err)
	}
	canonicalRoot, err := canonicalExistingManagedDir(worktreeRoot)
	if err != nil {
		return "", fmt.Errorf("resolve managed worktree root: %w", err)
	}

	root := filepath.Join(worktreeRoot, project.ID)
	if err := rejectManagedSymlink(root, "project managed worktree root"); err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", fmt.Errorf("create project managed worktree root: %w", err)
	}
	canonicalProjectRoot, err := canonicalExistingManagedDir(root)
	if err != nil {
		return "", fmt.Errorf("resolve project managed worktree root: %w", err)
	}
	if !pathWithinRoot(canonicalProjectRoot, canonicalRoot) {
		return "", fmt.Errorf("project worktree root escapes Nexus managed worktree root")
	}

	path := filepath.Join(root, req.EntityID)
	if err := rejectManagedSymlink(path, "managed worktree"); err != nil {
		return "", err
	}
	if info, statErr := os.Lstat(path); statErr == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("existing worktree path is not a directory: %s", path)
		}
		if out, gitErr := executil.CommandContext(ctx, "git", "-C", path, "rev-parse", "--show-toplevel").CombinedOutput(); gitErr == nil {
			resolved := strings.TrimSpace(string(out))
			if resolved != "" {
				canonicalResolved, canonicalErr := config.CanonicalWorkspacePath(resolved)
				if canonicalErr != nil {
					return "", canonicalErr
				}
				if !pathWithinRoot(canonicalResolved, canonicalProjectRoot) {
					return "", fmt.Errorf("existing managed worktree escapes Nexus managed project root")
				}
				return canonicalResolved, nil
			}
		}
		return "", fmt.Errorf("existing worktree path is not a valid git worktree: %s", path)
	} else if statErr != nil && !os.IsNotExist(statErr) {
		return "", statErr
	}

	branchSuffix := unsafeBranchChars.ReplaceAllString(req.EntityID, "-")
	branchSuffix = strings.Trim(branchSuffix, "-./")
	if branchSuffix == "" {
		branchSuffix = "workspace"
	}
	branch := req.BranchPrefix + branchSuffix
	baseRef := strings.TrimSpace(req.BaseRef)
	if baseRef == "" {
		baseRef = "HEAD"
	}

	branchExists := executil.CommandContext(ctx, "git", "-C", project.CanonicalPath, "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil
	args := []string{"-C", project.CanonicalPath, "worktree", "add"}
	if branchExists {
		args = append(args, path, branch)
	} else {
		args = append(args, "-b", branch, path, baseRef)
	}
	if out, gitErr := executil.CommandContext(ctx, "git", args...).CombinedOutput(); gitErr != nil {
		return "", fmt.Errorf("create managed worktree: %w: %s", gitErr, strings.TrimSpace(string(out)))
	}
	resolved, err := config.CanonicalWorkspacePath(path)
	if err != nil {
		return "", err
	}
	if !pathWithinRoot(resolved, canonicalProjectRoot) {
		return "", fmt.Errorf("created managed worktree escapes Nexus managed project root")
	}
	return resolved, nil
}
