package runner

import (
	"testing"
	"time"
)

func TestClassifyAttention_BlockedNeedsUser(t *testing.T) {
	run := &MissionRun{State: StateBlockedNeedsUser}
	if got := ClassifyAttention(run); got != AttentionRequireUser {
		t.Fatalf("expected REQUIRE_USER, got %s", got)
	}
}

func TestClassifyAttention_FailedNoProgressWithEscalation(t *testing.T) {
	run := &MissionRun{State: StateFailedNoProgress, Contract: AutonomyContract{EscalateOnFailure: true}}
	if got := ClassifyAttention(run); got != AttentionNotify {
		t.Fatalf("expected NOTIFY, got %s", got)
	}
}

func TestClassifyAttention_FailedNoProgressWithoutEscalation(t *testing.T) {
	run := &MissionRun{State: StateFailedNoProgress, Contract: AutonomyContract{EscalateOnFailure: false}}
	if got := ClassifyAttention(run); got != AttentionInApp {
		t.Fatalf("expected IN_APP, got %s", got)
	}
}

func TestClassifyAttention_CompletedVerified(t *testing.T) {
	run := &MissionRun{State: StateCompletedVerified}
	if got := ClassifyAttention(run); got != AttentionInApp {
		t.Fatalf("expected IN_APP, got %s", got)
	}
}

func TestClassifyAttention_Executing(t *testing.T) {
	run := &MissionRun{State: StateExecuting}
	if got := ClassifyAttention(run); got != AttentionIgnore {
		t.Fatalf("expected IGNORE, got %s", got)
	}
}

func TestClassifyAttention_Paused(t *testing.T) {
	run := &MissionRun{State: StatePaused}
	if got := ClassifyAttention(run); got != AttentionInApp {
		t.Fatalf("expected IN_APP, got %s", got)
	}
}

func TestClassifyAttention_NilRun(t *testing.T) {
	if got := ClassifyAttention(nil); got != AttentionIgnore {
		t.Fatalf("expected IGNORE, got %s", got)
	}
}

func TestAttentionFromRun_NeedsHumanFields(t *testing.T) {
	now := time.Now().UTC()
	run := &MissionRun{
		ID:    "run_123",
		State: StateBlockedNeedsUser,
		NeedsHuman: &HumanIntervention{
			ID:                 "intv_1",
			ReasonCode:         "DISPATCH_OUTCOME_UNKNOWN",
			Summary:            "Provider outcome unknown",
			Question:           "Inspect provider runtime?",
			RecommendedActions: []string{"Check logs", "Retry"},
			Impact:             "Only affected task paused",
			MissionID:          "run_123",
			TaskID:             "pkg_1",
			Source:             "mission_runner",
			Timestamp:          now,
			Version:            1,
		},
		UpdatedAt: now,
	}
	item := AttentionFromRun(run)
	if item == nil {
		t.Fatal("expected non-nil attention item")
	}
	if item.Level != AttentionRequireUser {
		t.Fatalf("expected REQUIRE_USER, got %s", item.Level)
	}
	if item.ReasonCode != "DISPATCH_OUTCOME_UNKNOWN" {
		t.Fatalf("expected DISPATCH_OUTCOME_UNKNOWN, got %s", item.ReasonCode)
	}
	if len(item.Recommended) != 2 {
		t.Fatalf("expected 2 recommended actions, got %d", len(item.Recommended))
	}
	if item.Intervention == nil {
		t.Fatal("expected intervention to be set")
	}
}

func TestAttentionFromRun_CompletedReturnsItem(t *testing.T) {
	now := time.Now().UTC()
	run := &MissionRun{
		ID:          "run_456",
		State:       StateCompletedVerified,
		CompletedAt: &now,
		UpdatedAt:   now,
	}
	item := AttentionFromRun(run)
	if item == nil {
		t.Fatal("expected non-nil attention item for completed mission")
	}
	if item.Level != AttentionInApp {
		t.Fatalf("expected IN_APP, got %s", item.Level)
	}
}

func TestAttentionFromRun_ExecutingReturnsNil(t *testing.T) {
	run := &MissionRun{State: StateExecuting}
	item := AttentionFromRun(run)
	if item != nil {
		t.Fatal("expected nil for executing mission")
	}
}

func TestAttentionDedupKey_StableForSameIntervention(t *testing.T) {
	run := &MissionRun{
		ID:    "run_1",
		State: StateBlockedNeedsUser,
		NeedsHuman: &HumanIntervention{
			ID: "intv_1", Version: 1, MissionID: "run_1",
		},
	}
	key1 := AttentionDedupKey(run)
	key2 := AttentionDedupKey(run)
	if key1 != key2 {
		t.Fatalf("dedup key not stable: %s != %s", key1, key2)
	}
}

func TestAttentionDedupKey_DifferentForDifferentVersion(t *testing.T) {
	run1 := &MissionRun{
		ID: "run_1", State: StateBlockedNeedsUser,
		NeedsHuman: &HumanIntervention{ID: "intv_1", Version: 1},
	}
	run2 := &MissionRun{
		ID: "run_1", State: StateBlockedNeedsUser,
		NeedsHuman: &HumanIntervention{ID: "intv_1", Version: 2},
	}
	if AttentionDedupKey(run1) == AttentionDedupKey(run2) {
		t.Fatal("dedup keys should differ for different versions")
	}
}

func TestAttentionDedupKey_EmptyForNil(t *testing.T) {
	if AttentionDedupKey(nil) != "" {
		t.Fatal("expected empty key for nil run")
	}
}
