package nexus

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestGlobalAttentionProjectionSurvivesStoreReopen(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "attention.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	projectA, err := st.CreateProject(store.Project{Name: "Attention A", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	projectB, err := st.CreateProject(store.Project{Name: "Attention B", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	planA, err := st.CreateWorkPlan(store.WorkPlan{ProjectID: projectA.ID, Title: "Attention A plan"})
	if err != nil {
		t.Fatal(err)
	}
	planB, err := st.CreateWorkPlan(store.WorkPlan{ProjectID: projectB.ID, Title: "Attention B plan"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	completedAt := now.Add(-time.Minute)

	runs := []*runner.MissionRun{
		{
			ID: "mission-a-needs-user", ProjectID: projectA.ID, PlanID: planA.ID, State: runner.StateBlockedNeedsUser,
			UpdatedAt: now,
			NeedsHuman: &runner.HumanIntervention{
				ID: "intervention-a", MissionID: "mission-a-needs-user", Timestamp: now, Version: 1,
				Question: "Review the pending decision.",
			},
		},
		{ID: "mission-b-executing", ProjectID: projectA.ID, PlanID: planA.ID, State: runner.StateExecuting, UpdatedAt: now},
		{
			ID: "mission-c-after-fallback", ProjectID: projectB.ID, PlanID: planB.ID, State: runner.StateExecuting, UpdatedAt: now,
			PackageRuns: []runner.PackageRun{{
				ID: "package-c", State: runner.StateExecuting, Provider: "codex", DesiredProvider: "claude",
			}},
		},
		{
			ID: "mission-d-completed", ProjectID: projectB.ID, PlanID: planB.ID, State: runner.StateCompletedVerified,
			CompletedAt: &completedAt, UpdatedAt: completedAt,
		},
		{
			ID: "mission-e-failed", ProjectID: projectB.ID, PlanID: planB.ID, State: runner.StateFailedNoProgress,
			Contract: runner.AutonomyContract{EscalateOnFailure: true}, UpdatedAt: now,
		},
	}
	repo := newStoreRunRepository(st)
	for _, run := range runs {
		if err := repo.SaveRun(ctx, run); err != nil {
			t.Fatalf("save %s: %v", run.ID, err)
		}
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	st, err = store.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	center := runner.NewAttentionCenter(newStoreRunRepository(st))
	global, err := center.ListAttention(ctx)
	if err != nil {
		t.Fatalf("list global attention after reopen: %v", err)
	}
	if len(global.AllItems) != 3 || global.TotalNeeds != 1 || len(global.Completed) != 1 || len(global.Failed) != 1 {
		t.Fatalf("unexpected canonical projection after reopen: %+v", global)
	}
	gotIDs := map[string]bool{}
	for _, item := range global.AllItems {
		gotIDs[item.MissionID] = true
	}
	for _, wantID := range []string{"mission-a-needs-user", "mission-d-completed", "mission-e-failed"} {
		if !gotIDs[wantID] {
			t.Errorf("attention projection lost %s after reopen", wantID)
		}
	}
	for _, excludedID := range []string{"mission-b-executing", "mission-c-after-fallback"} {
		if gotIDs[excludedID] {
			t.Errorf("non-attention mission %s leaked into the projection", excludedID)
		}
	}

	projectGroup, err := center.ListAttentionFiltered(ctx, runner.AttentionFilter{ProjectID: projectB.ID})
	if err != nil {
		t.Fatalf("filter by project after reopen: %v", err)
	}
	if len(projectGroup.AllItems) != 2 || len(projectGroup.Completed) != 1 || len(projectGroup.Failed) != 1 {
		t.Fatalf("project filter mixed missions or lost categories: %+v", projectGroup)
	}

	missionGroup, err := center.ListAttentionFiltered(ctx, runner.AttentionFilter{
		ProjectID: projectA.ID,
		MissionID: "mission-a-needs-user",
	})
	if err != nil {
		t.Fatalf("filter by mission after reopen: %v", err)
	}
	if len(missionGroup.AllItems) != 1 || missionGroup.AllItems[0].MissionID != "mission-a-needs-user" {
		t.Fatalf("mission filter returned unexpected items: %+v", missionGroup.AllItems)
	}

	conflictingGroup, err := center.ListAttentionFiltered(ctx, runner.AttentionFilter{
		ProjectID: projectA.ID,
		MissionID: "mission-d-completed",
	})
	if err != nil {
		t.Fatalf("filter conflicting project and mission after reopen: %v", err)
	}
	if len(conflictingGroup.AllItems) != 0 {
		t.Fatalf("combined filters should intersect without cross-project leakage: %+v", conflictingGroup.AllItems)
	}
}
