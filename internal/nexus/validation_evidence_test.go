package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestMissionValidationEvidenceBindsIdentityAndVerifiesChain(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "evidence", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	run := &runner.MissionRun{ID: "run-evidence", PlanID: "plan", PlanRevision: 3, ProjectID: project.ID, TotalIterations: 1}
	result := runner.VerificationResult{Command: "go test ./...", Passed: true, ExitCode: 0}
	if err := n.recordMissionValidationEvidence(context.Background(), run, "mission/test", result.Command, []runner.VerificationResult{result}, "OpenCode", "profile-a", "", 1); err != nil {
		t.Fatal(err)
	}
	stream, err := st.GetValidationEvidenceStreamByName(project.ID, missionEvidenceStreamName)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.VerifyValidationEvidenceChain(stream.ID); err != nil {
		t.Fatal(err)
	}
	entries, err := st.ListValidationEvidenceEntries(stream.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].RepositoryState != "NON_GIT" || entries[0].IdentityDigest == "" || entries[0].Confidence != store.EvidenceObserved || entries[0].Outcome != store.EvidenceNotVerified {
		t.Fatalf("unexpected evidence entry: %+v", entries)
	}
	var evidence map[string]any
	if err := json.Unmarshal([]byte(entries[0].EvidenceJSON), &evidence); err != nil {
		t.Fatal(err)
	}
}

func TestStableEvidenceIDIsIdempotent(t *testing.T) {
	first := stableEvidenceID("run", "scenario", 1, 1)
	second := stableEvidenceID("run", "scenario", 1, 1)
	if first != second {
		t.Fatal("evidence id must be stable")
	}
	if first == stableEvidenceID("run", "scenario", 2, 1) {
		t.Fatal("different attempts must produce distinct evidence ids")
	}
}

