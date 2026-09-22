package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

const maxIntegrationEvidenceFiles = 512

type deliveryFileOwner struct {
	PackageID string
	State     string
}

// IntegrateDelivery materializes the verified package deltas into one
// Nexus-owned detached worktree. The canonical project checkout is never
// modified. Conflicting sibling workstreams fail closed.
func (e *nexusPackageExecutor) IntegrateDelivery(ctx context.Context, run *runner.MissionRun) (runner.DeliveryIntegrationEvidence, error) {
	if e == nil || e.n == nil || run == nil {
		return runner.DeliveryIntegrationEvidence{}, fmt.Errorf("delivery integration requires Nexus executor and Mission run")
	}
	st, err := e.n.OpenProject()
	if err != nil {
		return runner.DeliveryIntegrationEvidence{}, err
	}
	project, err := st.GetProject(run.ProjectID)
	if err != nil {
		return runner.DeliveryIntegrationEvidence{}, err
	}
	base, err := gitOutput(ctx, project.CanonicalPath, "rev-parse", "HEAD")
	if err != nil {
		return runner.DeliveryIntegrationEvidence{}, fmt.Errorf("resolve canonical integration base: %w", err)
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return runner.DeliveryIntegrationEvidence{}, fmt.Errorf("canonical integration base is empty")
	}

	dataDir, err := config.DataDir()
	if err != nil {
		return runner.DeliveryIntegrationEvidence{}, err
	}
	if err := validateIntegrationPathSegment("project id", run.ProjectID); err != nil {
		return runner.DeliveryIntegrationEvidence{}, err
	}
	if err := validateIntegrationPathSegment("mission run id", run.ID); err != nil {
		return runner.DeliveryIntegrationEvidence{}, err
	}
	integrationRoot := filepath.Join(dataDir, "integrations")
	target := filepath.Join(integrationRoot, run.ProjectID, run.ID)
	if err := recreateDetachedIntegrationWorktree(ctx, project.CanonicalPath, integrationRoot, target, base); err != nil {
		return runner.DeliveryIntegrationEvidence{}, err
	}

	ordered, err := missionTopologicalPackages(run)
	if err != nil {
		return runner.DeliveryIntegrationEvidence{}, err
	}
	owners := map[string]deliveryFileOwner{}
	changed := map[string]struct{}{}
	applied := []string{}
	for _, pkg := range ordered {
		if pkg.WorkReceipt == nil || !strings.EqualFold(pkg.WorkReceipt.Status, "VERIFIED") {
			continue
		}
		paths, deltaErr := packageDeliveryDeltaPaths(ctx, pkg, base)
		if deltaErr != nil {
			return runner.DeliveryIntegrationEvidence{}, fmt.Errorf("package %s delivery delta: %w", pkg.PackageID, deltaErr)
		}
		if len(paths) == 0 {
			continue
		}
		if err := applyPackageDeliveryDelta(ctx, run, pkg, target, paths, owners); err != nil {
			return runner.DeliveryIntegrationEvidence{}, err
		}
		applied = append(applied, pkg.PackageID)
		for _, rel := range paths {
			changed[rel] = struct{}{}
		}
	}

	if len(applied) == 0 {
		return runner.DeliveryIntegrationEvidence{}, fmt.Errorf("no verified package delivery could be materialized")
	}
	files := sortedMapKeys(changed)
	if len(files) > maxIntegrationEvidenceFiles {
		files = files[:maxIntegrationEvidenceFiles]
	}
	return runner.DeliveryIntegrationEvidence{
		Workspace:    target,
		BaseRevision: base,
		PackageIDs:   applied,
		ChangedFiles: files,
		IntegratedAt: time.Now().UTC(),
	}, nil
}

