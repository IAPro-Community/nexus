package nexus

import (
	"reflect"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

func goalLoopFixture() *runner.MissionRun {
	r := &runner.MissionRun{
		ID: "run_goal_loop", State: runner.StateFailedVerification,
		Contract: runner.DefaultAutonomyContract(), TotalIterations: 3,
		PackageRuns: []runner.PackageRun{
			{PackageID:"architecture",State:runner.StateVerified,Attempt:1,WorkReceipt:&runner.WorkReceipt{Status:"COMPLETED"}},
			{PackageID:"backend",Dependencies:[]string{"architecture"},State:runner.StateVerified,Attempt:1,WorkReceipt:&runner.WorkReceipt{Status:"COMPLETED"}},
			{PackageID:"frontend",Dependencies:[]string{"architecture"},State:runner.StateFailedVerification,Attempt:1,LastFailureSignature:"older-failure"},
			{PackageID:"qa",Dependencies:[]string{"backend","frontend"},State:runner.StateVerified,Attempt:1,WorkReceipt:&runner.WorkReceipt{Status:"COMPLETED"}},
		},
	}
	return r
}

func TestGoalLoopPreviewSelectsOnlyFailedBranchAndDownstreamVerification(t *testing.T) {
	run:=goalLoopFixture()
	before := make([]runner.PackageRun,len(run.PackageRuns))
	copy(before,run.PackageRuns)
	preview:=PreviewGoalLoop(run,GoalAssessment{
		Result:GoalAssessmentFail,Violations:[]string{"Frontend auth callback failed integration tests"},
	},"frontend","fingerprint-auth-callback")
	if preview.Decision!="REPAIR_CANDIDATE" || !preview.RequiresApproval {
		t.Fatalf("repair must be explicit review candidate: %+v",preview)
	}
	if !reflect.DeepEqual(preview.AffectedPackageIDs,[]string{"frontend","qa"}) {
		t.Fatalf("wrong invalidation impact: %+v",preview.AffectedPackageIDs)
	}
	if !reflect.DeepEqual(preview.PreservedPackageIDs,[]string{"architecture","backend"}) {
		t.Fatalf("verified independent work lost: %+v",preview.PreservedPackageIDs)
	}
	if !reflect.DeepEqual(before,run.PackageRuns) {
		t.Fatal("read-only Goal Loop mutated the MissionRun")
	}
}

func TestGoalLoopPreviewRejectsUnverifiedOrUnscopedVerdicts(t *testing.T) {
	run:=goalLoopFixture()
	for _,tt:=range []struct{
		name string
		assessment GoalAssessment
		source string
		fingerprint string
	}{
		{"unsupported",GoalAssessment{},"frontend","fingerprint"},
		{"pass without runner verification",GoalAssessment{Result:GoalAssessmentPass},"",""},
		{"no origin",GoalAssessment{Result:GoalAssessmentRedo,Violations:[]string{"test failure"}},"","fingerprint"},
		{"no evidence",GoalAssessment{Result:GoalAssessmentFail},"frontend","fingerprint"},
		{"no fingerprint",GoalAssessment{Result:GoalAssessmentFail,Violations:[]string{"failed"}},"frontend",""},
		{"unknown origin",GoalAssessment{Result:GoalAssessmentFail,Violations:[]string{"failed"}},"unknown","fingerprint"},
	}{
		t.Run(tt.name,func(t *testing.T){
			got:=PreviewGoalLoop(run,tt.assessment,tt.source,tt.fingerprint)
			if got.Decision!="BLOCKED_NEEDS_USER" || !got.RequiresApproval {
				t.Fatalf("unverified proposal incorrectly admitted: %+v",got)
			}
		})
	}
}

func TestGoalLoopPreviewBlocksUncertainDispatchAndRepeatedFailures(t *testing.T) {
	check:=func(name string,configure func(*runner.MissionRun)) {
		t.Run(name,func(t *testing.T){
			run:=goalLoopFixture()
			configure(run)
			preview:=PreviewGoalLoop(run,GoalAssessment{Result:GoalAssessmentFail,
				Violations:[]string{"Integration check failed"}},"frontend","same-failure")
			if preview.Decision!="BLOCKED_NEEDS_USER" {
				t.Fatalf("unsafe repair accepted: %+v",preview)
			}
		})
	}
	check("dispatch intent",func(r *runner.MissionRun){r.PackageRuns[2].DispatchState=runner.DispatchIntent})
	check("unknown external outcome",func(r *runner.MissionRun){r.PackageRuns[2].DispatchState=runner.DispatchUnknownExternalOutcome})
	check("replay prohibited",func(r *runner.MissionRun){r.PackageRuns[2].ReplayProhibited=true})
	check("repeated fingerprint",func(r *runner.MissionRun){
		r.PackageRuns[2].LastFailureSignature="same-failure"
		r.PackageRuns[2].NoProgressCount=1
	})
	check("retry exhausted",func(r *runner.MissionRun){r.PackageRuns[2].Attempt=r.Contract.MaxRetries})
	check("iteration budget",func(r *runner.MissionRun){r.TotalIterations=r.Contract.MaxTotalIterations})
	check("auto remediation disabled",func(r *runner.MissionRun){r.Contract.AutoRemediate=false})
	check("existing intervention",func(r *runner.MissionRun){r.NeedsHuman=&runner.HumanIntervention{ID:"intervention_pending"}})
	check("run cancelled",func(r *runner.MissionRun){r.State=runner.StateCanceledByUser})
}

func TestGoalLoopPreviewRequiresFactualEvidenceForPass(t *testing.T) {
	run:=goalLoopFixture()
	run.State=runner.StateCompletedVerified
	for i:=range run.PackageRuns {
		run.PackageRuns[i].State=runner.StateVerified
		run.PackageRuns[i].WorkReceipt=&runner.WorkReceipt{Status:"COMPLETED"}
	}
	pass:=PreviewGoalLoop(run,GoalAssessment{Result:GoalAssessmentPass},"","")
	if pass.Decision!="VERIFIED" || pass.RequiresApproval {
		t.Fatalf("verified run was not recognized: %+v",pass)
	}
	run.PackageRuns[2].WorkReceipt=nil
	blocked:=PreviewGoalLoop(run,GoalAssessment{Result:GoalAssessmentPass},"","")
	if blocked.Decision!="BLOCKED_NEEDS_USER" || !strings.Contains(blocked.Reason,"receipts") {
		t.Fatalf("unsupported PASS was promoted without evidence: %+v",blocked)
	}
}

func TestGoalLoopPreviewRejectsBrokenTopology(t *testing.T) {
	run:=goalLoopFixture()
	run.PackageRuns[3].Dependencies=[]string{"unknown"}
	got:=PreviewGoalLoop(run,GoalAssessment{Result:GoalAssessmentFail,Violations:[]string{"QA failed"}},"frontend","digest")
	if got.Decision!="BLOCKED_NEEDS_USER" {
		t.Fatalf("broken DAG dependencies were ignored: %+v",got)
	}
}
