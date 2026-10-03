package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/flags"
	coresecurity "github.com/IAPro-Community/nexus/internal/core/security"
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
	if err := applyProjectManifestWorkspaceResources(project.CanonicalPath, path); err != nil {
		return ExecutionWorkspace{}, fmt.Errorf("prepare execution workspace resources: %w", err)
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


const maxCopiedWorkspaceFileBytes int64 = 16 << 20

func applyProjectManifestWorkspaceResources(canonical, target string) error {
	manifest, _, err := LoadProjectManifest(canonical)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, rel := range manifest.Workspace.SharedDirectories {
		src, err := secureManifestSource(canonical, rel, true)
		if err != nil {
			return fmt.Errorf("shared directory %q: %w", rel, err)
		}
		dst, err := secureWorkspaceDestination(target, rel)
		if err != nil {
			return fmt.Errorf("shared directory destination %q: %w", rel, err)
		}
		if err := coresecurity.SafeLinkOrCopy(src, dst); err != nil {
			return fmt.Errorf("share directory %q: %w", rel, err)
		}
	}
	for _, rel := range manifest.Workspace.CopiedFiles {
		src, err := secureManifestSource(canonical, rel, false)
		if err != nil {
			return fmt.Errorf("copied file %q: %w", rel, err)
		}
		info, err := os.Stat(src)
		if err != nil {
			return err
		}
		if info.Size() > maxCopiedWorkspaceFileBytes {
			return fmt.Errorf("copied file %q exceeds %d bytes", rel, maxCopiedWorkspaceFileBytes)
		}
		dst, err := secureWorkspaceDestination(target, rel)
		if err != nil {
			return fmt.Errorf("copied file destination %q: %w", rel, err)
		}
		if err := coresecurity.CopyFile(src, dst); err != nil {
			return fmt.Errorf("copy file %q: %w", rel, err)
		}
	}
	return nil
}

func secureManifestSource(root, rel string, wantDir bool) (string, error) {
	if err := validateManifestRelativePath(rel); err != nil {
		return "", err
	}
	candidate := filepath.Join(root, filepath.FromSlash(rel))
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", err
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	canonicalRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if !pathWithinRoot(resolved, canonicalRoot) {
		return "", fmt.Errorf("source resolves outside project root")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if wantDir && !info.IsDir() {
		return "", fmt.Errorf("source is not a directory")
	}
	if !wantDir && !info.Mode().IsRegular() {
		return "", fmt.Errorf("source is not a regular file")
	}
	return resolved, nil
}

func secureWorkspaceDestination(root, rel string) (string, error) {
	if err := validateManifestRelativePath(rel); err != nil {
		return "", err
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	parts := strings.Split(filepath.Clean(filepath.FromSlash(rel)), string(filepath.Separator))
	current := root
	for i, part := range parts {
		current = filepath.Join(current, part)
		if !pathWithinRoot(current, root) {
			return "", fmt.Errorf("destination escapes execution workspace")
		}
		info, statErr := os.Lstat(current)
		isLeaf := i == len(parts)-1
		if statErr == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("destination traverses symlink at %s", part)
			}
			if !isLeaf && !info.IsDir() {
				return "", fmt.Errorf("destination parent %s is not a directory", part)
			}
			continue
		}
		if !os.IsNotExist(statErr) {
			return "", statErr
		}
		if !isLeaf {
			if err := os.Mkdir(current, 0o700); err != nil && !os.IsExist(err) {
				return "", err
			}
		}
	}
	return filepath.Join(root, filepath.FromSlash(rel)), nil
}
