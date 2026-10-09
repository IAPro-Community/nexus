package contextsnapshot

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func gitTestCommand(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v (%s)", args, err, output)
	}
}

func TestCommandOutputBlockingHelper(t *testing.T) {
	if os.Getenv("NEXUS_CONTEXTSNAPSHOT_TEST_BLOCK") == "1" {
		// A timer keeps the helper alive without triggering Go's deadlock detector at GOMAXPROCS=1.
		for {
			time.Sleep(time.Hour)
		}
	}
}

func TestCommandOutputHonorsDeadline(t *testing.T) {
	t.Setenv("NEXUS_CONTEXTSNAPSHOT_TEST_BLOCK", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCommandOutputBlockingHelper$")
	// Keep the child lightweight while go test runs packages concurrently on Windows.
	command.Env = append(os.Environ(), "GOMAXPROCS=1")

	started := time.Now()
	_, err := commandOutput(ctx, command)
	if !errors.Is(err, context.DeadlineExceeded) {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("command error=%v, stderr=%q, want context deadline exceeded", err, exitErr.Stderr)
		}
		t.Fatalf("command error=%v, want context deadline exceeded", err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("command took %s after its 100ms deadline", elapsed)
	}
}

func TestInspectCodeIdentityTracksWorktreeAndUntrackedChanges(t *testing.T) {
	root := t.TempDir()
	gitTestCommand(t, root, "init")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n\ngo 1.25\n"), 0600); err != nil {
		t.Fatal(err)
	}
	gitTestCommand(t, root, "add", "go.mod")
	gitTestCommand(t, root, "-c", "user.email=test@example.invalid", "-c", "user.name=Test", "commit", "-m", "initial")
	clean, err := InspectCodeIdentity(root)
	if err != nil {
		t.Fatal(err)
	}
	if clean.State != CodeIdentityClean || !clean.Complete || clean.HeadSHA == "" || clean.IdentityDigest == "" {
		t.Fatalf("unexpected clean identity: %+v", clean)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module changed\n\ngo 1.25\n"), 0600); err != nil {
		t.Fatal(err)
	}
	changed, err := InspectCodeIdentity(root)
	if err != nil {
		t.Fatal(err)
	}
	if changed.State != CodeIdentityDirty || changed.WorktreeDigest == clean.WorktreeDigest || changed.IdentityDigest == clean.IdentityDigest {
		t.Fatalf("worktree change was not captured: clean=%+v changed=%+v", clean, changed)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("untracked"), 0600); err != nil {
		t.Fatal(err)
	}
	untracked, err := InspectCodeIdentity(root)
	if err != nil {
		t.Fatal(err)
	}
	if untracked.UntrackedDigest == "" || untracked.IdentityDigest == changed.IdentityDigest {
		t.Fatalf("untracked change was not captured: %+v", untracked)
	}
}

func TestDiscoverReturnsStructuredFactsWithProvenance(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".github", "workflows"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"demo","scripts":{"test":"vitest"},"dependencies":{"react":"19"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.ts"), []byte("export {}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app.test.ts"), []byte("test('ok',()=>{})"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".github", "workflows", "ci.yml"), []byte("name: ci"), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := Discover(root, Metadata{ProjectID: "project-1"})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ProjectID != "project-1" || snapshot.ScannerVersion == "" || len(snapshot.Facts) < 4 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	for _, fact := range snapshot.Facts {
		if fact.ObservedAt.IsZero() || len(fact.Provenance) == 0 || strings.HasPrefix(fact.Provenance[0].SourcePath, root) {
			t.Fatalf("fact lacks safe provenance: %+v", fact)
		}
	}
}

func TestDiscoverWithIdentityReusesCallerObservation(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("Project conventions"), 0600); err != nil {
		t.Fatal(err)
	}
	identity := CodeIdentity{
		RepositoryKind: "filesystem",
		State:          CodeIdentityNonGit,
		Complete:       true,
		IdentityDigest: "caller-observed-identity",
	}

	snapshot, err := DiscoverWithIdentity(root, Metadata{ProjectID: "project-cached"}, identity)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Identity.IdentityDigest != identity.IdentityDigest {
		t.Fatalf("discovery identity=%q, want supplied identity %q", snapshot.Identity.IdentityDigest, identity.IdentityDigest)
	}
	if snapshot.ProjectID != "project-cached" || len(snapshot.Facts) == 0 {
		t.Fatalf("unexpected snapshot metadata or facts: %+v", snapshot)
	}
}

