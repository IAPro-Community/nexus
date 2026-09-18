package runner

import (
	"context"
	"testing"
	"time"
)

func TestAttentionCenter_GroupsCorrectly(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRunRepository()
	center := NewAttentionCenter(repo)
	now := time.Now().UTC()

	runA := &MissionRun{
		ID: "run_a", ProjectID: "proj_1", State: StateBlockedNeedsUser,
		NeedsHuman: &HumanIntervention{
			ID: "intv_a", ReasonCode: "DISPATCH_OUTCOME_UNKNOWN", Summary: "Provider unknown",
			Question: "Inspect?", MissionID: "run_a", TaskID: "pkg_a",
			Source: "mission_runner", Timestamp: now, Version: 1,
		},
		UpdatedAt: now,
	}
	if err := repo.SaveRun(ctx, runA); err != nil {
		t.Fatal(err)
	}

	completedAt := now.Add(-time.Minute)
	runB := &MissionRun{
		ID: "run_b", ProjectID: "proj_2", State: StateCompletedVerified,
		CompletedAt: &completedAt, UpdatedAt: completedAt,
	}
	if err := repo.SaveRun(ctx, runB); err != nil {
		t.Fatal(err)
	}

	runC := &MissionRun{
		ID: "run_c", ProjectID: "proj_3", State: StateFailedNoProgress,
		Contract: AutonomyContract{EscalateOnFailure: true}, UpdatedAt: now,
	}
	if err := repo.SaveRun(ctx, runC); err != nil {
		t.Fatal(err)
	}

	runD := &MissionRun{
		ID: "run_d", ProjectID: "proj_1", State: StateExecuting, UpdatedAt: now,
	}
	if err := repo.SaveRun(ctx, runD); err != nil {
		t.Fatal(err)
	}

	group, err := center.ListAttention(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(group.NeedsYou) != 1 {
		t.Fatalf("expected 1 NeedsYou, got %d", len(group.NeedsYou))
	}
	if group.NeedsYou[0].MissionID != "run_a" {
		t.Fatalf("expected run_a in NeedsYou, got %s", group.NeedsYou[0].MissionID)
	}

	if len(group.Completed) != 1 {
		t.Fatalf("expected 1 Completed, got %d", len(group.Completed))
	}
	if group.Completed[0].MissionID != "run_b" {
		t.Fatalf("expected run_b in Completed, got %s", group.Completed[0].MissionID)
	}

	if len(group.Failed) != 1 {
		t.Fatalf("expected 1 Failed, got %d", len(group.Failed))
	}
	if group.Failed[0].MissionID != "run_c" {
		t.Fatalf("expected run_c in Failed, got %s", group.Failed[0].MissionID)
	}

	if len(group.AllItems) != 3 {
		t.Fatalf("expected 3 AllItems (excluding executing), got %d", len(group.AllItems))
	}

	if group.TotalNeeds != 1 {
		t.Fatalf("expected TotalNeeds=1, got %d", group.TotalNeeds)
	}
}

func TestAttentionCenter_NeedsUserCount(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRunRepository()
	center := NewAttentionCenter(repo)
	now := time.Now().UTC()

	repo.SaveRun(ctx, &MissionRun{
		ID: "r1", State: StateBlockedNeedsUser, UpdatedAt: now,
		NeedsHuman: &HumanIntervention{ID: "i1", MissionID: "r1", Timestamp: now, Version: 1},
	})
	repo.SaveRun(ctx, &MissionRun{
		ID: "r2", State: StateBlockedNeedsUser, UpdatedAt: now,
		NeedsHuman: &HumanIntervention{ID: "i2", MissionID: "r2", Timestamp: now, Version: 1},
	})
	repo.SaveRun(ctx, &MissionRun{ID: "r3", State: StateExecuting, UpdatedAt: now})

	count, err := center.NeedsUserCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestAttentionCenter_GetAttention(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRunRepository()
	center := NewAttentionCenter(repo)
	now := time.Now().UTC()

	repo.SaveRun(ctx, &MissionRun{
		ID: "r1", ProjectID: "p1", State: StateBlockedNeedsUser, UpdatedAt: now,
		NeedsHuman: &HumanIntervention{
			ID: "i1", ReasonCode: "SECURITY", Summary: "Needs approval",
			Question: "Approve?", MissionID: "r1", TaskID: "t1",
			Source: "runner", Timestamp: now, Version: 1,
		},
	})

	item, err := center.GetAttention(ctx, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if item == nil {
		t.Fatal("expected non-nil item")
	}
	if item.ReasonCode != "SECURITY" {
		t.Fatalf("expected SECURITY, got %s", item.ReasonCode)
	}
}
