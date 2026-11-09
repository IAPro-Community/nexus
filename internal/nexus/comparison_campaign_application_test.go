package nexus

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestComparisonCampaignCreatesTwoFrozenVariantsFromOneCleanBase(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	st, err := store.Open(filepath.Join(dataDir, "nexus.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	repo := filepath.Join(dataDir, "project")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	comparisonGit(t, repo, "init", "-q")
	comparisonGit(t, repo, "config", "user.email", "comparison@example.invalid")
	comparisonGit(t, repo, "config", "user.name", "Comparison Fixture")
	if err := os.WriteFile(filepath.Join(repo, "readme.txt"), []byte("base\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	comparisonGit(t, repo, "add", "readme.txt")
	comparisonGit(t, repo, "commit", "-qm", "baseline")
	project, err := st.CreateProject(store.Project{Name: "comparison", CanonicalPath: repo})
	if err != nil {
		t.Fatal(err)
	}
	source, err := st.CreateWorkPlan(store.WorkPlan{
		ProjectID: project.ID, Title: "Improve the parser", Description: "One shared objective",
		Phases: []store.PlanPhase{{ID: "phase-source", Title: "Implement", Packages: []store.WorkPackage{
			{ID: "pkg-core", Title: "Core", Goal: "Implement the shared behavior", Role: "implementer", AcceptanceCriteria: []string{"preserve behavior"}},
			{ID: "pkg-tests", Title: "Tests", Goal: "Verify behavior", Dependencies: []string{"pkg-core"}, Role: "tester"},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	n := NewWithStoreAndExecutor(st, nil)
	runs := NewRunApplicationService(n)
	service := NewComparisonCampaignApplicationService(n, runs)
	view, err := service.Create(context.Background(), CreateComparisonCampaignRequest{
		ProjectID: project.ID, PlanID: source.ID, PlanRevision: source.CurrentRevision,
		VariantA: ComparisonCampaignVariantInput{Name: "minimal patch", Instructions: "Prefer the smallest local change."},
		VariantB: ComparisonCampaignVariantInput{Name: "redesign", Instructions: "Refactor the parser around explicit stages."},
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Campaign.State != store.ComparisonCampaignPrepared || view.Campaign.BaseRevision != comparisonGit(t, repo, "rev-parse", "HEAD") {
		t.Fatalf("campaign did not preserve the source base: %+v", view.Campaign)
	}
	if view.PlanA.ID == view.PlanB.ID || view.PlanA.ID == source.ID || view.PlanB.ID == source.ID {
		t.Fatal("alternatives must be independent frozen plans")
	}
	if len(view.PlanA.Phases) != 1 || len(view.PlanB.Phases) != 1 {
		t.Fatal("variant phases missing")
	}
	packagesA := view.PlanA.Phases[0].Packages
	packagesB := view.PlanB.Phases[0].Packages
	if packagesA[0].ID == packagesB[0].ID || packagesA[0].ID == "pkg-core" || packagesB[0].ID == "pkg-core" {
		t.Fatal("variant package identities must be isolated")
	}
	if packagesA[1].Dependencies[0] != packagesA[0].ID || packagesB[1].Dependencies[0] != packagesB[0].ID {
		t.Fatal("dependency graph must be remapped within each alternative")
	}
	if !strings.Contains(packagesA[0].Goal, "smallest local change") || !strings.Contains(packagesB[0].Goal, "explicit stages") {
		t.Fatal("alternative instructions weren't frozen into their execution goals")
	}
	if view.PreviewA == nil || view.PreviewB == nil {
		t.Fatal("both variants need their own Execution Preview")
	}
	if view.PreviewA.PlanID == view.PreviewB.PlanID || view.PreviewA.Revision != 1 || view.PreviewB.Revision != 1 {
		t.Fatal("variant approvals must be bound to distinct frozen revisions")
	}
	loadedSource, err := st.GetWorkPlan(source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loadedSource.Phases[0].Packages[0].Goal != "Implement the shared behavior" {
		t.Fatal("preparing alternatives mutated the original plan")
	}
	if _, err := service.Launch(context.Background(), view.Campaign.ID, LaunchComparisonCampaignRequest{}); err == nil {
		t.Fatal("unapproved and/or blocked previews must not launch alternatives")
	}
	stillPrepared, err := st.GetComparisonCampaign(view.Campaign.ID)
	if err != nil || stillPrepared.State != store.ComparisonCampaignPrepared {
		t.Fatalf("preview rejection must remain safely retryable: %+v %v", stillPrepared, err)
	}
	if _, err := st.BeginComparisonCampaignLaunch(view.Campaign.ID); err != nil {
		t.Fatal(err)
	}
	restarted := NewComparisonCampaignApplicationService(n, runs)
	recovered, err := restarted.Get(context.Background(), view.Campaign.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Campaign.State != store.ComparisonCampaignPrepared {
		t.Fatalf("restart with no acquired Mission runs should recover safely to PREPARED: %s", recovered.Campaign.State)
	}
}

func TestComparisonCampaignRejectsDirtyCheckoutAndInvalidVariant(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	st, err := store.Open(filepath.Join(dataDir, "nexus.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	repo := filepath.Join(dataDir, "project")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	comparisonGit(t, repo, "init", "-q")
	comparisonGit(t, repo, "config", "user.email", "comparison@example.invalid")
	comparisonGit(t, repo, "config", "user.name", "Comparison Fixture")
	if err := os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	comparisonGit(t, repo, "add", "base.txt")
	comparisonGit(t, repo, "commit", "-qm", "baseline")
	project, err := st.CreateProject(store.Project{Name: "comparison", CanonicalPath: repo})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(store.WorkPlan{ProjectID: project.ID, Title: "Plan", Phases: []store.PlanPhase{{Packages: []store.WorkPackage{{ID: "step", Goal: "Do the work"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	n := NewWithStoreAndExecutor(st, nil)
	service := NewComparisonCampaignApplicationService(n, NewRunApplicationService(n))
	request := CreateComparisonCampaignRequest{ProjectID: project.ID, PlanID: plan.ID, PlanRevision: 1,
		VariantA: ComparisonCampaignVariantInput{Name: "A", Instructions: "A"}, VariantB: ComparisonCampaignVariantInput{Name: "B", Instructions: "B"}}
	if _, err := service.Create(context.Background(), CreateComparisonCampaignRequest{ProjectID: project.ID, PlanID: plan.ID, PlanRevision: 1, VariantA: request.VariantA}); err == nil {
		t.Fatal("missing alternative B must be rejected")
	}
	if err := os.WriteFile(filepath.Join(repo, "uncommitted.txt"), []byte("dirty"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(context.Background(), request); err == nil {
		t.Fatal("dirty base must be rejected before creating variants")
	}
	plans, err := st.ListWorkPlans(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 {
		t.Fatalf("rejected preparation left variant plans behind: %d", len(plans))
	}
}

func comparisonGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out))
}