func TestDiscoverExtractsOperationalProjectIntelligence(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "packages", "web"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25\n\nuse (\n ./services/api\n ./packages/web\n)\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"workspace","scripts":{"build":"vite build","test":"vitest run","lint":"eslint .","e2e":"playwright test"},"devDependencies":{"@playwright/test":"1.0.0","vite":"1.0.0","vitest":"1.0.0","react":"1.0.0"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "packages", "web", "package.json"), []byte(`{"name":"web"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".github", "workflows"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".github", "workflows", "ci.yml"), []byte("steps:\n  - run: npm run lint\n  - run: npm run test\n  - run: npm run build\n"), 0600); err != nil {
		t.Fatal(err)
	}

	snapshot, err := Discover(root, Metadata{})
	if err != nil {
		t.Fatal(err)
	}

	values := map[string]any{}
	for _, fact := range snapshot.Facts {
		values[fact.Category+":"+fact.Key] = fact.Value
	}
	for key, expected := range map[string]string{
		"commands:build":        "vite build",
		"commands:test":         "vitest run",
		"commands:lint":         "eslint .",
		"commands:e2e":          "playwright test",
		"frameworks:vite":       "vite",
		"frameworks:vitest":     "vitest",
		"frameworks:playwright": "@playwright/test",
	} {
		if values[key] != expected {
			t.Fatalf("missing operational fact %s=%q, got %v", key, expected, values[key])
		}
	}
	uses, ok := values["workspace:go.use"].([]string)
	if !ok || len(uses) != 2 || uses[0] != "./services/api" || uses[1] != "./packages/web" {
		t.Fatalf("unexpected go workspace facts: %#v", values["workspace:go.use"])
	}
	if _, ok := values["workspace:package.packages/web"]; !ok {
		t.Fatalf("expected nested workspace package fact, got keys %v", values)
	}
	if _, ok := values["ci:commands"]; !ok {
		t.Fatalf("expected CI command fact, got keys %v", values)
	}
}

func TestDiscoverPreservesCommandsAcrossMultipleManifests(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n\ngo 1.25\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"build":"vite build","test":"vitest run"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := Discover(root, Metadata{})
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]any{}
	for _, fact := range snapshot.Facts {
		if fact.Category == "commands" {
			values[fact.Key] = fact.Value
		}
	}
	if values["build"] != "vite build" || values["test"] != "vitest run" {
		t.Fatalf("package.json command facts were lost or overwritten by synthetic go.mod commands: %#v", values)
	}
	if _, invented := values["build@go.mod"]; invented {
		t.Fatalf("go.mod must not invent command facts: %#v", values)
	}
}

func TestDiscoverSkipsFixtureSrcAsArchitectureModule(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "module.benchmark-harness", "fixtures", "bug-context-noise-001", "src"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.ts"), []byte("export {}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "module.benchmark-harness", "fixtures", "bug-context-noise-001", "src", "app.ts"), []byte("export {}"), 0600); err != nil {
		t.Fatal(err)
	}

	snapshot, err := Discover(root, Metadata{})
	if err != nil {
		t.Fatal(err)
	}
	for _, fact := range snapshot.Facts {
		if fact.Category == "architecture" && strings.Contains(fact.Key, "fixtures") {
			t.Fatalf("fixture path must not become an architecture module: %+v", fact)
		}
	}
	foundRootSrc := false
	for _, fact := range snapshot.Facts {
		if fact.Category == "architecture" && fact.Key == "module.src" {
			foundRootSrc = true
		}
	}
	if !foundRootSrc {
		t.Fatal("expected top-level src to remain an architecture module")
	}
}

func TestDiscoverDoesNotIndexEveryDevMarkdownAsFact(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "DEV"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"report-a.md", "report-b.md", "report-c.md"} {
		if err := os.WriteFile(filepath.Join(root, "DEV", name), []byte("# report\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# agents\n"), 0600); err != nil {
		t.Fatal(err)
	}

	snapshot, err := Discover(root, Metadata{})
	if err != nil {
		t.Fatal(err)
	}
	sawDevDocs := false
	sawAgents := false
	for _, fact := range snapshot.Facts {
		if strings.Contains(fact.Key, "report-") || strings.HasPrefix(fact.Key, "DEV/") {
			t.Fatalf("DEV markdown must not become a per-file fact: %+v", fact)
		}
		if fact.Category == "conventions" && fact.Key == "dev-docs" {
			sawDevDocs = true
		}
		if fact.Category == "conventions" && fact.Key == "agents" {
			sawAgents = true
		}
	}
	if !sawDevDocs || !sawAgents {
		t.Fatalf("expected compact convention facts, got %+v", snapshot.Facts)
	}
}
