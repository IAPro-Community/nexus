package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

// collisionFixture builds a git repository with the primary worktree plus one
// linked worktree, applies the given edits (file, line, content) to each, and
// returns the repo path. Edits are left uncommitted so the detector scans
// uncommitted changes against HEAD, mirroring how agents actually diverge.
func collisionFixture(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	linked := filepath.Join(root, "linked")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v in %s: %v: %s", args, dir, err, out)
		}
	}
	body := ""
	for i := 1; i <= 30; i++ {
		body += "line " + string(rune('0'+i%10)) + "\n"
	}
	if err := os.WriteFile(filepath.Join(repo, "shared.go"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	run(repo, "init", "-q", "-b", "main")
	run(repo, "add", ".")
	run(repo, "commit", "-q", "-m", "initial")
	run(repo, "worktree", "add", "-b", "feature/parallel", linked)
	return root
}

func overwriteLine(t *testing.T, file string, line int, content string) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	lines := splitLines(string(data))
	if line < 1 || line > len(lines) {
		t.Fatalf("line %d out of range (%d lines)", line, len(lines))
	}
	lines[line-1] = content
	if err := os.WriteFile(file, []byte(joinLines(lines)), 0o600); err != nil {
		t.Fatal(err)
	}
}

func splitLines(s string) []string {
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func joinLines(lines []string) string {
	out := ""
	for _, line := range lines {
		out += line + "\n"
	}
	return out
}

func createCollisionProject(t *testing.T, client *http.Client, srvURL, repo, csrf string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"name": "Collision Fixture", "path": repo})
	req, _ := http.NewRequest(http.MethodPost, srvURL+"/api/v1/projects", bytes.NewReader(body))
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create project status=%d", resp.StatusCode)
	}
	var project struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		t.Fatal(err)
	}
	return project.ID
}

func TestProjectChangeCollisionsDetectsOverlappingHunks(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	base := srv.URL()
	root := collisionFixture(t)
	repo := filepath.Join(root, "repo")
	linked := filepath.Join(root, "linked")
	overwriteLine(t, filepath.Join(repo, "shared.go"), 14, "edited by primary")
	overwriteLine(t, filepath.Join(linked, "shared.go"), 14, "edited by linked")

	projectID := createCollisionProject(t, client, base, repo, csrf)
	resp, err := client.Get(base + "/api/v1/projects/" + projectID + "/change-collisions")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET change-collisions status=%d", resp.StatusCode)
	}
	var report nexus.ChangeCollisionReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	if len(report.Collisions) != 1 {
		t.Fatalf("expected 1 collision, got %+v", report.Collisions)
	}
	collision := report.Collisions[0]
	if collision.File != "shared.go" || collision.Confidence != nexus.CollisionHunkOverlap || collision.Uncertain {
		t.Fatalf("wrong collision: %+v", collision)
	}
	if len(collision.Worktrees) != 2 {
		t.Fatalf("collision must involve both worktrees: %+v", collision)
	}
	if len(report.Worktrees) != 2 {
		t.Fatalf("both worktrees must be listed as scanned: %+v", report.Worktrees)
	}
	if report.UncertainCount != 0 || len(report.ScanErrors) != 0 {
		t.Fatalf("clean overlap scan must have no uncertainty or errors: %+v", report)
	}
}

func TestProjectChangeCollisionsFlagsDisjointEditsAsUncertain(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	base := srv.URL()
	root := collisionFixture(t)
	repo := filepath.Join(root, "repo")
	linked := filepath.Join(root, "linked")
	overwriteLine(t, filepath.Join(repo, "shared.go"), 2, "edited near top")
	overwriteLine(t, filepath.Join(linked, "shared.go"), 28, "edited near bottom")

	projectID := createCollisionProject(t, client, base, repo, csrf)
	resp, err := client.Get(base + "/api/v1/projects/" + projectID + "/change-collisions")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET change-collisions status=%d", resp.StatusCode)
	}
	var report nexus.ChangeCollisionReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	if len(report.Collisions) != 1 {
		t.Fatalf("expected 1 candidate, got %+v", report.Collisions)
	}
	collision := report.Collisions[0]
	if collision.Confidence != nexus.CollisionSameFile || !collision.Uncertain {
		t.Fatalf("disjoint edits must be uncertain same-file candidates: %+v", collision)
	}
	if report.UncertainCount != 1 {
		t.Fatalf("uncertain count must be surfaced: %+v", report)
	}
}

func TestProjectChangeCollisionsRejectsNonGet(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	base := srv.URL()
	root := collisionFixture(t)
	projectID := createCollisionProject(t, client, base, filepath.Join(root, "repo"), csrf)
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/projects/"+projectID+"/change-collisions", bytes.NewReader(nil))
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("POST must never scan: got %d", resp.StatusCode)
	}
}