// prepareAutonomousPackageWorkspace proves a package worktree starts from the
// current canonical HEAD and is clean. For dependent packages, verified
// ancestor deltas are then materialized before ContextCapsule captures the
// package baseline, so later WorkReceipt evidence attributes only this
// package's own changes.
func prepareAutonomousPackageWorkspace(ctx context.Context, run *runner.MissionRun, pkg *runner.PackageRun, canonicalWorkspace, target string) error {
	if run == nil || pkg == nil {
		return fmt.Errorf("autonomous workspace preparation requires run and package")
	}
	base, err := gitOutput(ctx, canonicalWorkspace, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("resolve canonical workspace HEAD: %w", err)
	}
	base = strings.TrimSpace(base)
	targetHead, err := gitOutput(ctx, target, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("resolve package worktree HEAD: %w", err)
	}
	if strings.TrimSpace(targetHead) != base {
		return fmt.Errorf("autonomous package worktree is stale: head=%s canonical=%s", strings.TrimSpace(targetHead), base)
	}
	status, err := gitOutput(ctx, target, "status", "--porcelain=v1", "--untracked-files=normal")
	if err != nil {
		return fmt.Errorf("inspect package worktree cleanliness: %w", err)
	}
	if strings.TrimSpace(status) != "" {
		return fmt.Errorf("autonomous package worktree contains pre-existing changes; reconcile it before Mission execution")
	}
	if len(pkg.Dependencies) == 0 {
		return nil
	}
	return materializeMissionDependencies(ctx, run, pkg, target, base)
}

func materializeMissionDependencies(ctx context.Context, run *runner.MissionRun, pkg *runner.PackageRun, target, base string) error {
	ids, err := transitiveDependencyPackageIDs(run, pkg.PackageID)
	if err != nil {
		return err
	}
	ordered, err := missionTopologicalPackages(run)
	if err != nil {
		return err
	}
	required := map[string]struct{}{}
	for _, id := range ids {
		required[id] = struct{}{}
	}
	owners := map[string]deliveryFileOwner{}
	for _, dep := range ordered {
		if _, ok := required[dep.PackageID]; !ok {
			continue
		}
		if dep.WorkReceipt == nil || !strings.EqualFold(dep.WorkReceipt.Status, "VERIFIED") {
			return fmt.Errorf("dependency %s has no verified WorkReceipt", dep.PackageID)
		}
		paths, err := packageDeliveryDeltaPaths(ctx, dep, base)
		if err != nil {
			return fmt.Errorf("dependency %s delivery delta: %w", dep.PackageID, err)
		}
		if err := applyPackageDeliveryDelta(ctx, run, dep, target, paths, owners); err != nil {
			return err
		}
	}
	return nil
}

func packageDeliveryDeltaPaths(ctx context.Context, pkg *runner.PackageRun, canonicalBase string) ([]string, error) {
	if pkg == nil || strings.TrimSpace(pkg.Workspace) == "" || pkg.WorkReceipt == nil {
		return nil, nil
	}
	base := strings.TrimSpace(pkg.WorkReceipt.BaseRevision)
	if base == "" && pkg.ContextCapsule != nil {
		base = strings.TrimSpace(pkg.ContextCapsule.Head)
	}
	if base == "" {
		return nil, fmt.Errorf("missing package base revision")
	}
	if base != canonicalBase {
		return nil, fmt.Errorf("package base %s differs from canonical Mission base %s", base, canonicalBase)
	}

	seen := map[string]struct{}{}
	diff, err := gitOutputBytes(ctx, pkg.Workspace, "diff", "--name-only", "-z", base, "--")
	if err != nil {
		return nil, fmt.Errorf("diff package workspace from base: %w", err)
	}
	for _, rel := range splitNULPaths(diff) {
		seen[rel] = struct{}{}
	}
	untracked, err := gitOutputBytes(ctx, pkg.Workspace, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, fmt.Errorf("list package untracked files: %w", err)
	}
	for _, rel := range splitNULPaths(untracked) {
		seen[rel] = struct{}{}
	}

	out := make([]string, 0, len(seen))
	for rel := range seen {
		clean, err := safeDeliveryRelativePath(rel)
		if err != nil {
			return nil, err
		}
		if pkg.ContextCapsule != nil {
			if before, ok := pkg.ContextCapsule.BaselineWorkspaceSnapshot[clean]; ok {
				after := baselineCompatibleDigest(filepath.Join(pkg.Workspace, clean))
				if after == before {
					continue
				}
			}
		}
		out = append(out, clean)
	}
	sort.Strings(out)
	return out, nil
}

