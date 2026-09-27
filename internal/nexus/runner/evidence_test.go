package runner

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type evidenceExecutor struct {
	fakeExecutor
	compileCapsules map[string]*ContextCapsule
}

func (e *evidenceExecutor) Compile(_ context.Context, _ *MissionRun, pkg *PackageRun) (PromptArtifact, error) {
	if e.compileCapsules == nil {
		e.compileCapsules = map[string]*ContextCapsule{}
	}
	if pkg.ContextCapsule == nil {
		return PromptArtifact{}, ErrMissingContextCapsule
	}
	copy := *pkg.ContextCapsule
	e.compileCapsules[pkg.PackageID] = &copy
	e.compiled++
	return PromptArtifact{VersionID: "prompt-" + pkg.PackageID, Content: "implement " + pkg.PackageID}, nil
}

func (e *evidenceExecutor) Execute(_ context.Context, _ *MissionRun, pkg *PackageRun, _ string) (ExecutionResult, error) {
	if pkg.ContextCapsule == nil {
		return ExecutionResult{}, ErrMissingContextCapsule
	}
	e.executed++
	return ExecutionResult{RuntimeID: "rt-" + pkg.PackageID, Output: "TRANSCRIPT_SENTINEL_" + pkg.PackageID}, nil
}

func TestMissionRunnerPersistsContextCapsuleBeforeCompileAndHandsOffReceipts(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &evidenceExecutor{fakeExecutor: fakeExecutor{reviewOK: true}}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{verificationSuccessCommand()}
	plan := PlanSpec{ID: "flow", ProjectID: "project", Revision: 9, Packages: []PackageSpec{
		{ID: "A", PhaseID: "p", Title: "A", Goal: "A goal", RelevantPaths: []string{"internal/a"}, AcceptanceCriteria: []string{"A done"}},
		{ID: "B", PhaseID: "p", Title: "B", Goal: "B goal", Dependencies: []string{"A"}, RelevantPaths: []string{"internal/b"}, AcceptanceCriteria: []string{"B done"}},
	}}
	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30 && run.State != StateCompletedVerified; i++ {
		run, _, err = r.ExecuteNextStep(context.Background(), run.ID)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	if run.State != StateCompletedVerified {
		t.Fatalf("state=%s", run.State)
	}

	a := run.PackageRuns[0]
	b := run.PackageRuns[1]
	if a.ContextCapsule == nil || a.ContextCapsule.FlowRevision != 9 || a.ContextCapsule.Step.ID != "A" {
		t.Fatalf("A capsule missing/invalid: %+v", a.ContextCapsule)
	}
	if a.WorkReceipt == nil || a.WorkReceipt.Status != "VERIFIED" || a.WorkReceipt.AgentID == "" {
		t.Fatalf("A receipt missing/invalid: %+v", a.WorkReceipt)
	}
	if len(a.WorkReceipt.Verification) == 0 || !a.WorkReceipt.Verification[0].Passed {
		t.Fatalf("A verification evidence missing: %+v", a.WorkReceipt)
	}
	if b.ContextCapsule == nil || len(b.ContextCapsule.DependencyReceipts) != 1 || b.ContextCapsule.DependencyReceipts[0].StepID != "A" {
		t.Fatalf("B did not receive direct A receipt: %+v", b.ContextCapsule)
	}
	encoded, _ := json.Marshal(b.ContextCapsule)
	if strings.Contains(string(encoded), "TRANSCRIPT_SENTINEL") {
		t.Fatalf("raw provider output leaked into dependency capsule: %s", encoded)
	}

	reloaded, err := r.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.PackageRuns[0].ContextCapsule == nil || reloaded.PackageRuns[0].WorkReceipt == nil {
		t.Fatalf("evidence did not survive durable reload: %+v", reloaded.PackageRuns[0])
	}
}

func TestMissionRunnerCreatesFactualFailedReceiptWithoutInventingEvidence(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &evidenceExecutor{fakeExecutor: fakeExecutor{reviewOK: true}}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.MaxRetries = 1
	contract.MaxNoProgress = 1
	contract.VerificationCommands = []string{verificationFailureCommand()}
	plan := PlanSpec{ID: "flow-fail", ProjectID: "project", Revision: 1, Packages: []PackageSpec{{ID: "A", PhaseID: "p", Title: "A", Goal: "fail"}}}
	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		updated, _, stepErr := r.ExecuteNextStep(context.Background(), run.ID)
		if updated != nil {
			run = updated
		}
		if run.PackageRuns[0].State == StateFailed {
			break
		}
		if stepErr != nil && run.PackageRuns[0].State != StateRemediating {
			break
		}
	}
	receipt := run.PackageRuns[0].WorkReceipt
	if receipt == nil || receipt.Status != "FAILED" {
		t.Fatalf("failed receipt missing: %+v", receipt)
	}
	if len(receipt.Artifacts) != 0 || len(receipt.Decisions) != 0 {
		t.Fatalf("receipt invented artifacts/decisions: %+v", receipt)
	}
	if len(receipt.RemainingIssues) == 0 {
		t.Fatalf("failed receipt must retain factual failure issue: %+v", receipt)
	}
}

