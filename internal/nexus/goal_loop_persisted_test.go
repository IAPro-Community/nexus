package nexus

import (
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

func TestGoalLoopFromPersistedRunUsesRunnerEvidenceOnly(t *testing.T) {
	run := goalLoopFixture()
	preview := PreviewGoalLoopFromPersistedRun(run)
	if preview.Decision != "REPAIR_CANDIDATE" || preview.FailedPackageID != "frontend" {
		t.Fatalf("expected evidence-backed targeted preview: %+v", preview)
	}
	if len(preview.AffectedPackageIDs) != 2 ||
		preview.AffectedPackageIDs[0] != "frontend" || preview.AffectedPackageIDs[1] != "qa" {
		t.Fatalf("preview failed dependency impact: %+v", preview)
	}
	if len(preview.PreservedPackageIDs) != 2 {
		t.Fatalf("independent verified packages were not preserved: %+v", preview)
	}
}

func TestGoalLoopFromPersistedRunBlocksAmbiguousOrUnsignedEvidence(t *testing.T) {
	run := goalLoopFixture()
	run.PackageRuns[2].LastFailureSignature = ""
	got := PreviewGoalLoopFromPersistedRun(run)
	if got.Decision != "BLOCKED_NEEDS_USER" ||
		!strings.Contains(got.Reason, "failure signature") {
		t.Fatalf("unsigned failure must remain blocked: %+v", got)
	}
	run = goalLoopFixture()
	run.PackageRuns[1].State = runner.StateFailed
	run.PackageRuns[1].ErrorMessage = "backend verification failed"
	got = PreviewGoalLoopFromPersistedRun(run)
	if got.Decision != "BLOCKED_NEEDS_USER" ||
		!strings.Contains(got.Reason, "multiple") {
		t.Fatalf("ambiguous failures must remain blocked: %+v", got)
	}
	run = goalLoopFixture()
	run.PackageRuns[2].State = runner.StateVerified
	got = PreviewGoalLoopFromPersistedRun(run)
	if got.Decision != "BLOCKED_NEEDS_USER" {
		t.Fatalf("no failure must not fabricate a Goal Loop repair: %+v", got)
	}
}

func TestGoalLoopFromPersistedRunVerifiesOnlyRunnerCompletedReceipt(t *testing.T) {
	run := goalLoopFixture()
	run.State = runner.StateCompletedVerified
	for i := range run.PackageRuns {
		run.PackageRuns[i].State = runner.StateVerified
		run.PackageRuns[i].WorkReceipt = &runner.WorkReceipt{Status:"COMPLETED"}
	}
	if got:=PreviewGoalLoopFromPersistedRun(run);got.Decision!="VERIFIED" {
		t.Fatalf("verified runner outcome was lost: %+v",got)
	}
	run.PackageRuns[0].WorkReceipt=nil
	if got:=PreviewGoalLoopFromPersistedRun(run);got.Decision!="BLOCKED_NEEDS_USER" {
		t.Fatalf("missing receipt must block PASS: %+v",got)
	}
}
