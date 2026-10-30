package nexus

import (
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

// GoalLoopPreview is a read-only decision projection. It never dispatches,
// retries, mutates an execution snapshot or changes MissionRunner ownership.
// Only a future admission-controlled coordination operation can apply a repair.
type GoalLoopPreview struct {
	RunID               string   `json:"run_id"`
	Decision            string   `json:"decision"` // VERIFIED | REPAIR_CANDIDATE | BLOCKED_NEEDS_USER
	Reason              string   `json:"reason"`
	FailedPackageID     string   `json:"failed_package_id,omitempty"`
	FailureFingerprint  string   `json:"failure_fingerprint,omitempty"`
	AffectedPackageIDs  []string `json:"affected_package_ids,omitempty"`
	PreservedPackageIDs []string `json:"preserved_package_ids,omitempty"`
	RequiresApproval    bool     `json:"requires_approval"`
}

// PreviewGoalLoop performs deterministic, conservative dependency impact
// analysis against the real MissionRun's frozen package topology. The input
// verdict is a proposal: provider claims and GoalAssessment alone are NOT
// completion evidence. A proposed repair is never an automatic GO.
func PreviewGoalLoop(run *runner.MissionRun, assessment GoalAssessment, failedPackageID, fingerprint string) GoalLoopPreview {
	block := func(reason string) GoalLoopPreview {
		id := ""
		if run != nil {
			id = run.ID
		}
		return GoalLoopPreview{
			RunID: id, Decision: "BLOCKED_NEEDS_USER", Reason: reason,
			FailedPackageID: strings.TrimSpace(failedPackageID),
			FailureFingerprint: strings.TrimSpace(fingerprint),
			RequiresApproval: true,
		}
	}
	if run == nil || strings.TrimSpace(run.ID) == "" {
		return block("an identifiable, persisted MissionRun is required")
	}
	if len(run.PackageRuns) == 0 {
		return block("no persisted WorkPackages are available for Goal Loop verification")
	}
	byID := make(map[string]*runner.PackageRun, len(run.PackageRuns))
	for i := range run.PackageRuns {
		pkg := &run.PackageRuns[i]
		if pkg.PackageID == "" {
			return block("MissionRun contains a WorkPackage without identity")
		}
		if _, duplicate := byID[pkg.PackageID]; duplicate {
			return block("MissionRun contains duplicate WorkPackage identity")
		}
		byID[pkg.PackageID] = pkg
	}
	for i := range run.PackageRuns {
		for _, dep := range run.PackageRuns[i].Dependencies {
			if _, exists := byID[dep]; !exists {
				return block(fmt.Sprintf("WorkPackage %s has an unknown dependency", run.PackageRuns[i].PackageID))
			}
		}
	}
	switch assessment.Result {
	case GoalAssessmentPass:
		if run.State != runner.StateCompletedVerified {
			return block("PASS cannot be promoted before the MissionRunner has verified completion")
		}
		for i := range run.PackageRuns {
			pkg := &run.PackageRuns[i]
			if pkg.State != runner.StateVerified || pkg.WorkReceipt == nil ||
				!strings.EqualFold(pkg.WorkReceipt.Status, "COMPLETED") {
				return block("PASS requires verified WorkPackages and factual completion receipts")
			}
		}
		return GoalLoopPreview{RunID: run.ID, Decision: "VERIFIED", Reason: "the MissionRunner and all factual WorkReceipts confirm completion"}
	case GoalAssessmentFail, GoalAssessmentRedo:
		// Require explicit traceability of the failing workstream. Never infer
		// it from a free-text AI diagnosis and invalidate an arbitrary step.
	default:
		return block("unsupported or missing GoalAssessment verdict")
	}
	failedPackageID = strings.TrimSpace(failedPackageID)
	fingerprint = strings.TrimSpace(fingerprint)
	if failedPackageID == "" || fingerprint == "" || len(assessment.Violations) == 0 {
		return block("a specific failed package, stable failure fingerprint and recorded violations are required")
	}
	source, ok := byID[failedPackageID]
	if !ok {
		return block("failed package does not belong to this MissionRun")
	}
	if run.NeedsHuman != nil {
		return block("the MissionRunner already has a pending human intervention")
	}
	switch run.State {
	case runner.StateCanceledByUser, runner.StateCompletedVerified,
		runner.StateFailedNoProgress, runner.StateFailedBudgetExceeded:
		return block("the MissionRun state disallows unattended repair")
	}
	if source.ReplayProhibited ||
		source.DispatchState == runner.DispatchIntent ||
		source.DispatchState == runner.DispatchUnknownExternalOutcome {
		return block("provider dispatch outcome is unresolved or replay is prohibited")
	}
	if run.Contract.MaxRetries <= 0 || source.Attempt >= run.Contract.MaxRetries ||
		run.Contract.MaxTotalIterations <= 0 || run.TotalIterations >= run.Contract.MaxTotalIterations ||
		run.Contract.MaxNoProgress <= 0 || source.NoProgressCount >= run.Contract.MaxNoProgress {
		return block("remediation attempt, iteration or no-progress budget is exhausted")
	}
	if source.LastFailureSignature == fingerprint && source.NoProgressCount > 0 {
		return block("the identical failure repeated without verified progress")
	}
	if !run.Contract.AutoRemediate {
		return block("automatic remediation is disabled by the autonomy contract")
	}

	// Only the failed package and its transitive dependents are affected.
	// This never resets any package: it is an inspection-only projection.
	affected := map[string]bool{failedPackageID: true}
	for changed := true; changed; {
		changed = false
		for i := range run.PackageRuns {
			pkg := &run.PackageRuns[i]
			if affected[pkg.PackageID] {
				continue
			}
			for _, dependency := range pkg.Dependencies {
				if affected[dependency] {
					affected[pkg.PackageID] = true
					changed = true
					break
				}
			}
		}
	}
	result := GoalLoopPreview{
		RunID: run.ID, Decision: "REPAIR_CANDIDATE",
		Reason: "targeted repair requires coordination admission and explicit approval; no task is replayed",
		FailedPackageID: failedPackageID, FailureFingerprint: fingerprint,
		RequiresApproval: true,
	}
	for i := range run.PackageRuns {
		pkg := &run.PackageRuns[i]
		if affected[pkg.PackageID] {
			result.AffectedPackageIDs = append(result.AffectedPackageIDs, pkg.PackageID)
		} else if pkg.State == runner.StateVerified && pkg.WorkReceipt != nil {
			result.PreservedPackageIDs = append(result.PreservedPackageIDs, pkg.PackageID)
		}
	}
	return result
}
