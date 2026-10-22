package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/contextsnapshot"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestEvidenceBundleRequiresAndProjectsCanonicalProof(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}

	// A saturated runner can starve a spawned git child past the per-command
	// deadline while validation evidence is recorded; the observed identity then
	// degrades to the documented deadline-loss INCOMPLETE (contextsnapshot
	// identity.go), which is intentional product behavior and matches the
	// signature whose tolerance contextsnapshot's inspectCodeIdentityRetrying
	// documents for identity assertions. Rebuild the scenario only while the
	// recorded entry carries exactly that signature; anything else fails here.
	const maxAttempts = 5
	for attempt := 1; ; attempt++ {
		bundle, deadlineDegraded := recordCompleteEvidenceBundle(t, n, st, attempt)
		if !bundle.CompletedVerified || !bundle.EvidenceComplete {
			if !deadlineDegraded || attempt >= maxAttempts {
				t.Fatalf("complete verified run must produce a complete bundle: %+v", bundle)
			}
			backoff := time.Duration(attempt) * 500 * time.Millisecond
			if backoff > 2*time.Second {
				backoff = 2 * time.Second
			}
			t.Logf("git identity lost its per-command deadline (attempt %d/%d); rebuilding scenario in %s", attempt, maxAttempts, backoff)
			time.Sleep(backoff)
			continue
		}
		if len(bundle.MissingEvidence) != 0 {
			t.Fatalf("unexpected missing evidence: %v", bundle.MissingEvidence)
		}
		if !bundle.Validation.ChainVerified || len(bundle.Validation.Entries) != 1 {
			t.Fatalf("validation evidence missing: %+v", bundle.Validation)
		}
		if bundle.Validation.Entries[0].Outcome != store.EvidencePass {
			t.Fatalf("validation outcome=%q, want PASS", bundle.Validation.Entries[0].Outcome)
		}
		if len(bundle.Receipts) != 1 || bundle.Receipts[0].Status != "VERIFIED" {
			t.Fatalf("verified receipt missing: %+v", bundle.Receipts)
		}
		return
	}
}

// recordCompleteEvidenceBundle seeds one complete verified mission scenario in
// a fresh repository and records its validation evidence. The boolean reports
// whether the projected bundle degraded with exactly contextsnapshot's
// deadline-loss identity signature; every other outcome stays strict.
func recordCompleteEvidenceBundle(t *testing.T, n *Nexus, st *store.Store, attempt int) (EvidenceBundle, bool) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("evidence bundle"), 0600); err != nil {
		t.Fatal(err)
	}
	git := exec.Command("git", "init")
	git.Dir = root
	if out, err := git.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (%s)", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "test"},
		{"add", "."},
		{"commit", "-m", "init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}

	suffix := strconv.Itoa(attempt)
	project, err := st.CreateProject(store.Project{Name: "Evidence Bundle " + suffix, CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(store.WorkPlan{
		ID: "plan-evidence-bundle-" + suffix, ProjectID: project.ID, Title: "Evidence bundle",
	})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	verification := runner.VerificationResult{Command: "go test ./...", Passed: true, ExitCode: 0}
	run := &runner.MissionRun{
		ID:           "run-evidence-bundle-" + suffix,
		PlanID:       plan.ID,
		PlanRevision: 1,
		ProjectID:    project.ID,
		Workspace:    root,
		State:        runner.StateCompletedVerified,
		PackageRuns: []runner.PackageRun{{
			PackageID:  "package-1",
			Title:      "Build",
			Workspace:  root,
			State:      runner.StateVerified,
			StartedAt:  now,
			FinishedAt: &now,
		}},
		GlobalVerifications: []runner.VerificationResult{verification},
		NeedsHuman:          &runner.HumanIntervention{Resolved: true},
		StartedAt:           now,
		UpdatedAt:           now,
		CompletedAt:         &now,
	}
	repo := newStoreRunRepository(st)
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	evidenceRepo, ok := repo.(runner.EvidenceRepository)
	if !ok {
		t.Fatal("store run repository must implement runner.EvidenceRepository")
	}
	if err := evidenceRepo.SaveWorkReceipt(context.Background(), &runner.WorkReceipt{
		ID:           "receipt-evidence-bundle-" + suffix,
		RunID:        run.ID,
		StepID:       "package-1",
		Status:       "VERIFIED",
		Summary:      "package verified",
		Verification: []runner.VerificationResult{verification},
		StartedAt:    now,
		CompletedAt:  now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := n.recordMissionValidationEvidence(
		context.Background(),
		run,
		"mission/evidence-bundle",
		verification.Command,
		[]runner.VerificationResult{verification},
		"local",
		"test",
		"",
		1,
	); err != nil {
		t.Fatal(err)
	}

	bundle, err := NewRunApplicationService(n).EvidenceBundle(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	deadlineDegraded := len(bundle.Validation.Entries) == 1 &&
		bundle.Validation.Entries[0].GitSHA == "" &&
		bundle.Validation.Entries[0].RepositoryState == string(contextsnapshot.CodeIdentityIncomplete) &&
		bundle.Validation.Entries[0].IdentityDigest == deadlineLostIdentityDigest
	return bundle, deadlineDegraded
}

// deadlineLostIdentityDigest is the constant digest contextsnapshot records
// when a spawned git child loses its per-command deadline: the degraded
// identity digests only fixed fields (repository kind, INCOMPLETE state, and
// the timeout warning), so the digest identifies that exact cause.
var deadlineLostIdentityDigest = func() string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		"git", string(contextsnapshot.CodeIdentityIncomplete), "git identity command timed out",
	}, "\x00")))
	return hex.EncodeToString(sum[:])
}()

func TestEvidenceBundleDoesNotPromoteTerminalStateWithoutProof(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	project, err := st.CreateProject(store.Project{Name: "Evidence Missing", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(store.WorkPlan{
		ID: "plan-evidence-missing", ProjectID: project.ID, Title: "Missing evidence",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	run := &runner.MissionRun{
		ID:           "run-evidence-missing",
		PlanID:       plan.ID,
		PlanRevision: 1,
		ProjectID:    project.ID,
		Workspace:    root,
		State:        runner.StateCompletedVerified,
		PackageRuns: []runner.PackageRun{{
			PackageID: "package-1", Title: "Build", Workspace: root, State: runner.StateVerified,
		}},
		GlobalVerifications: []runner.VerificationResult{{Command: "true", Passed: true}},
		NeedsHuman:          &runner.HumanIntervention{Resolved: false},
		StartedAt:           now,
		UpdatedAt:           now,
		CompletedAt:         &now,
	}
	repo := newStoreRunRepository(st)
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	evidenceRepo, ok := repo.(runner.EvidenceRepository)
	if !ok {
		t.Fatal("store run repository must implement runner.EvidenceRepository")
	}
	if err := evidenceRepo.SaveWorkReceipt(context.Background(), &runner.WorkReceipt{
		ID: "receipt-evidence-missing", RunID: run.ID, StepID: "package-1",
		Status: "VERIFIED", Summary: "package verified", CompletedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	bundle, err := NewRunApplicationService(n).EvidenceBundle(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.EvidenceComplete {
		t.Fatalf("terminal state without validation stream must not become complete evidence: %+v", bundle)
	}
	for _, required := range []string{
		"validation_chain_verified",
		"validation_entries",
		"unresolved_human_intervention",
	} {
		if !slices.Contains(bundle.MissingEvidence, required) {
			t.Fatalf("missing_evidence=%v, want %q", bundle.MissingEvidence, required)
		}
	}
}
