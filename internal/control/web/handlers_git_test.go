package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func initTestGitRepo(t testing.TB, dir string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("commit", "--allow-empty", "-m", "init")
}

func TestReadGitBranchFastRepoWorktreeDetachedAndMissing(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	root := t.TempDir()
	if got := readGitBranchFast(root); got != "" {
		t.Fatalf("expected empty branch for non-git dir, got %q", got)
	}

	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	initTestGitRepo(t, repo)

	if got := readGitBranchFast(repo); got != "main" {
		t.Fatalf("readGitBranchFast(repo) = %q, want main", got)
	}

	linked := filepath.Join(root, "wt-fast")
	cmd := exec.Command("git", "-C", repo, "worktree", "add", "-b", "feat/fast-wt", linked)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v: %s", err, out)
	}
	if got := readGitBranchFast(linked); got != "feat/fast-wt" {
		t.Fatalf("readGitBranchFast(linked) = %q, want feat/fast-wt", got)
	}

	headOut, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	sha := strings.TrimSpace(string(headOut))
	if out, err := exec.Command("git", "-C", repo, "checkout", "--detach", sha).CombinedOutput(); err != nil {
		t.Fatalf("git checkout --detach: %v: %s", err, out)
	}
	if got := readGitBranchFast(repo); got != sha[:7] {
		t.Fatalf("readGitBranchFast(detached) = %q, want %q", got, sha[:7])
	}
}

func TestHandleProjectGitBranchesSummarySkipsFullEnumeration(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NEXUS_REAL_HOME", "")

	repo := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	initTestGitRepo(t, repo)

	st, err := nexus.Default().OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	proj, err := st.CreateProject(store.Project{
		Name:          "git-summary-test",
		CanonicalPath: repo,
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatal(err)
	}

	handler := NewNexusHandler(nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+proj.ID+"/git/branches?summary=1", nil)
	handler.handleProjectGitBranches(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("summary status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var resp ProjectGitBranchesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.CurrentBranch != "main" {
		t.Fatalf("CurrentBranch = %q, want main", resp.CurrentBranch)
	}
	if len(resp.Branches) != 1 || resp.Branches[0] != "main" || resp.WorktreesAvailable {
		t.Fatalf("unexpected summary response: %+v", resp)
	}
}

func BenchmarkHandleProjectGitBranchesSummary(b *testing.B) {
	if _, err := exec.LookPath("git"); err != nil {
		b.Skip("git not installed")
	}
	b.Setenv("HOME", b.TempDir())
	b.Setenv("NEXUS_REAL_HOME", "")

	repo := filepath.Join(b.TempDir(), "bench-proj")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		b.Fatal(err)
	}
	initTestGitRepo(b, repo)

	st, err := nexus.Default().OpenProject()
	if err != nil {
		b.Fatal(err)
	}
	proj, err := st.CreateProject(store.Project{
		Name:          "git-bench",
		CanonicalPath: repo,
		DefaultBranch: "main",
	})
	if err != nil {
		b.Fatal(err)
	}

	handler := NewNexusHandler(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+proj.ID+"/git/branches?summary=1", nil)
		handler.handleProjectGitBranches(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status=%d", rec.Code)
		}
	}
}

func BenchmarkHandleProjectGitBranchesFull(b *testing.B) {
	if _, err := exec.LookPath("git"); err != nil {
		b.Skip("git not installed")
	}
	b.Setenv("HOME", b.TempDir())
	b.Setenv("NEXUS_REAL_HOME", "")

	repo := filepath.Join(b.TempDir(), "bench-proj-full")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		b.Fatal(err)
	}
	initTestGitRepo(b, repo)

	st, err := nexus.Default().OpenProject()
	if err != nil {
		b.Fatal(err)
	}
	proj, err := st.CreateProject(store.Project{
		Name:          "git-bench-full",
		CanonicalPath: repo,
		DefaultBranch: "main",
	})
	if err != nil {
		b.Fatal(err)
	}

	handler := NewNexusHandler(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+proj.ID+"/git/branches", nil)
		handler.handleProjectGitBranches(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status=%d", rec.Code)
		}
	}
}

func TestHandleProjectGitDiffAndWorktreeValidation(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NEXUS_REAL_HOME", "")

	root := t.TempDir()
	repo := filepath.Join(root, "proj-diff")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	initTestGitRepo(t, repo)

	if err := os.WriteFile(filepath.Join(repo, "app.go"), []byte("package main\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	addCommit := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	addCommit("add", "app.go")
	addCommit("commit", "-m", "add app.go")

	st, err := nexus.Default().OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	proj, err := st.CreateProject(store.Project{
		Name:          "git-diff-test",
		CanonicalPath: repo,
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewNexusHandler(nil)

	// 1. Clean repo
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+proj.ID+"/git/diff", nil)
	handler.handleProjectGitDiff(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("clean diff status=%d body=%s", rec.Code, rec.Body.String())
	}
	var cleanResp ProjectGitDiffResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &cleanResp); err != nil {
		t.Fatal(err)
	}
	if !cleanResp.IsClean || len(cleanResp.Files) != 0 {
		t.Fatalf("expected clean diff, got %+v", cleanResp)
	}

	// 2. Dirty modification + untracked file
	if err := os.WriteFile(filepath.Join(repo, "app.go"), []byte("package main\nfunc main() { println(1) }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "notes.txt"), []byte("todo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+proj.ID+"/git/diff", nil)
	handler.handleProjectGitDiff(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("dirty diff status=%d body=%s", rec.Code, rec.Body.String())
	}
	var dirtyResp ProjectGitDiffResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &dirtyResp); err != nil {
		t.Fatal(err)
	}
	if dirtyResp.IsClean || len(dirtyResp.Files) != 2 || !strings.Contains(dirtyResp.DiffText, "println(1)") {
		t.Fatalf("unexpected dirty diff response: %+v", dirtyResp)
	}

	// 3. Invalid external worktree must be rejected with 400 Bad Request
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+proj.ID+"/git/diff?worktree=/etc", nil)
	handler.handleProjectGitDiff(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for external worktree path, got %d", rec.Code)
	}
}
