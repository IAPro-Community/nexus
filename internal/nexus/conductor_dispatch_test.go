package nexus

import (
	"context"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
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

func TestWaitDispatchDoesNotCreateOrWaitForRejectedAdmission(t *testing.T) {
	n := openTestNexus(t)
	rejected := &DispatchResult{
		GoalID: "goal_test",
		Status: DispatchRejected,
		Error:  "strict admission blocked",
	}
	got, err := n.WaitDispatch(context.Background(), rejected)
	if err != nil {
		t.Fatal(err)
	}
	if got != rejected {
		t.Fatalf("non-accepted dispatch must be returned unchanged: got=%+v want=%+v", got, rejected)
	}
}

func TestWaitDispatchRejectsNilAcceptedResult(t *testing.T) {
	n := openTestNexus(t)
	if _, err := n.WaitDispatch(context.Background(), nil); err == nil {
		t.Fatal("nil accepted result must be rejected")
	}
}

func TestWaitDispatchFailureIsNeverMarkedFallbackSafeAfterAcceptance(t *testing.T) {
	n := openTestNexus(t)
	got, err := n.WaitDispatch(context.Background(), &DispatchResult{
		GoalID: "goal_test",
		PlanID: "plan_test",
		Status: DispatchAccepted,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.FallbackSafe {
		t.Fatalf("post-acceptance failure must never be eligible for Direct fallback: %+v", got)
	}
}

func TestStartDispatchReturnsPreparedPlanOnAdmissionFailure(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{
		Name:             "Blocked promotion",
		CanonicalPath:    t.TempDir(),
		DefaultIsolation: "project",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := n.StartDispatch(context.Background(), DispatchRequest{
		ProjectID:            project.ID,
		Goal:                 "implemente backend e frontend com testes",
		SourcePrompt:         "implemente backend e frontend com testes",
		DelegationMode:       DelegationAuto,
		AutoPromoteToMission: true,
		PromotionSource:      "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || result.Status != DispatchRejected {
		t.Fatalf("expected rejected strict admission, got %+v", result)
	}
	if !result.AdmissionBlocked || result.AdmissionStage != "strict_admission" || !result.FallbackSafe {
		t.Fatalf("strict admission failure must be typed and safe for interactive fallback: %+v", result)
	}
	if strings.TrimSpace(result.PlanID) == "" {
		t.Fatalf("admission failure must expose prepared plan lineage: %+v", result)
	}
	if !strings.Contains(strings.ToLower(result.Error), "worktree") {
		t.Fatalf("expected actionable worktree blocker, got %q", result.Error)
	}
}
