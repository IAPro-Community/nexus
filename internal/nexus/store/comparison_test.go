package store

import "testing"

func TestComparisonLabPersistsHumanDecisionAndRejectsReplacement(t *testing.T) {
	st := openTestStore(t)
	project, err := st.CreateProject(Project{Name: "comparison", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(WorkPlan{ProjectID: project.ID, Title: "same frozen plan"})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"run-a", "run-b"} {
		if err := st.UpsertMissionRun(MissionRunRecord{ID: id, PlanID: plan.ID, ProjectID: project.ID, State: "COMPLETED_VERIFIED", PayloadJSON: `{}`}); err != nil {
			t.Fatal(err)
		}
	}
	created, err := st.CreateComparisonLab(ComparisonLabRecord{
		ProjectID: project.ID, PlanID: plan.ID, PlanRevision: 1, RunAID: "run-a", RunBID: "run-b", EvidenceJSON: `{"run_a":{"run_id":"run-a"},"run_b":{"run_id":"run-b"}}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Decision != ComparisonPending {
		t.Fatalf("initial decision=%q, want %q", created.Decision, ComparisonPending)
	}
	selected, err := st.DecideComparisonLab(created.ID, ComparisonSelectB, "B passed the required checks with the smaller diff")
	if err != nil {
		t.Fatal(err)
	}
	if selected.Decision != ComparisonSelectB || selected.DecidedAt == nil || selected.Reason == "" {
		t.Fatalf("human decision was not persisted: %+v", selected)
	}
	if _, err := st.DecideComparisonLab(created.ID, ComparisonSelectB, selected.Reason); err != nil {
		t.Fatalf("same decision should be idempotent: %v", err)
	}
	if _, err := st.DecideComparisonLab(created.ID, ComparisonSelectA, "replace"); err == nil {
		t.Fatal("a different decision must not replace the recorded human choice")
	}
}

func TestComparisonLabRequiresTwoDistinctRunsAndReason(t *testing.T) {
	st := openTestStore(t)
	if _, err := st.CreateComparisonLab(ComparisonLabRecord{
		ProjectID: "project", PlanID: "plan", PlanRevision: 1, RunAID: "same", RunBID: "same",
	}); err == nil {
		t.Fatal("same run cannot be compared against itself")
	}
	if _, err := st.DecideComparisonLab("missing", ComparisonRejectAll, " "); err == nil {
		t.Fatal("human decision requires a rationale")
	}
}
