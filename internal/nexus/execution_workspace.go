package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/flags"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

type ExecutionWorkspaceKind string

const (
	ExecutionWorkspaceProject  ExecutionWorkspaceKind = "PROJECT"
	ExecutionWorkspaceWorktree ExecutionWorkspaceKind = "WORKTREE"
)

// ExecutionWorkspace identifies the checkout/environment used by one task or
// candidate. It is intentionally independent from Agent identity.
type ExecutionWorkspace struct {
	ID        string                 `json:"id"`
	ProjectID string                 `json:"project_id"`
	Kind      ExecutionWorkspaceKind `json:"kind"`
	Path      string                 `json:"path"`
	Branch    string                 `json:"branch,omitempty"`
	BaseRef   string                 `json:"base_ref,omitempty"`
}

type ExecutionWorkspaceRequest struct {
	ID      string `json:"id"`
	BaseRef string `json:"base_ref,omitempty"`
}

// ResolveTaskExecutionWorkspace creates or reuses a task-scoped worktree.
// The v1 Agent-owned resolver remains untouched at its call sites until the
// v1.1 rollout flag is explicitly enabled by a higher-level execution path.
func (n *Nexus) ResolveTaskExecutionWorkspace(ctx context.Context, project store.Project, req ExecutionWorkspaceRequest) (ExecutionWorkspace, error) {
	if !flags.Enabled(flags.ExecutionWorkspacesV2) {
		return ExecutionWorkspace{}, errors.New("execution_workspaces_v2 is disabled")
	}
	if strings.TrimSpace(req.ID) == "" {
		return ExecutionWorkspace{}, errors.New("execution workspace id is required")
	}
	baseRef := strings.TrimSpace(req.BaseRef)
	if baseRef == "" {
		baseRef = "HEAD"
	}
	path, err := ensureManagedWorktree(ctx, project, managedWorktreeRequest{
		RootName:     "execution-workspaces",
		EntityLabel:  "execution workspace id",
		EntityID:     req.ID,
		BranchPrefix: "iapro/workspace/",
		BaseRef:      baseRef,
	})
	if err != nil {
		return ExecutionWorkspace{}, err
	}
	return ExecutionWorkspace{
		ID:        req.ID,
		ProjectID: project.ID,
		Kind:      ExecutionWorkspaceWorktree,
		Path:      path,
		Branch:    "iapro/workspace/" + sanitizedWorkspaceBranchSuffix(req.ID),
		BaseRef:   baseRef,
	}, nil
}

func sanitizedWorkspaceBranchSuffix(id string) string {
	suffix := unsafeBranchChars.ReplaceAllString(id, "-")
	suffix = strings.Trim(suffix, "-./")
	if suffix == "" {
		return "workspace"
	}
	return suffix
}

func executionWorkspaceManagedRoot(dataDir, projectID string) string {
	return filepath.Join(dataDir, "execution-workspaces", projectID)
}

func taskExecutionWorkspaceID(runID, packageID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(runID) + "\x00" + strings.TrimSpace(packageID)))
	return "ws_" + hex.EncodeToString(sum[:10])
}
