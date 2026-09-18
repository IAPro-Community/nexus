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

func (n *Nexus) resolveAutonomousExecutionWorkspace(ctx context.Context, project store.Project, agent store.Agent, cfg AgentConfig) (string, error) {
	if strings.TrimSpace(cfg.Workspace) != "" || strings.ToLower(strings.TrimSpace(cfg.Isolation)) != "worktree" && strings.ToLower(strings.TrimSpace(project.DefaultIsolation)) != "worktree" {
		return "", fmt.Errorf("autonomous execution requires AgentConfig Isolation=worktree; configure a dedicated worktree before starting the mission")
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

func ensureAgentWorktree(ctx context.Context, project store.Project, agent store.Agent) (string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", fmt.Errorf("worktree isolation requires git: %w", err)
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", project.CanonicalPath, "rev-parse", "--is-inside-work-tree").CombinedOutput(); err != nil || strings.TrimSpace(string(out)) != "true" {
		return "", fmt.Errorf("worktree isolation requires a git repository at %s", project.CanonicalPath)
	}
	dataDir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	root := filepath.Join(dataDir, "worktrees", project.ID)
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", fmt.Errorf("create worktree root: %w", err)
	}
	path := filepath.Join(root, agent.ID)
	if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
		if out, err := exec.CommandContext(ctx, "git", "-C", path, "rev-parse", "--show-toplevel").CombinedOutput(); err == nil {
			resolved := strings.TrimSpace(string(out))
			if resolved != "" {
				return config.CanonicalWorkspacePath(resolved)
			}
		}
		return "", fmt.Errorf("existing worktree path is not a valid git worktree: %s", path)
	} else if statErr != nil && !os.IsNotExist(statErr) {
		return "", statErr
	}

	branchSuffix := unsafeBranchChars.ReplaceAllString(agent.ID, "-")
	branchSuffix = strings.Trim(branchSuffix, "-./")
	if branchSuffix == "" {
		branchSuffix = "agent"
	}
	branch := "iapro/agent/" + branchSuffix

	branchExists := exec.CommandContext(ctx, "git", "-C", project.CanonicalPath, "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil
	args := []string{"-C", project.CanonicalPath, "worktree", "add"}
	if branchExists {
		args = append(args, path, branch)
	} else {
		args = append(args, "-b", branch, path, "HEAD")
	}
	if out, err := exec.CommandContext(ctx, "git", args...).CombinedOutput(); err != nil {
		return "", fmt.Errorf("create agent worktree: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return config.CanonicalWorkspacePath(path)
}
