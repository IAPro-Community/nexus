package nexus

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestComparisonApplicationComparesIsolatedRunsAndPersistsHumanOnlyDecision(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	st, err := store.Open(filepath.Join(dataDir, "nexus.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	canonical := t.TempDir()
	project, err := st.CreateProject(store.Project{Name: "comparison", CanonicalPath: canonical})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(store.WorkPlan{ProjectID: project.ID, Title: "same plan"})
	if err != nil {
		t.Fatal(err)
	}
	baseA := createComparisonFixtureRepo(t, filepath.Join(dataDir, "integrations", project.ID, "run-a"), "alternative A")
	baseB := createComparisonFixtureRepo(t, filepath.Join(dataDir, "integrations", project.ID, "run-b"), "alternative B")
	if baseA != baseB {
		t.Fatalf("fixture bases differ: %s != %s", baseA, baseB)
	}
	now := time.Now().UTC()
	for _, fixture := range []struct {
		id, label string
	}{{"run-a", "alternative-a"}, {"run-b", "alternative-b"}} {
		workspace := filepath.Join(dataDir, "integrations", project.ID, fixture.id)
		run := runner.MissionRun{
			ID: fixture.id, PlanID: plan.ID, PlanRevision: 1, ProjectID: project.ID,
			State: runner.StateCompletedVerified, Workspace: canonical, IntegratedWorkspace: workspace,
			DeliveryIntegration: &runner.DeliveryIntegrationEvidence{
				Workspace: workspace, BaseRevision: baseA, ChangedFiles: []string{"alternative.txt"}, IntegratedAt: now,
			},
			PackageRuns: []runner.PackageRun{{
				Title:         fixture.label,
				Verifications: []runner.VerificationResult{{Command: "go test ./...", Passed: fixture.id == "run-a", ExitCode: 0, DurationMs: 120}},
			}},
			GlobalVerifications: []runner.VerificationResult{{Command: "make verify", Passed: true, DurationMs: 300}},
			StartedAt:           now, UpdatedAt: now,
		}
		payload, marshalErr := json.Marshal(run)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if err := st.UpsertMissionRun(store.MissionRunRecord{
			ID: run.ID, PlanID: run.PlanID, ProjectID: run.ProjectID, State: string(run.State), PayloadJSON: string(payload),
		}); err != nil {
			t.Fatal(err)
		}
	}
	knownCost := int64(1200)
	if _, err := st.RecordCostObservation(store.CostObservation{
		ProjectID: project.ID, MissionRunID: "run-a", CostMicroUSD: &knownCost,
		CostKind: store.CostKindProviderReported, Source: "test-fixture", ObservedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RecordCostObservation(store.CostObservation{
		ProjectID: project.ID, MissionRunID: "run-b", CostKind: store.CostKindUnknown,
		Source: "test-fixture", ObservedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	service := NewComparisonApplicationService(NewWithStoreAndExecutor(st, nil))
	view, err := service.Create(context.Background(), project.ID, "run-a", "run-b")
	if err != nil {
		t.Fatal(err)
	}
	if view.RunA.BaseRevision != view.RunB.BaseRevision || view.RunA.Diff == "" || view.RunB.Diff == "" {
		t.Fatalf("comparison did not expose common-base diffs: A=%+v B=%+v", view.RunA, view.RunB)
	}
	if !strings.Contains(view.RunA.Diff, "alternative A") || !strings.Contains(view.RunB.Diff, "alternative B") {
		t.Fatalf("diffs do not show each isolated alternative: A=%q B=%q", view.RunA.Diff, view.RunB.Diff)
	}
	savedDiff := view.RunA.Diff
	if err := os.WriteFile(filepath.Join(dataDir, "integrations", project.ID, "run-a", "alternative.txt"), []byte("changed after comparison\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reloaded, err := service.Get(context.Background(), view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.RunA.Diff != savedDiff {
		t.Fatal("saved comparison evidence changed after the run workspace changed")
	}
	if len(view.RunA.Checks) != 2 || !view.RunA.Checks[0].Passed || view.RunB.Checks[0].Passed {
		t.Fatalf("test evidence was not compared: A=%+v B=%+v", view.RunA.Checks, view.RunB.Checks)
	}
	if view.RunA.CostEvidenceStatus != "COMPLETE" || view.RunA.Cost.ProviderReportedMicroUSD != knownCost {
		t.Fatalf("known cost evidence missing: %+v", view.RunA)
	}
	if view.RunB.CostEvidenceStatus != "PARTIAL" || view.RunB.Cost.UnknownCostObservations != 1 {
		t.Fatalf("unknown cost must remain explicit: %+v", view.RunB)
	}
	decided, err := service.Decide(context.Background(), view.ID, store.ComparisonSelectA, "A passed the focused checks")
	if err != nil {
		t.Fatal(err)
	}
	if decided.Decision != store.ComparisonSelectA || decided.DecidedAt == nil {
		t.Fatalf("decision not persisted: %+v", decided.ComparisonLabRecord)
	}
	if _, err := os.Stat(filepath.Join(canonical, "alternative-a.txt")); !os.IsNotExist(err) {
		t.Fatalf("comparison decision unexpectedly changed canonical project: %v", err)
	}
}

func TestComparisonApplicationFailsClosedWhenUnavailable(t *testing.T) {
	service := NewComparisonApplicationService(nil)
	if _, err := service.Create(context.Background(), "project", "a", "b"); err == nil {
		t.Fatal("unavailable service must fail closed")
	}
}

func createComparisonFixtureRepo(t *testing.T, root, alternative string) string {
	t.Helper()
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("config", "user.email", "comparison@example.invalid")
	git("config", "user.name", "Comparison Fixture")
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("shared baseline\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git("add", "base.txt")
	git("commit", "-qm", "baseline")
	base := git("rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(root, "alternative.txt"), []byte(alternative+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return base
}
