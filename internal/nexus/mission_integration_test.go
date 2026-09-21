package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"\n\t"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestMissionDeliveryIntegrationCombinesParallelDisjointWorktrees(t *testing.T) {
	n, st, project, base := integrationTestProject(t)
	a := integrationTestWorktree(t, project.CanonicalPath, "a")
	b := integrationTestWorktree(t, project.CanonicalPath, "b")
	writeIntegrationFile(t, a, "backend.txt", "backend change\n")
	writeIntegrationFile(t, b, "frontend.txt", "frontend change\n")

	run := &runner.MissionRun{
		ID: "run-integrate", ProjectID: project.ID, Workspace: project.CanonicalPath, Autonomous: true,
		PackageRuns: []runner.PackageRun{
			integrationVerifiedPackage("backend", a, base, nil, nil),
			integrationVerifiedPackage("frontend", b, base, nil, nil),
		},
	}
	exec := &nexusPackageExecutor{n: n}
	evidence, err := exec.IntegrateDelivery(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	if got := readIntegrationFile(t, evidence.Workspace, "backend.txt"); got != "backend change\n" {
		t.Fatalf("integrated backend=%q", got)
	}
	if got := readIntegrationFile(t, evidence.Workspace, "frontend.txt"); got != "frontend change\n" {
		t.Fatalf("integrated frontend=%q", got)
	}
	if _, err := os.Stat(filepath.Join(project.CanonicalPath, "backend.txt")); !os.IsNotExist(err) {
		t.Fatalf("canonical checkout was mutated: %v", err)
	}
	if len(evidence.PackageIDs) != 2 {
		t.Fatalf("integration packages=%v, want 2", evidence.PackageIDs)
	}
	_ = st
}

func TestMissionDeliveryIntegrationRejectsParallelPathConflict(t *testing.T) {
	n, _, project, base := integrationTestProject(t)
	a := integrationTestWorktree(t, project.CanonicalPath, "conflict-a")
	b := integrationTestWorktree(t, project.CanonicalPath, "conflict-b")
	writeIntegrationFile(t, a, "shared.txt", "from A\n")
	writeIntegrationFile(t, b, "shared.txt", "from B\n")

	run := &runner.MissionRun{
		ID: "run-conflict", ProjectID: project.ID, Workspace: project.CanonicalPath, Autonomous: true,
		PackageRuns: []runner.PackageRun{
			integrationVerifiedPackage("A", a, base, nil, nil),
			integrationVerifiedPackage("B", b, base, nil, nil),
		},
	}
	_, err := (&nexusPackageExecutor{n: n}).IntegrateDelivery(context.Background(), run)
	if err == nil || !strings.Contains(err.Error(), "parallel delivery conflict") {
		t.Fatalf("parallel conflicting delivery must fail closed, got %v", err)
	}
	if got := readIntegrationFile(t, project.CanonicalPath, "shared.txt"); got != "baseline\n" {
		t.Fatalf("canonical shared file changed to %q", got)
	}
}

func TestMissionDeliveryIntegrationAllowsDescendantOverride(t *testing.T) {
	n, _, project, base := integrationTestProject(t)
	parent := integrationTestWorktree(t, project.CanonicalPath, "parent")
	child := integrationTestWorktree(t, project.CanonicalPath, "child")

	writeIntegrationFile(t, parent, "shared.txt", "parent state\n")
	writeIntegrationFile(t, child, "shared.txt", "parent state\n")
	parentDigest := integrationBaselineDigest(t, filepath.Join(child, "shared.txt"))
	writeIntegrationFile(t, child, "shared.txt", "child state\n")

	run := &runner.MissionRun{
		ID: "run-descendant", ProjectID: project.ID, Workspace: project.CanonicalPath, Autonomous: true,
		PackageRuns: []runner.PackageRun{
			integrationVerifiedPackage("parent", parent, base, nil, nil),
			integrationVerifiedPackage("child", child, base, []string{"parent"}, map[string]string{"shared.txt": parentDigest}),
		},
	}
	evidence, err := (&nexusPackageExecutor{n: n}).IntegrateDelivery(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	if got := readIntegrationFile(t, evidence.Workspace, "shared.txt"); got != "child state\n" {
		t.Fatalf("descendant override=%q, want child state", got)
	}
}

func TestPrepareAutonomousPackageWorkspaceMaterializesDependencies(t *testing.T) {
	_, _, project, base := integrationTestProject(t)
	parent := integrationTestWorktree(t, project.CanonicalPath, "dependency-parent")
	child := integrationTestWorktree(t, project.CanonicalPath, "dependency-child")
	writeIntegrationFile(t, parent, "backend.txt", "dependency result\n")

	dep := integrationVerifiedPackage("backend", parent, base, nil, nil)
	target := runner.PackageRun{PackageID: "qa", Dependencies: []string{"backend"}, Workspace: child}
	run := &runner.MissionRun{
		ID: "run-dependency", ProjectID: project.ID, Workspace: project.CanonicalPath, Autonomous: true,
		PackageRuns: []runner.PackageRun{dep, target},
	}
	if err := prepareAutonomousPackageWorkspace(context.Background(), run, &run.PackageRuns[1], project.CanonicalPath, child); err != nil {
		t.Fatal(err)
	}
	if got := readIntegrationFile(t, child, "backend.txt"); got != "dependency result\n" {
		t.Fatalf("dependency materialization=%q", got)
	}
	if _, err := os.Stat(filepath.Join(project.CanonicalPath, "backend.txt")); !os.IsNotExist(err) {
		t.Fatalf("canonical checkout was mutated: %v", err)
	}
}

func TestPrepareAutonomousPackageWorkspaceRejectsDirtyOrStaleReuse(t *testing.T) {
	_, _, project, _ := integrationTestProject(t)
	dirty := integrationTestWorktree(t, project.CanonicalPath, "dirty")
	writeIntegrationFile(t, dirty, "leftover.txt", "old work\n")
	run := &runner.MissionRun{ID: "run-dirty", ProjectID: project.ID, Workspace: project.CanonicalPath, Autonomous: true}
	pkg := &runner.PackageRun{PackageID: "root"}
	if err := prepareAutonomousPackageWorkspace(context.Background(), run, pkg, project.CanonicalPath, dirty); err == nil || !strings.Contains(err.Error(), "pre-existing changes") {
		t.Fatalf("dirty reused worktree must fail closed, got %v", err)
	}

	stale := integrationTestWorktree(t, project.CanonicalPath, "stale")
	writeIntegrationFile(t, stale, "stale.txt", "commit on stale branch\n")
	runIntegrationGit(t, stale, "add", "stale.txt")
	runIntegrationGit(t, stale, "commit", "-m", "stale")
	if err := prepareAutonomousPackageWorkspace(context.Background(), run, pkg, project.CanonicalPath, stale); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale reused worktree must fail closed, got %v", err)
	}
}

func integrationTestProject(t *testing.T) (*Nexus, *store.Store, store.Project, string) {
	t.Helper()
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	runIntegrationGit(t, root, "init", "-q")
	runIntegrationGit(t, root, "config", "user.email", "nexus-test@example.invalid")
	runIntegrationGit(t, root, "config", "user.name", "Nexus Test")
	writeIntegrationFile(t, root, "shared.txt", "baseline\n")
	runIntegrationGit(t, root, "add", ".")
	runIntegrationGit(t, root, "commit", "-m", "baseline")
	base := strings.TrimSpace(runIntegrationGit(t, root, "rev-parse", "HEAD"))
	project, err := st.CreateProject(store.Project{Name: "Integration", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	return n, st, project, base
}

func integrationTestWorktree(t *testing.T, canonical, suffix string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "wt-"+suffix)
	runIntegrationGit(t, canonical, "worktree", "add", "-q", "-b", "test/"+suffix, path, "HEAD")
	return path
}

func integrationVerifiedPackage(id, workspace, base string, deps []string, baseline map[string]string) runner.PackageRun {
	if baseline == nil {
		baseline = map[string]string{}
	}
	now := integrationTestTime()
	return runner.PackageRun{
		PackageID: id,
		Workspace: workspace,
		Dependencies: deps,
		State: runner.StateVerified,
		ContextCapsule: &runner.ContextCapsule{Head: base, BaselineWorkspaceSnapshot: baseline},
		WorkReceipt: &runner.WorkReceipt{
			ID: "receipt-" + id, RunID: "run", StepID: id, Status: "VERIFIED",
			BaseRevision: base, ResultRevision: base, CompletedAt: now,
		},
	}
}

func integrationTestTime() time.Time {
	return time.Unix(1_700_000_000, 0).UTC()
}

func writeIntegrationFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readIntegrationFile(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func integrationBaselineDigest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func runIntegrationGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}
