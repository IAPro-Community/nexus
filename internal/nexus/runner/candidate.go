package runner

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

type candidateScore struct {
	pkg       *PackageRun
	risk      int
	failures  int
	index     int
	packageID string
}

func candidateDeliveryEligible(pkg *PackageRun) bool {
	if pkg == nil {
		return false
	}
	if strings.TrimSpace(pkg.CandidateGroupID) == "" {
		return true
	}
	if pkg.CandidateCoordinator {
		return false
	}
	return pkg.CandidateDisposition == CandidateSelected
}

func (r *MissionRunner) finalizeCandidateCoordinator(ctx context.Context, run *MissionRun, coordinator *PackageRun) error {
	if run == nil || coordinator == nil || !coordinator.CandidateCoordinator {
		return fmt.Errorf("candidate coordinator is required")
	}
	groupID := strings.TrimSpace(coordinator.CandidateGroupID)
	if groupID == "" {
		return fmt.Errorf("candidate coordinator %s has no group id", coordinator.PackageID)
	}

	candidates := make([]*PackageRun, 0, 3)
	for i := range run.PackageRuns {
		pkg := &run.PackageRuns[i]
		if pkg.CandidateCoordinator || pkg.CandidateGroupID != groupID {
			continue
		}
		candidates = append(candidates, pkg)
	}
	if len(candidates) < 2 || len(candidates) > 3 {
		return fmt.Errorf("candidate group %s has %d implementation candidates; expected 2 or 3", groupID, len(candidates))
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].CandidateIndex != candidates[j].CandidateIndex {
			return candidates[i].CandidateIndex < candidates[j].CandidateIndex
		}
		return candidates[i].PackageID < candidates[j].PackageID
	})

	scores := make([]candidateScore, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.State != StateVerified || candidate.WorkReceipt == nil || candidate.WorkReceipt.Status != "VERIFIED" {
			return fmt.Errorf("candidate %s is not durably verified", candidate.PackageID)
		}
		verdict, ok := latestIndependentApprovedVerdict(candidate.Verdicts)
		if !ok {
			continue
		}
		scores = append(scores, candidateScore{
			pkg:       candidate,
			risk:      unresolvedFindingRisk(verdict.StructuredFindings),
			failures:  failedVerificationCount(candidate.Verifications),
			index:     candidate.CandidateIndex,
			packageID: candidate.PackageID,
		})
	}
	if len(scores) == 0 {
		return fmt.Errorf("candidate group %s has no independently approved candidate", groupID)
	}

	sort.SliceStable(scores, func(i, j int) bool {
		if scores[i].risk != scores[j].risk {
			return scores[i].risk < scores[j].risk
		}
		if scores[i].failures != scores[j].failures {
			return scores[i].failures < scores[j].failures
		}
		if scores[i].index != scores[j].index {
			return scores[i].index < scores[j].index
		}
		return scores[i].packageID < scores[j].packageID
	})
	winner := scores[0].pkg

	candidateIDs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidateIDs = append(candidateIDs, candidate.PackageID)
		if candidate.PackageID == winner.PackageID {
			candidate.CandidateDisposition = CandidateSelected
		} else {
			candidate.CandidateDisposition = CandidateRejected
		}
	}

	now := time.Now().UTC()
	reason := fmt.Sprintf(
		"APPLY selected %s using independent review evidence (unresolved-risk=%d, failed-verifications=%d); ties use candidate index then package id",
		winner.PackageID,
		scores[0].risk,
		scores[0].failures,
	)
	coordinator.CandidateDisposition = CandidateCoordinator
	coordinator.CandidateEvaluation = &CandidateEvaluation{
		GroupID:             groupID,
		Strategy:            "APPLY",
		WinnerPackageID:     winner.PackageID,
		CandidatePackageIDs: candidateIDs,
		Reason:              reason,
		EvaluatedAt:         now,
	}
	coordinator.AssignedAgent = winner.AssignedAgent
	coordinator.Provider = winner.Provider
	coordinator.Profile = winner.Profile
	coordinator.ResultSummary = reason
	coordinator.State = StateVerified
	coordinator.FinishedAt = &now

	receipt := *winner.WorkReceipt
	receipt.ID = "receipt_" + ids.NewRuntimeID()
	receipt.StepID = coordinator.PackageID
	receipt.Summary = reason
	receipt.AgentID = winner.AssignedAgent
	receipt.StartedAt = coordinator.StartedAt
	receipt.CompletedAt = now
	receipt.Decisions = append(append([]string(nil), winner.WorkReceipt.Decisions...), reason)
	receipt.ChangedFiles = append([]string(nil), winner.WorkReceipt.ChangedFiles...)
	receipt.Commands = append([]string(nil), winner.WorkReceipt.Commands...)
	receipt.Tests = append([]VerificationResult(nil), winner.WorkReceipt.Tests...)
	receipt.Artifacts = append([]string(nil), winner.WorkReceipt.Artifacts...)
	receipt.RemainingIssues = append([]string(nil), winner.WorkReceipt.RemainingIssues...)
	receipt.Verification = append([]VerificationResult(nil), winner.WorkReceipt.Verification...)

	evidence, err := evidenceRepository(r.repo)
	if err != nil {
		return err
	}
	if err := evidence.SaveWorkReceipt(ctx, &receipt); err != nil {
		return fmt.Errorf("persist candidate coordinator receipt: %w", err)
	}
	coordinator.WorkReceipt = &receipt
	return nil
}

func latestIndependentApprovedVerdict(verdicts []ReviewVerdict) (ReviewVerdict, bool) {
	for i := len(verdicts) - 1; i >= 0; i-- {
		verdict := verdicts[i]
		if verdict.Approved && strings.EqualFold(strings.TrimSpace(verdict.ReviewIndependence), "DISTINCT_AGENT") {
			return verdict, true
		}
	}
	return ReviewVerdict{}, false
}

func unresolvedFindingRisk(findings []ReviewFinding) int {
	score := 0
	for _, finding := range findings {
		if finding.Status == FindingResolved {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(finding.Severity)) {
		case "critical":
			score += 10000
		case "high":
			score += 1000
		case "medium":
			score += 100
		case "low":
			score += 10
		default:
			score += 25
		}
	}
	return score
}

func failedVerificationCount(results []VerificationResult) int {
	count := 0
	for _, result := range results {
		if !result.Passed {
			count++
		}
	}
	return count
}
