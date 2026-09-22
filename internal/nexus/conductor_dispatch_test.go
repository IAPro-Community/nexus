package nexus

import (
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

// PAUSED and BLOCKED_NEEDS_USER are non-terminal: reporting them as
// COMPLETED would let callers assume success for a mission that is still
// alive and waiting on a human.
func TestDispatchStatusNeverCompletesNonTerminalStates(t *testing.T) {
	cases := []struct {
		state runner.State
		want  DispatchStatus
	}{
		{runner.StateCompletedVerified, DispatchCompleted},
		{runner.StateFailed, DispatchFailed},
		{runner.StateFailedNoProgress, DispatchFailed},
		{runner.StateCanceledByUser, DispatchCanceled},
		{runner.StateBlockedNeedsUser, DispatchBlocked},
		{runner.StatePaused, DispatchPaused},
		// In-flight states have no honest completion mapping; they must
		// not surface as COMPLETED either.
		{runner.StateExecuting, DispatchFailed},
		{runner.StatePending, DispatchFailed},
	}
	for _, tc := range cases {
		if got := dispatchStatusForRunState(tc.state); got != tc.want {
			t.Errorf("dispatchStatusForRunState(%q) = %q, want %q", tc.state, got, tc.want)
		}
	}
}

// Goal IDs for persistent/distributed entities must be collision-resistant
// and sortable: ULID, not UnixNano.
func TestGeneratedGoalIDIsULID(t *testing.T) {
	id := "goal_" + ids.NewULID()
	if !strings.HasPrefix(id, "goal_") {
		t.Fatalf("goal id %q missing prefix", id)
	}
	if _, ok := ids.DecodeULID(strings.TrimPrefix(id, "goal_")); !ok {
		t.Fatalf("goal id %q suffix is not a valid ULID", id)
	}
	other := "goal_" + ids.NewULID()
	if id == other {
		t.Fatal("consecutive goal IDs collided")
	}
}


func TestDispatchTopologySeparatesMissionPromotionFromDelegation(t *testing.T) {
	nonDelegated := DelegationDecision{Delegate: false}
	if dispatchTopologyAllowed(DispatchRequest{}, nonDelegated) {
		t.Fatal("ordinary Conductor dispatch must keep the existing delegation brake")
	}
	if !dispatchTopologyAllowed(DispatchRequest{AutoPromoteToMission: true}, nonDelegated) {
		t.Fatal("an upstream-approved conversation promotion must be allowed to create a single-topology Mission")
	}
	if !dispatchTopologyAllowed(DispatchRequest{}, DelegationDecision{Delegate: true}) {
		t.Fatal("delegated topology must remain dispatchable")
	}
}
