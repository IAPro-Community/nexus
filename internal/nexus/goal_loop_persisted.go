package nexus

import (
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

// PreviewGoalLoopFromPersistedRun interprets only factual MissionRunner state.
// The result is advisory/read-only, not authorization to retry a package.
// No client-supplied GoalAssessment or fabricated failure fingerprint is used.
func PreviewGoalLoopFromPersistedRun(run *runner.MissionRun) GoalLoopPreview {
	if run == nil {
		return PreviewGoalLoop(nil, GoalAssessment{}, "", "")
	}
	if run.State == runner.StateCompletedVerified {
		return PreviewGoalLoop(run, GoalAssessment{Result: GoalAssessmentPass}, "", "")
	}

	var failed *runner.PackageRun
	for i := range run.PackageRuns {
		pkg := &run.PackageRuns[i]
		switch pkg.State {
		case runner.StateFailed, runner.StateFailedVerification,
			runner.StateFailedNoProgress, runner.StateRemediating:
			if failed != nil {
				return GoalLoopPreview{
					RunID: run.ID, Decision: "BLOCKED_NEEDS_USER",
					Reason: "multiple WorkPackages have failed; choose a specific evidence-backed failure before repair",
					RequiresApproval: true,
				}
			}
			failed = pkg
		}
	}
	if failed == nil {
		return GoalLoopPreview{
			RunID: run.ID, Decision: "BLOCKED_NEEDS_USER",
			Reason: "no persisted failed WorkPackage is available for a Goal Loop assessment",
			RequiresApproval: true,
		}
	}
	fingerprint := strings.TrimSpace(failed.LastFailureSignature)
	if fingerprint == "" {
		return GoalLoopPreview{
			RunID: run.ID, Decision: "BLOCKED_NEEDS_USER",
			FailedPackageID: failed.PackageID,
			Reason: "a stable persisted failure signature is required before proposing a targeted repair",
			RequiresApproval: true,
		}
	}
	// The model does not produce this verdict; it comes from a recorded
	// failure on the selected PackageRun. Do not echo raw error text.
	return PreviewGoalLoop(run, GoalAssessment{
		Result: GoalAssessmentFail,
		Violations: []string{"MissionRunner recorded a failure for the selected WorkPackage"},
	}, failed.PackageID, fingerprint)
}
