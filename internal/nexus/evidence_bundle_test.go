package nexus

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestEvidenceBundleRequiresAndProjectsCanonicalProof(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}

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

	project, err := st.CreateProject(store.Project{Name: "Evidence Bundle", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(store.WorkPlan{
		ID: "plan-evidence-bundle", ProjectID: project.ID, Title: "Evidence bundle",
	})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	verification := runner.VerificationResult{Command: "go test ./...", Passed: true, ExitCode: 0}
	run := &runner.MissionRun{
		ID:           "run-evidence-bundle",
		PlanID:       plan.ID,
		PlanRevision: 1,
		ProjectID:    project.ID,
		Workspace:    root,
		State:        runner.StateCompletedVerified,
		PackageRuns: []runner.PackageRun{{
			PackageID: "package-1",
			Title:     "Build",
			Workspace: root,
			State:     runner.StateVerified,
			StartedAt: now,
			FinishedAt: &now,
		}},
		GlobalVerifications: []runner.VerificationResult{verification},
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
		ID:          "receipt-evidence-bundle",
		RunID:       run.ID,
		StepID:      "package-1",
		Status:      "VERIFIED",
		Summary:     "package verified",
		Verification: []runner.VerificationResult{verification},
		StartedAt:   now,
		CompletedAt: now,
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
	if !bundle.CompletedVerified || !bundle.EvidenceComplete {
		t.Fatalf("complete verified run must produce a complete bundle: %+v", bundle)
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
}

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
		StartedAt: now,
		UpdatedAt: now,
		CompletedAt: &now,
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
	} {
		if !slices.Contains(bundle.MissingEvidence, required) {
			t.Fatalf("missing_evidence=%v, want %q", bundle.MissingEvidence, required)
		}
	}
}
