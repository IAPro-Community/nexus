package runner

import (
	"context"
	"testing"
	"time"
)

// TestMultiMissionIndependentProgression proves that multiple missions can
// progress independently: one blocked, one executing, one completed, one failed.
func TestMultiMissionIndependentProgression(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRunRepository()
	center := NewAttentionCenter(repo)
	now := time.Now().UTC()

	// Mission A: BLOCKED_NEEDS_USER (needs human decision)
	runA := &MissionRun{
		ID: "run_a", ProjectID: "proj_1", State: StateBlockedNeedsUser,
		NeedsHuman: &HumanIntervention{
			ID: "intv_a", ReasonCode: "DISPATCH_OUTCOME_UNKNOWN",
			Summary: "Provider outcome unknown", Question: "Inspect provider?",
			MissionID: "run_a", TaskID: "pkg_a", Source: "runner",
			Timestamp: now, Version: 1,
		},
		UpdatedAt: now,
	}
	repo.SaveRun(ctx, runA)

	// Mission B: EXECUTING (should NOT appear in attention)
	runB := &MissionRun{
		ID: "run_b", ProjectID: "proj_2", State: StateExecuting,
		UpdatedAt: now,
	}
	repo.SaveRun(ctx, runB)

	// Mission C: COMPLETED_VERIFIED
	completedAt := now.Add(-2 * time.Minute)
	runC := &MissionRun{
		ID: "run_c", ProjectID: "proj_3", State: StateCompletedVerified,
		CompletedAt: &completedAt, UpdatedAt: completedAt,
	}
	repo.SaveRun(ctx, runC)

	// Mission D: FAILED_NO_PROGRESS
	runD := &MissionRun{
		ID: "run_d", ProjectID: "proj_4", State: StateFailedNoProgress,
		Contract: AutonomyContract{EscalateOnFailure: true}, UpdatedAt: now,
	}
	repo.SaveRun(ctx, runD)

	group, err := center.ListAttention(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Verify: only 3 items in attention (B is IGNORE)
	if len(group.AllItems) != 3 {
		t.Fatalf("expected 3 attention items, got %d", len(group.AllItems))
	}

	// Verify: NeedsYou = A only
	if len(group.NeedsYou) != 1 {
		t.Fatalf("expected 1 NeedsYou, got %d", len(group.NeedsYou))
	}
	if group.NeedsYou[0].MissionID != "run_a" {
		t.Fatalf("expected run_a in NeedsYou, got %s", group.NeedsYou[0].MissionID)
	}
	if group.NeedsYou[0].Level != AttentionRequireUser {
		t.Fatalf("expected REQUIRE_USER for run_a, got %s", group.NeedsYou[0].Level)
	}

	// Verify: Completed = C only
	if len(group.Completed) != 1 {
		t.Fatalf("expected 1 Completed, got %d", len(group.Completed))
	}
	if group.Completed[0].MissionID != "run_c" {
		t.Fatalf("expected run_c in Completed, got %s", group.Completed[0].MissionID)
	}

	// Verify: Failed = D only
	if len(group.Failed) != 1 {
		t.Fatalf("expected 1 Failed, got %d", len(group.Failed))
	}
	if group.Failed[0].MissionID != "run_d" {
		t.Fatalf("expected run_d in Failed, got %s", group.Failed[0].MissionID)
	}

	// Verify: TotalNeeds = 1
	if group.TotalNeeds != 1 {
		t.Fatalf("expected TotalNeeds=1, got %d", group.TotalNeeds)
	}

	// Verify: B (EXECUTING) does not appear anywhere
	for _, item := range group.AllItems {
		if item.MissionID == "run_b" {
			t.Fatal("executing mission should not appear in attention")
		}
	}
}

// TestResolveInterventionIdempotent proves that resolving the same intervention
// twice produces exactly one semantic resolution.
func TestResolveInterventionIdempotent(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()
	now := time.Now().UTC()

	run := &MissionRun{
		ID: "run_idempotent", State: StateBlockedNeedsUser,
		PackageRuns: []PackageRun{{ID: "pkg-1", PackageID: "pkg-1", State: StateFailed, DispatchState: DispatchFailedBeforeDispatch}},
		NeedsHuman: &HumanIntervention{
			ID: "intv_idempotent", ReasonCode: "SECURITY",
			Summary: "Security approval", Question: "Approve?",
			MissionID: "run_idempotent", TaskID: "pkg-1",
			Source: "runner", Timestamp: now, Version: 1,
			Options: []InterventionOption{{ID: "replan-package", Operation: InterventionReplanPackage, PackageID: "pkg-1"}},
		},
		UpdatedAt: now,
	}
	repo.SaveRun(ctx, run)

	// First resolution
	resolved1, err := r.ResolveIntervention(ctx, "run_idempotent", "intv_idempotent", 1, "replan-package", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if resolved1.State != StateExecuting {
		t.Fatalf("first: expected EXECUTING, got %s", resolved1.State)
	}
	if !resolved1.NeedsHuman.Resolved {
		t.Fatal("first: expected resolved=true")
	}

	// Save the resolved run
	repo.SaveRun(ctx, resolved1)

	// Second resolution (same intervention and option) must be idempotent.
	resolved2, err := r.ResolveIntervention(ctx, "run_idempotent", "intv_idempotent", 1, "replan-package", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if resolved2.NeedsHuman.Resolution.IdempotencyKey != resolved1.NeedsHuman.Resolution.IdempotencyKey {
		t.Fatal("duplicate decision returned a different semantic resolution")
	}
}

// TestStaleInterventionRejected proves that an old intervention version is rejected.
func TestStaleInterventionRejected(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()
	now := time.Now().UTC()

	// Current intervention is version 2
	run := &MissionRun{
		ID: "run_stale", State: StateBlockedNeedsUser,
		NeedsHuman: &HumanIntervention{
			ID: "intv_current", ReasonCode: "SECURITY",
			Summary: "Current intervention", Question: "Current?",
			MissionID: "run_stale", TaskID: "pkg_1",
			Source: "runner", Timestamp: now, Version: 2,
		},
		UpdatedAt: now,
	}
	repo.SaveRun(ctx, run)

	// Try to resolve with stale version 1
	_, err := r.ResolveIntervention(ctx, "run_stale", "intv_current", 1, "approve", "")
	if err == nil {
		t.Fatal("expected error for stale version")
	}

	// Try to resolve with wrong intervention ID
	_, err = r.ResolveIntervention(ctx, "run_stale", "intv_old", 2, "approve", "")
	if err == nil {
		t.Fatal("expected error for wrong intervention ID")
	}
}

// TestResumeFromDurableCheckpoint proves that after resolving an intervention,
// the mission resumes from its durable state without re-executing completed work.
func TestResumeFromDurableCheckpoint(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()
	now := time.Now().UTC()

	// Mission with one verified package and one blocked
	run := &MissionRun{
		ID: "run_resume", ProjectID: "proj_1", State: StateBlockedNeedsUser,
		Contract: AutonomyContract{MaxRetries: 3, MaxTotalIterations: 120, MaxNoProgress: 2},
		PackageRuns: []PackageRun{
			{ID: "pkg_1", PackageID: "pkg_1", State: StateVerified, AssignedAgent: "agent_1"},
			{ID: "pkg_2", PackageID: "pkg_2", State: StateExecuting, AssignedAgent: "agent_2", DispatchState: DispatchUnknownExternalOutcome, DispatchID: "dispatch-2"},
		},
		NeedsHuman: &HumanIntervention{
			ID: "intv_resume", ReasonCode: "DISPATCH_OUTCOME_UNKNOWN",
			Summary: "Provider unknown", Question: "Inspect?",
			MissionID: "run_resume", TaskID: "pkg_2",
			Source: "runner", Timestamp: now, Version: 1,
			Options: []InterventionOption{{ID: "confirm-external-completion", Operation: InterventionConfirmExternalOutcome, PackageID: "pkg_2"}},
		},
		UpdatedAt: now,
	}
	repo.SaveRun(ctx, run)

	// Resolve the intervention
	resolved, err := r.ResolveIntervention(ctx, "run_resume", "intv_resume", 1, "confirm-external-completion", "user-1")
	if err != nil {
		t.Fatal(err)
	}

	// Verify: state is EXECUTING
	if resolved.State != StateExecuting {
		t.Fatalf("expected EXECUTING, got %s", resolved.State)
	}

	// Verify: pkg_1 is still VERIFIED (not re-executed)
	for _, pkg := range resolved.PackageRuns {
		if pkg.PackageID == "pkg_1" && pkg.State != StateVerified {
			t.Fatalf("pkg_1 should remain VERIFIED, got %s", pkg.State)
		}
	}

	// Verify: NeedsHuman is resolved
	if !resolved.NeedsHuman.Resolved {
		t.Fatal("NeedsHuman should be resolved")
	}
	if resolved.NeedsHuman.ResolutionDecision != "confirm-external-completion" {
		t.Fatalf("expected confirm option, got %s", resolved.NeedsHuman.ResolutionDecision)
	}
}

// TestQuotaFailoverNoAttention proves that quota/rate-limit failover does not
// generate attention items.
func TestQuotaFailoverNoAttention(t *testing.T) {
	repo := NewMemoryRunRepository()
	center := NewAttentionCenter(repo)
	ctx := context.Background()
	now := time.Now().UTC()

	// Mission that went through quota failover but is now executing on new provider
	run := &MissionRun{
		ID: "run_failover", State: StateExecuting, UpdatedAt: now,
		PackageRuns: []PackageRun{
			{ID: "pkg_1", State: StateExecuting, AssignedAgent: "agent_1"},
		},
	}
	repo.SaveRun(ctx, run)

	group, err := center.ListAttention(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Should not appear in any attention group
	if len(group.AllItems) != 0 {
		t.Fatalf("expected 0 attention items for failover, got %d", len(group.AllItems))
	}
}

// TestNotificationDedupKey proves that the dedup key is stable across cycles.
func TestNotificationDedupKey(t *testing.T) {
	run := &MissionRun{
		ID: "run_dedup", State: StateBlockedNeedsUser,
		NeedsHuman: &HumanIntervention{
			ID: "intv_dedup", Version: 3, MissionID: "run_dedup",
		},
	}
	key := AttentionDedupKey(run)
	expected := "BLOCKED_NEEDS_USER:intv_dedup:v3:run_dedup"
	if key != expected {
		t.Fatalf("expected %q, got %q", expected, key)
	}
}

// TestAutonomyContractRespected proves that the attention policy respects the
// AutonomyContract settings.
func TestAutonomyContractRespected(t *testing.T) {
	// With EscalateOnFailure=true, failed missions should be NOTIFY
	run1 := &MissionRun{
		State:    StateFailedNoProgress,
		Contract: AutonomyContract{EscalateOnFailure: true},
	}
	if got := ClassifyAttention(run1); got != AttentionNotify {
		t.Fatalf("expected NOTIFY with escalation, got %s", got)
	}

	// With EscalateOnFailure=false, failed missions should be IN_APP
	run2 := &MissionRun{
		State:    StateFailedNoProgress,
		Contract: AutonomyContract{EscalateOnFailure: false},
	}
	if got := ClassifyAttention(run2); got != AttentionInApp {
		t.Fatalf("expected IN_APP without escalation, got %s", got)
	}
}

// TestCrossProjectAggregation proves that attention items are aggregated across
// all projects.
func TestCrossProjectAggregation(t *testing.T) {
	repo := NewMemoryRunRepository()
	center := NewAttentionCenter(repo)
	ctx := context.Background()
	now := time.Now().UTC()

	repo.SaveRun(ctx, &MissionRun{
		ID: "r1", ProjectID: "proj_a", State: StateBlockedNeedsUser, UpdatedAt: now,
		NeedsHuman: &HumanIntervention{ID: "i1", MissionID: "r1", Timestamp: now, Version: 1},
	})
	repo.SaveRun(ctx, &MissionRun{
		ID: "r2", ProjectID: "proj_b", State: StateBlockedNeedsUser, UpdatedAt: now,
		NeedsHuman: &HumanIntervention{ID: "i2", MissionID: "r2", Timestamp: now, Version: 1},
	})
	repo.SaveRun(ctx, &MissionRun{
		ID: "r3", ProjectID: "proj_c", State: StateCompletedVerified,
		CompletedAt: &now, UpdatedAt: now,
	})

	group, err := center.ListAttention(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if group.TotalNeeds != 2 {
		t.Fatalf("expected 2 total needs across projects, got %d", group.TotalNeeds)
	}
	if len(group.NeedsYou) != 2 {
		t.Fatalf("expected 2 NeedsYou items, got %d", len(group.NeedsYou))
	}
	if len(group.Completed) != 1 {
		t.Fatalf("expected 1 Completed item, got %d", len(group.Completed))
	}
}
