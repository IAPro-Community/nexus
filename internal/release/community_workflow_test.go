package release

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestForkPullRequestsNeverReachPrivilegedRunners protects the security
// boundary between untrusted fork patches and persistent release infrastructure.
// It does not replace repository branch protections or a real fork PR smoke test.
func TestForkPullRequestsNeverReachPrivilegedRunners(t *testing.T) {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve source location")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(self), "..", ".."))
	read := func(path string) string {
		b, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		return string(b)
	}
	privileged := read(".github/workflows/ci.yml")
	community := read(".github/workflows/community-pr.yml")

	for _, line := range strings.Split(privileged, "\n") {
		if strings.TrimSpace(line) == "pull_request:" || strings.TrimSpace(line) == "pull_request_target:" {
			t.Fatal("self-hosted CI must not be triggered by untrusted PRs")
		}
	}
	if !strings.Contains(privileged, "runs-on: [self-hosted,") {
		t.Fatal("expected the separate privileged CI workflow to remain present")
	}

	sawPullRequest, sawReadOnly, sawHosted := false, false, false
	for _, line := range strings.Split(community, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "pull_request:" {
			sawPullRequest = true
		}
		if trim == "pull_request_target:" {
			t.Fatal("never use pull_request_target for untrusted checkout")
		}
		if strings.HasPrefix(trim, "runs-on:") {
			if strings.Contains(trim, "self-hosted") {
				t.Fatal("community PR jobs cannot run on persistent self-hosted runners")
			}
			if strings.Contains(trim, "ubuntu-24.04") {
				sawHosted = true
			}
		}
		if trim == "contents: read" {
			sawReadOnly = true
		}
		if trim == "contents: write" || trim == "id-token: write" || trim == "actions: write" {
			t.Fatalf("community PR has privileged token permission: %s", trim)
		}
	}
	if !sawPullRequest || !sawReadOnly || !sawHosted {
		t.Fatalf("missing secure community workflow essentials: PR=%v readOnly=%v hosted=%v", sawPullRequest, sawReadOnly, sawHosted)
	}
	if !strings.Contains(community, "persist-credentials: false") {
		t.Fatal("community checkout must not persist GitHub credentials")
	}
	if strings.Contains(community, "${{ secrets.") {
		t.Fatal("untrusted pull requests cannot be supplied release secrets")
	}
}