func applyPackageDeliveryDelta(ctx context.Context, run *runner.MissionRun, pkg *runner.PackageRun, target string, paths []string, owners map[string]deliveryFileOwner) error {
	for _, rel := range paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		clean, err := safeDeliveryRelativePath(rel)
		if err != nil {
			return err
		}
		state, err := deliveryPathState(filepath.Join(pkg.Workspace, clean))
		if err != nil {
			return fmt.Errorf("inspect package %s path %s: %w", pkg.PackageID, clean, err)
		}
		if previous, exists := owners[clean]; exists && previous.PackageID != pkg.PackageID && previous.State != state {
			if !missionPackageDependsOn(run, pkg.PackageID, previous.PackageID) {
				return fmt.Errorf("parallel delivery conflict on %s between %s and %s", clean, previous.PackageID, pkg.PackageID)
			}
		}
		if err := copyDeliveryPath(filepath.Join(pkg.Workspace, clean), filepath.Join(target, clean), target); err != nil {
			return fmt.Errorf("materialize package %s path %s: %w", pkg.PackageID, clean, err)
		}
		owners[clean] = deliveryFileOwner{PackageID: pkg.PackageID, State: state}
	}
	return nil
}

func validateIntegrationPathSegment(name, value string) error {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." ||
		strings.ContainsAny(value, `/\\`) || filepath.Base(value) != value {
		return fmt.Errorf("unsafe integration %s %q", name, value)
	}
	return nil
}