func TestMissionValidationEvidenceRecordsFailedResultsAsFail(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	// Initialize a git repo so HeadSHA is present and FAIL can be recorded.
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
	project, err := st.CreateProject(store.Project{Name: "fail-evidence", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	run := &runner.MissionRun{ID: "run-fail", PlanID: "plan", PlanRevision: 1, ProjectID: project.ID}
	result := runner.VerificationResult{Command: "go test ./...", Passed: false, ExitCode: 1}
	if err := n.recordMissionValidationEvidence(context.Background(), run, "mission/test", result.Command, []runner.VerificationResult{result}, "OpenCode", "profile-a", "", 1); err != nil {
		t.Fatal(err)
	}
	stream, err := st.GetValidationEvidenceStreamByName(project.ID, missionEvidenceStreamName)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := st.ListValidationEvidenceEntries(stream.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Outcome != store.EvidenceFail || entries[0].Confidence != store.EvidenceObserved {
		t.Fatalf("unexpected fail evidence: %+v", entries)
	}
}

func TestMissionValidationEvidenceFailedNonGitIsStillFail(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "fail-nongit", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	run := &runner.MissionRun{ID: "run-fail-nongit", PlanID: "plan", PlanRevision: 1, ProjectID: project.ID}
	result := runner.VerificationResult{Command: "go test ./...", Passed: false, ExitCode: 1}
	if err := n.recordMissionValidationEvidence(context.Background(), run, "mission/test", result.Command, []runner.VerificationResult{result}, "OpenCode", "profile-a", "", 1); err != nil {
		t.Fatal(err)
	}
	stream, err := st.GetValidationEvidenceStreamByName(project.ID, missionEvidenceStreamName)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := st.ListValidationEvidenceEntries(stream.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].RepositoryState != "NON_GIT" || entries[0].Outcome != store.EvidenceFail {
		t.Fatalf("failed verification in non-git must be FAIL, got: %+v", entries)
	}
}

func TestMissionValidationEvidenceCapturesBoundedToolchainEnvironment(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"evidence-fixture"}`), 0600); err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "toolchain-evidence", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	run := &runner.MissionRun{ID: "run-toolchain", PlanID: "plan", PlanRevision: 1, ProjectID: project.ID}
	result := runner.VerificationResult{Command: "true", Passed: true, ExitCode: 0}
	if err := n.recordMissionValidationEvidence(context.Background(), run, "mission/toolchain", result.Command, []runner.VerificationResult{result}, "local", "test", "economy", 1); err != nil {
		t.Fatal(err)
	}
	stream, err := st.GetValidationEvidenceStreamByName(project.ID, missionEvidenceStreamName)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := st.ListValidationEvidenceEntries(stream.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one evidence entry, got %d", len(entries))
	}
	var environment map[string]string
	if err := json.Unmarshal([]byte(entries[0].EnvironmentJSON), &environment); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"go", "os", "arch"} {
		if environment[key] == "" {
			t.Fatalf("environment missing %q: %#v", key, environment)
		}
	}
	if environment["package_manager"] != "npm" {
		t.Fatalf("package manager = %q, want npm: %#v", environment["package_manager"], environment)
	}
	if _, err := exec.LookPath("node"); err == nil && environment["node"] == "" {
		t.Fatalf("node is available but version was not captured: %#v", environment)
	}
	if _, err := exec.LookPath("npm"); err == nil && environment["package_manager_version"] == "" {
		t.Fatalf("npm is available but version was not captured: %#v", environment)
	}
}

func TestRunApplicationProjectsCanonicalValidationEvidence(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	project, err := st.CreateProject(store.Project{Name: "evidence-report", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateWorkPlan(store.WorkPlan{ID: "evidence-report-plan", ProjectID: project.ID, Title: "Evidence report"}); err != nil {
		t.Fatal(err)
	}
	run, err := n.Runner().StartMissionRun(context.Background(), runner.PlanSpec{
		ID: "evidence-report-plan", ProjectID: project.ID, Revision: 1,
		Packages: []runner.PackageSpec{{ID: "package-1", Title: "Evidence"}},
	}, root, runner.DefaultAutonomyContract(), "")
	if err != nil {
		t.Fatal(err)
	}
	result := runner.VerificationResult{Command: "true", Passed: true, ExitCode: 0}
	if err := n.recordMissionValidationEvidence(context.Background(), run, "mission/report", result.Command, []runner.VerificationResult{result}, "local", "test", "economy", 1); err != nil {
		t.Fatal(err)
	}
	report, err := NewRunApplicationService(n).ValidationEvidence(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if report.RunID != run.ID || report.Stream == nil || !report.ChainVerified || len(report.Entries) != 1 {
		t.Fatalf("canonical evidence report = %+v", report)
	}
	if report.Entries[0].Scenario != "mission/report" || report.Entries[0].Outcome != store.EvidenceNotVerified {
		t.Fatalf("unexpected projected evidence entry: %+v", report.Entries[0])
	}
}

func TestRunApplicationProjectsOnlyEvidenceForRequestedMission(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	project, err := st.CreateProject(store.Project{Name: "evidence-isolation", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	repo := newStoreRunRepository(st)
	runA := &runner.MissionRun{ID: "run-evidence-a", PlanID: "plan-a", PlanRevision: 1, ProjectID: project.ID}
	runB := &runner.MissionRun{ID: "run-evidence-b", PlanID: "plan-b", PlanRevision: 1, ProjectID: project.ID}
	for _, planID := range []string{"plan-a", "plan-b"} {
		if _, err := st.CreateWorkPlan(store.WorkPlan{ID: planID, ProjectID: project.ID, Title: planID}); err != nil {
			t.Fatal(err)
		}
	}
	for _, run := range []*runner.MissionRun{runA, runB} {
		if err := repo.SaveRun(context.Background(), run); err != nil {
			t.Fatal(err)
		}
	}
	for _, fixture := range []struct {
		run      *runner.MissionRun
		scenario string
	}{
		{run: runA, scenario: "mission/a"},
		{run: runB, scenario: "mission/b"},
	} {
		result := runner.VerificationResult{Command: "true", Passed: true, ExitCode: 0}
		if err := n.recordMissionValidationEvidence(context.Background(), fixture.run, fixture.scenario, result.Command, []runner.VerificationResult{result}, "local", "test", "economy", 1); err != nil {
			t.Fatal(err)
		}
	}

	service := NewRunApplicationService(n)
	for _, fixture := range []struct {
		run      *runner.MissionRun
		scenario string
	}{
		{run: runA, scenario: "mission/a"},
		{run: runB, scenario: "mission/b"},
	} {
		report, err := service.ValidationEvidence(context.Background(), fixture.run.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !report.ChainVerified || len(report.Entries) != 1 {
			t.Fatalf("mission %s received non-isolated evidence: %+v", fixture.run.ID, report)
		}
		if report.Entries[0].Scenario != fixture.scenario {
			t.Fatalf("mission %s received scenario %q, want %q", fixture.run.ID, report.Entries[0].Scenario, fixture.scenario)
		}
	}
}

func TestRunApplicationRejectsMissionEvidenceWithoutRunID(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "evidence-metadata", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateWorkPlan(store.WorkPlan{ID: "plan-metadata", ProjectID: project.ID, Title: "Metadata"}); err != nil {
		t.Fatal(err)
	}
	run := &runner.MissionRun{ID: "run-metadata", PlanID: "plan-metadata", PlanRevision: 1, ProjectID: project.ID}
	if err := newStoreRunRepository(st).SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	stream, err := st.CreateValidationEvidenceStream(project.ID, missionEvidenceStreamName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AppendValidationEvidence(store.ValidationEvidenceAppendRequest{
		StreamID: stream.ID,
		Entry: store.ValidationEvidenceEntry{
			ID: "entry-without-run-id", Scenario: "mission/legacy", Outcome: store.EvidencePass,
			Confidence: store.EvidenceObserved, EvidenceJSON: `{"plan_id":"plan-metadata"}`,
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewRunApplicationService(n).ValidationEvidence(context.Background(), run.ID); err == nil {
		t.Fatal("evidence without a persisted run_id must fail closed")
	}
}

func TestRunApplicationProjectsCompleteMissionEvidenceBeyondDefaultPage(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "evidence-complete", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateWorkPlan(store.WorkPlan{ID: "plan-complete", ProjectID: project.ID, Title: "Complete"}); err != nil {
		t.Fatal(err)
	}
	run := &runner.MissionRun{ID: "run-complete", PlanID: "plan-complete", PlanRevision: 1, ProjectID: project.ID}
	if err := newStoreRunRepository(st).SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	stream, err := st.CreateValidationEvidenceStream(project.ID, missionEvidenceStreamName)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 101; i++ {
		if _, err := st.AppendValidationEvidence(store.ValidationEvidenceAppendRequest{
			StreamID: stream.ID,
			Entry: store.ValidationEvidenceEntry{
				ID: fmt.Sprintf("entry-complete-%03d", i), Scenario: fmt.Sprintf("mission/step/%03d", i),
				Outcome: store.EvidencePass, Confidence: store.EvidenceObserved,
				EvidenceJSON: `{"run_id":"run-complete"}`,
			},
		}); err != nil {
			t.Fatalf("append evidence %d: %v", i, err)
		}
	}
	report, err := NewRunApplicationService(n).ValidationEvidence(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !report.ChainVerified || len(report.Entries) != 101 {
		t.Fatalf("complete evidence report was truncated: chain=%v entries=%d", report.ChainVerified, len(report.Entries))
	}
}

func TestRunApplicationProjectsValidationEvidenceAfterStoreRestart(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	project, err := st.CreateProject(store.Project{Name: "evidence-restart", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateWorkPlan(store.WorkPlan{ID: "evidence-restart-plan", ProjectID: project.ID, Title: "Evidence restart"}); err != nil {
		t.Fatal(err)
	}
	run, err := n.Runner().StartMissionRun(context.Background(), runner.PlanSpec{
		ID: "evidence-restart-plan", ProjectID: project.ID, Revision: 1,
		Packages: []runner.PackageSpec{{ID: "package-1", Title: "Evidence"}},
	}, root, runner.DefaultAutonomyContract(), "")
	if err != nil {
		t.Fatal(err)
	}
	result := runner.VerificationResult{Command: "true", Passed: true, ExitCode: 0}
	if err := n.recordMissionValidationEvidence(context.Background(), run, "mission/restart", result.Command, []runner.VerificationResult{result}, "local", "test", "economy", 1); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(os.Getenv("AI_CLI_DATA_DIR"), "nexus.db")
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	restarted := &Nexus{st: reopened}
	report, err := NewRunApplicationService(restarted).ValidationEvidence(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if report.RunID != run.ID || report.Stream == nil || !report.ChainVerified || len(report.Entries) != 1 {
		t.Fatalf("restart evidence report = %+v", report)
	}
	if report.Entries[0].Scenario != "mission/restart" {
		t.Fatalf("unexpected restart evidence entry: %+v", report.Entries[0])
	}
}
