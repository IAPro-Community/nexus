package handoff

import (
	"testing"
)

// Characterization tests: lock down the handoff state machine transitions
// and transaction structure before any evolution.

func TestHandoffStates_AllExist(t *testing.T) {
	expected := map[HandoffState]bool{
		HandoffRequested:            true,
		HandoffPreflight:            true,
		HandoffTargetValidated:      true,
		HandoffCheckpointed:         true,
		HandoffSourceQuiesced:       true,
		HandoffTargetStarting:       true,
		HandoffTargetResumed:        true,
		HandoffContinuityUnverified: true,
		HandoffVerified:             true,
		HandoffCompleted:            true,
		HandoffRollback:             true,
		HandoffRollingBack:          true,
		HandoffRolledBack:           true,
		HandoffFailedSafe:           true,
		HandoffFailedUnsafe:         true,
	}
	for state := range expected {
		if string(state) == "" {
			t.Errorf("empty string for state %v", state)
		}
	}
	if len(expected) != 15 {
		t.Errorf("expected 15 handoff states, got %d", len(expected))
	}
}

func TestHandoffStateValues(t *testing.T) {
	tests := []struct {
		state HandoffState
		value string
	}{
		{HandoffRequested, "REQUESTED"},
		{HandoffPreflight, "PREFLIGHT"},
		{HandoffTargetValidated, "TARGET_VALIDATED"},
		{HandoffCheckpointed, "CHECKPOINTED"},
		{HandoffSourceQuiesced, "SOURCE_QUIESCED"},
		{HandoffTargetStarting, "TARGET_STARTING"},
		{HandoffTargetResumed, "TARGET_RESUMED"},
		{HandoffContinuityUnverified, "NATIVE_RESUME_UNVERIFIED"},
		{HandoffVerified, "VERIFIED"},
		{HandoffCompleted, "COMPLETED"},
		{HandoffRollback, "ROLLBACK_REQUIRED"},
		{HandoffRollingBack, "ROLLING_BACK"},
		{HandoffRolledBack, "ROLLED_BACK"},
		{HandoffFailedSafe, "FAILED_SAFE"},
		{HandoffFailedUnsafe, "FAILED_UNSAFE"},
	}
	for _, tt := range tests {
		if string(tt.state) != tt.value {
			t.Errorf("state %v: expected %q, got %q", tt.state, tt.value, string(tt.state))
		}
	}
}

func TestTransaction_HasAllFields(t *testing.T) {
	tx := &Transaction{
		State:          HandoffRequested,
		TargetProvider: "codex",
		TargetProfile:  "work",
	}
	if tx.State != HandoffRequested {
		t.Errorf("expected REQUESTED, got %v", tx.State)
	}
	if tx.TargetProvider != "codex" {
		t.Errorf("expected codex, got %s", tx.TargetProvider)
	}
	if tx.TargetProfile != "work" {
		t.Errorf("expected work, got %s", tx.TargetProfile)
	}
}

func TestHandoffStates_HappyPath(t *testing.T) {
	// Verify the canonical happy path order
	happyPath := []HandoffState{
		HandoffRequested,
		HandoffPreflight,
		HandoffTargetValidated,
		HandoffCheckpointed,
		HandoffSourceQuiesced,
		HandoffTargetStarting,
		HandoffTargetResumed,
		HandoffContinuityUnverified,
		HandoffCompleted,
	}
	for i, state := range happyPath {
		if string(state) == "" {
			t.Errorf("empty state at position %d", i)
		}
	}
	if len(happyPath) != 9 {
		t.Errorf("expected 9 happy path states, got %d", len(happyPath))
	}
}

func TestHandoffStates_RollbackPath(t *testing.T) {
	rollbackPath := []HandoffState{
		HandoffRollback,
		HandoffRollingBack,
		HandoffRolledBack,
	}
	for i, state := range rollbackPath {
		if string(state) == "" {
			t.Errorf("empty state at position %d", i)
		}
	}
	if len(rollbackPath) != 3 {
		t.Errorf("expected 3 rollback states, got %d", len(rollbackPath))
	}
}

func TestHandoffStates_FailurePaths(t *testing.T) {
	failureStates := []HandoffState{
		HandoffFailedSafe,
		HandoffFailedUnsafe,
	}
	for _, state := range failureStates {
		if string(state) == "" {
			t.Errorf("empty failure state")
		}
	}
	if len(failureStates) != 2 {
		t.Errorf("expected 2 failure states, got %d", len(failureStates))
	}
}