func recreateDetachedIntegrationWorktree(ctx context.Context, canonical, integrationRoot, target, base string) error {
	if strings.TrimSpace(integrationRoot) == "" || !pathWithinRoot(target, integrationRoot) {
		return fmt.Errorf("integration target escapes Nexus integration root")
	}
	if _, err := os.Stat(target); err == nil {
		_ = exec.CommandContext(ctx, "git", "-C", canonical, "worktree", "remove", "--force", target).Run()
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("remove stale integration workspace: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	out, err := exec.CommandContext(ctx, "git", "-C", canonical, "worktree", "add", "--detach", target, base).CombinedOutput()
	if err != nil {
		return fmt.Errorf("create integration worktree: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func missionTopologicalPackages(run *runner.MissionRun) ([]*runner.PackageRun, error) {
	if run == nil {
		return nil, fmt.Errorf("mission run is required")
	}
	byID := map[string]*runner.PackageRun{}
	indegree := map[string]int{}
	children := map[string][]string{}
	for i := range run.PackageRuns {
		pkg := &run.PackageRuns[i]
		if strings.TrimSpace(pkg.PackageID) == "" {
			return nil, fmt.Errorf("mission package ID is required")
		}
		byID[pkg.PackageID] = pkg
		indegree[pkg.PackageID] = len(pkg.Dependencies)
		for _, dep := range pkg.Dependencies {
			children[dep] = append(children[dep], pkg.PackageID)
		}
	}
	queue := []string{}
	for _, pkg := range run.PackageRuns {
		if indegree[pkg.PackageID] == 0 {
			queue = append(queue, pkg.PackageID)
		}
	}
	out := make([]*runner.PackageRun, 0, len(run.PackageRuns))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		pkg := byID[id]
		if pkg == nil {
			return nil, fmt.Errorf("dependency graph references unknown package %s", id)
		}
		out = append(out, pkg)
		for _, child := range children[id] {
			indegree[child]--
			if indegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}
	if len(out) != len(run.PackageRuns) {
		return nil, fmt.Errorf("mission dependency graph is cyclic or incomplete")
	}
	return out, nil
}

func transitiveDependencyPackageIDs(run *runner.MissionRun, packageID string) ([]string, error) {
	byID := map[string]*runner.PackageRun{}
	for i := range run.PackageRuns {
		byID[run.PackageRuns[i].PackageID] = &run.PackageRuns[i]
	}
	start := byID[packageID]
	if start == nil {
		return nil, fmt.Errorf("package %s not found", packageID)
	}
	seen := map[string]struct{}{}
	var visit func(string) error
	visit = func(id string) error {
		pkg := byID[id]
		if pkg == nil {
			return fmt.Errorf("dependency %s not found", id)
		}
		for _, dep := range pkg.Dependencies {
			if _, ok := seen[dep]; ok {
				continue
			}
			seen[dep] = struct{}{}
			if err := visit(dep); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(packageID); err != nil {
		return nil, err
	}
	delete(seen, packageID)
	return sortedMapKeys(seen), nil
}

func missionPackageDependsOn(run *runner.MissionRun, packageID, ancestorID string) bool {
	ids, err := transitiveDependencyPackageIDs(run, packageID)
	if err != nil {
		return false
	}
	for _, id := range ids {
		if id == ancestorID {
			return true
		}
	}
	return false
}

func safeDeliveryRelativePath(rel string) (string, error) {
	rel = filepath.Clean(strings.TrimSpace(strings.ReplaceAll(rel, "\\", "/")))
	if rel == "" || rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe delivery path %q", rel)
	}
	return rel, nil
}

func deliveryPathState(path string) (string, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "DELETED", nil
	}
	if err != nil {
		return "", err
	}
	h := sha256.New()
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		_, _ = io.WriteString(h, "symlink\x00"+target)
	case info.Mode().IsRegular():
		file, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer file.Close()
		_, _ = io.WriteString(h, fmt.Sprintf("file\x00%o\x00", info.Mode().Perm()))
		if _, err := io.Copy(h, file); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported delivery file type %s", info.Mode())
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func baselineCompatibleDigest(path string) string {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "deleted-or-unreadable"
	}
	if err != nil || !info.Mode().IsRegular() {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "deleted-or-unreadable"
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func copyDeliveryPath(source, destination, integrationRoot string) error {
	info, err := os.Lstat(source)
	if os.IsNotExist(err) {
		return os.RemoveAll(destination)
	}
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(source)
		if err != nil {
			return err
		}
		if filepath.IsAbs(target) {
			return fmt.Errorf("absolute symlink target is not allowed")
		}
		resolved := filepath.Clean(filepath.Join(filepath.Dir(destination), target))
		if !pathWithinRoot(resolved, integrationRoot) {
			return fmt.Errorf("symlink target escapes integrated workspace")
		}
		_ = os.RemoveAll(destination)
		return os.Symlink(target, destination)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("unsupported file type %s", info.Mode())
	}
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	tmp, err := os.CreateTemp(filepath.Dir(destination), ".nexus-integrate-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := io.Copy(tmp, src); err != nil {
		return err
	}
	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, destination); err != nil {
		return err
	}
	ok = true
	return nil
}

func pathWithinRoot(path, root string) bool {
	absPath, err1 := filepath.Abs(path)
	absRoot, err2 := filepath.Abs(root)
	if err1 != nil || err2 != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absPath)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func sortedMapKeys[T any](values map[string]T) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func splitNULPaths(raw []byte) []string {
	parts := strings.Split(string(raw), "\x00")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, strings.ReplaceAll(part, "\\", "/"))
	}
	return out
}

func gitOutput(ctx context.Context, workspace string, args ...string) (string, error) {
	out, err := gitOutputBytes(ctx, workspace, args...)
	return string(out), err
}

func gitOutputBytes(ctx context.Context, workspace string, args ...string) ([]byte, error) {
	all := append([]string{"-C", workspace}, args...)
	out, err := exec.CommandContext(ctx, "git", all...).Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}