func TestRenderContextCapsuleIncludesReceiptsAndExcludesTranscriptSemantics(t *testing.T) {
	capsule := &ContextCapsule{FlowID: "flow", FlowRevision: 3, Branch: "main", Head: "abc", Step: ContextCapsuleStep{ID: "B", Title: "Backend", Goal: "Implement", Dependencies: []string{"A"}}, RelevantPaths: []string{"internal"}, DependencyReceipts: []WorkReceipt{{StepID: "A", Status: "VERIFIED", Summary: "A ended in VERIFIED", ChangedFiles: []string{"a.go"}, Commands: []string{"go test ./..."}}}, AcceptanceCriteria: []string{"done"}, Constraints: []string{"no push"}}
	rendered := RenderContextCapsule(capsule)
	for _, want := range []string{"Flow flow revision 3", "Step B", "Dependency Work Receipts", "A ended in VERIFIED", "a.go", "go test ./...", "Acceptance criteria", "no push"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered capsule missing %q: %s", want, rendered)
		}
	}
	if strings.Contains(strings.ToLower(rendered), "transcript") {
		t.Fatalf("rendered capsule must not request/embed transcript: %s", rendered)
	}
}

func TestReceiptChangedFilesIncludesCommittedPathWithSpaces(t *testing.T) {
	workspace := t.TempDir()
	commands := [][]string{
		{"git", "init", "-b", "main", workspace},
		{"git", "-C", workspace, "config", "user.email", "nexus-test@example.invalid"},
		{"git", "-C", workspace, "config", "user.name", "Nexus Test"},
	}
	for _, args := range commands {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("%v: %v: %s", args, err, out)
		}
	}
	path := filepath.Join(workspace, "file with space.txt")
	if err := os.WriteFile(path, []byte("one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"git", "-C", workspace, "add", "."}, {"git", "-C", workspace, "commit", "-m", "base"}} {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("%v: %v: %s", args, err, out)
		}
	}
	base := gitText(context.Background(), workspace, "rev-parse", "HEAD")
	if err := os.WriteFile(path, []byte("two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"git", "-C", workspace, "add", "."}, {"git", "-C", workspace, "commit", "-m", "change"}} {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("%v: %v: %s", args, err, out)
		}
	}
	result := gitText(context.Background(), workspace, "rev-parse", "HEAD")
	got := receiptChangedFiles(context.Background(), workspace, base, result, map[string]string{}, map[string]string{})
	if len(got) != 1 || got[0] != "file with space.txt" {
		t.Fatalf("committed change lost or path split: %#v", got)
	}
}
