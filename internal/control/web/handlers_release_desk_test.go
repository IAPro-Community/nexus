package web

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

// releaseDeskFixture builds a repo with two commits on top of an initial one,
// one referencing a PR and another referencing an issue, and returns
// (repoPath, baseSHA, headSHA).
func releaseDeskFixture(t *testing.T) (string, string, string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return string(out)
	}
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q", "-b", "main")
	write("a.txt", "one\n")
	run("add", ".")
	run("commit", "-q", "-m", "initial commit")
	base := stringsTrim(run("rev-parse", "HEAD"))

	write("a.txt", "two\n")
	run("add", ".")
	run("commit", "-q", "-m", "feat: companion surface (#55)")
	write("a.txt", "three\n")
	run("add", ".")
	run("commit", "-q", "-m", "fix: parking honesty, issue #114")
	head := stringsTrim(run("rev-parse", "HEAD"))
	return repo, base, head
}

func stringsTrim(s string) string {
	out := ""
	for _, r := range s {
		if r != '\n' && r != '\r' {
			out += string(r)
		}
	}
	return out
}

func TestProjectReleaseDeskLinksRangeAndBlocksIntegration(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	base := srv.URL()
	repo, baseSHA, headSHA := releaseDeskFixture(t)
	projectID := createCollisionProject(t, client, base, repo, csrf)

	resp, err := client.Get(base + "/api/v1/projects/" + projectID + "/release-desk?base=" + baseSHA)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET release-desk status=%d", resp.StatusCode)
	}
	var report nexus.ReleaseDeskReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	if report.Branch != "main" || report.BaseSHA != baseSHA || report.HeadSHA != headSHA {
		t.Fatalf("range mismatch: %+v", report)
	}
	if len(report.Commits) != 2 {
		t.Fatalf("commits = %d, want 2 (initial excluded by merge-base)", len(report.Commits))
	}
	if len(report.PullRequests) != 1 || report.PullRequests[0] != "#55" {
		t.Fatalf("pull requests = %v, want [#55]", report.PullRequests)
	}
	if len(report.Issues) != 1 || report.Issues[0] != "#114" {
		t.Fatalf("issues = %v, want [#114]", report.Issues)
	}
	if report.MergePublishAllowed || len(report.BlockReasons) == 0 {
		t.Fatalf("integration must be blocked without approval: %+v", report)
	}
	hasApprovalReason := false
	for _, reason := range report.BlockReasons {
		if reason == "no recorded approval for this release" {
			hasApprovalReason = true
		}
	}
	if !hasApprovalReason {
		t.Fatalf("missing approval blocker: %v", report.BlockReasons)
	}
	if len(report.Gates) == 0 {
		t.Fatal("canonical gates must be seeded")
	}
}

func TestProjectReleaseDeskRejectsInvalidBaseRef(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	base := srv.URL()
	repo, _, _ := releaseDeskFixture(t)
	projectID := createCollisionProject(t, client, base, repo, csrf)
	req, _ := http.NewRequest(http.MethodGet,
		base+"/api/v1/projects/"+projectID+"/release-desk?base=main%3Brm", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid ref must 400, got %d", resp.StatusCode)
	}
	_ = csrf
}

func TestProjectReleaseDeskRejectsNonGet(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	base := srv.URL()
	repo, _, _ := releaseDeskFixture(t)
	projectID := createCollisionProject(t, client, base, repo, csrf)
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/projects/"+projectID+"/release-desk", nil)
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("POST must never project release desk: got %d", resp.StatusCode)
	}
}

func TestProjectReleaseDeskHonorsSignedApproval(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	base := srv.URL()
	repo, baseSHA, headSHA := releaseDeskFixture(t)
	projectID := createCollisionProject(t, client, base, repo, csrf)

	// Gate: verifyReleaseGate via artifact file + trust store must be honored.
	// Refuse to fabricate: only gates marked verified with evidence unlock.
	resp, err := client.Get(base + "/api/v1/projects/" + projectID + "/release-desk?base=" + baseSHA)
	if err != nil {
		t.Fatal(err)
	}
	var report nexus.ReleaseDeskReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if report.MergePublishAllowed {
		t.Fatal("gates are unverified by default; integration must stay blocked")
	}

	// Now enroll a key, sign an approval bound to headSHA and drop both files
	// into the repo. Gates remain unverified, so integration must STILL be
	// blocked (approval alone is not enough) — and the SHA-mismatch reason
	// must disappear.
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pemBlock, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	trust := map[string]any{
		"schemaVersion": 1,
		"reviewers": []map[string]string{{
			"id":           "owner",
			"publicKeyPem": string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pemBlock})),
		}},
	}
	trustRaw, err := json.Marshal(trust)
	if err != nil {
		t.Fatal(err)
	}
	referenceDir := filepath.Join(repo, "docs", "reference")
	if err := os.MkdirAll(referenceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(referenceDir, "release-reviewers.json"), trustRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{
		"approvalId":    "GO-TEST-1",
		"approvedAtUtc": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"approver":      "maestro",
		"schemaVersion": 1,
		"sourceSha":     headSHA,
	}
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var canonical []byte
	canonical = append(canonical, '{')
	for i, key := range keys {
		if i > 0 {
			canonical = append(canonical, ',')
		}
		keyJSON, _ := json.Marshal(key)
		canonical = append(canonical, keyJSON...)
		canonical = append(canonical, ':')
		switch value := payload[key].(type) {
		case string:
			valueJSON, _ := json.Marshal(value)
			canonical = append(canonical, valueJSON...)
		case int:
			canonical = append(canonical, []byte(strconv.Itoa(value))...)
		}
	}
	canonical = append(canonical, '}')
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, canonical))
	approval := map[string]any{}
	for key, value := range payload {
		approval[key] = value
	}
	approval["signature"] = map[string]any{"keyId": "owner", "base64": signature}
	approvalRaw, err := json.Marshal(approval)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(referenceDir, "release-approval.json"), approvalRaw, 0o600); err != nil {
		t.Fatal(err)
	}

	resp, err = client.Get(base + "/api/v1/projects/" + projectID + "/release-desk?base=" + baseSHA)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	if report.Approval == nil || report.Approval.Approver != "maestro" {
		t.Fatalf("signed approval must surface in the report: %+v", report.Approval)
	}
	shaMismatch := false
	for _, reason := range report.BlockReasons {
		if strings.Contains(reason, "approval is bound to") {
			shaMismatch = true
		}
	}
	if shaMismatch {
		t.Fatalf("approval bound to headSHA must clear the mismatch reason: %v", report.BlockReasons)
	}
	if report.MergePublishAllowed {
		t.Fatal("unverified gates must still block integration even with approval")
	}
}
