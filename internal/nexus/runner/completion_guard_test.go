package runner

import (
	"context"
	"testing"
)

func TestCompleteRunFailsClosedWithPendingIntervention(t *testing.T) {
	r := NewMissionRunner(NewMemoryRunRepository(), &fakeExecutor{reviewOK: true})
	run := &MissionRun{
		ID:    "run-pending-intervention",
		State: StateBlockedNeedsUser,
		PackageRuns: []PackageRun{{
			PackageID: "step",
			State:     StateVerified,
		}},
		NeedsHuman: &HumanIntervention{ID: "intervention-1", Version: 1},
	}

	got, completed, err := r.completeRun(context.Background(), run)
	if err == nil {
		t.Fatal("pending human intervention must block COMPLETED_VERIFIED")
	}
	if completed {
		t.Fatal("pending human intervention reported completion")
	}
	if got == nil || got.State == StateCompletedVerified {
		t.Fatalf("run crossed completion boundary with pending intervention: %+v", got)
	}
}

func TestCompleteRunFailsClosedWithUnverifiedPackage(t *testing.T) {
	r := NewMissionRunner(NewMemoryRunRepository(), &fakeExecutor{reviewOK: true})
	run := &MissionRun{
		ID:    "run-unverified-package",
		State: StateVerifying,
		PackageRuns: []PackageRun{{
			PackageID: "step",
			State:     StateTesting,
		}},
	}

	got, completed, err := r.completeRun(context.Background(), run)
	if err == nil {
		t.Fatal("unverified package must block COMPLETED_VERIFIED")
	}
	if completed {
		t.Fatal("unverified package reported completion")
	}
	if got == nil || got.State == StateCompletedVerified {
		t.Fatalf("run crossed completion boundary with unverified work: %+v", got)
	}
}
