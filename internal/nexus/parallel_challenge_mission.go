package nexus

import (
	"context"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

// Mission-native parallel challenge (capability parallel-challenge, dispatch
// half). A challenge is one plan specification dispatched to two or more
// leads; each completed MissionRun is one participant with its own isolated
// workspace. Evidence comes exclusively from the runs themselves (work
// receipts and global verifications): this mapping never fabricates numbers.

// MissionChallengeParticipants maps completed MissionRuns of the same plan
// into challenge participants. Rules:
//   - at least two runs, all for the same plan, all completed-verified —
//     failed or still-running runs carry no decisive evidence and are refused
//     instead of being silently compared;
//   - label = lead agent id (fallback: run id when the run has no lead);
//   - worktree = the run's own workspace (mission runs are worktree-isolated
//     by construction — see mission_executor);
//   - tests passed/failed counted from work receipts and global verifications;
//   - cost evidence: none exists on MissionRun today, so CostKnown stays
//     false — UNKNOWN is never treated as zero.
func MissionChallengeParticipants(runs []runner.MissionRun) ([]ChallengeParticipant, error) {
	if len(runs) < 2 {
		return nil, fmt.Errorf("parallel challenge: at least two completed runs are required, got %d", len(runs))
	}
	planID := strings.TrimSpace(runs[0].PlanID)
	if planID == "" {
		return nil, fmt.Errorf("parallel challenge: runs must belong to a plan")
	}
	participants := make([]ChallengeParticipant, 0, len(runs))
	labels := make(map[string]bool, len(runs))
	worktrees := make(map[string]string, len(runs))
	for _, run := range runs {
		if strings.TrimSpace(run.PlanID) != planID {
			return nil, fmt.Errorf("parallel challenge: run %s belongs to plan %s, expected %s", run.ID, run.PlanID, planID)
		}
		if run.State != runner.StateCompleted {
			return nil, fmt.Errorf("parallel challenge: run %s is %s; only completed runs carry decisive evidence", run.ID, run.State)
		}
		if strings.TrimSpace(run.Workspace) == "" {
			return nil, fmt.Errorf("parallel challenge: run %s has no workspace; isolation cannot be proven", run.ID)
		}
		if other, clash := worktrees[run.Workspace]; clash {
			return nil, fmt.Errorf("parallel challenge: runs %s and %s share workspace %s; isolation is required", other, run.ID, run.Workspace)
		}
		worktrees[run.Workspace] = run.ID

		label := strings.TrimSpace(run.LeadAgentID)
		if label == "" {
			label = run.ID
		}
		// Ranking keys must stay unique when one lead ran the plan twice.
		if labels[label] {
			label = fmt.Sprintf("%s@%s", label, run.ID)
		}
		labels[label] = true

		passed, failed := countRunTestEvidence(run)
		participants = append(participants, ChallengeParticipant{
			Label:        label,
			AgentID:      run.LeadAgentID,
			WorktreePath: run.Workspace,
			TestsPassed:  passed,
			TestsFailed:  failed,
			CostKnown:    false,
			CostTotal:    0,
		})
	}
	return participants, nil
}

// countRunTestEvidence sums verification outcomes from every work receipt and
// the run's global verifications. Absent receipts count as zero evidence —
// they lower the score honestly instead of inventing a pass.
func countRunTestEvidence(run runner.MissionRun) (passed, failed int) {
	tally := func(results []runner.VerificationResult) {
		for _, verification := range results {
			if verification.Passed {
				passed++
			} else {
				failed++
			}
		}
	}
	tally(run.GlobalVerifications)
	for _, packageRun := range run.PackageRuns {
		if packageRun.WorkReceipt == nil {
			continue
		}
		tally(packageRun.WorkReceipt.Tests)
		tally(packageRun.WorkReceipt.Verification)
	}
	return passed, failed
}

// EvaluatePlanChallenge evaluates a parallel challenge over every completed
// MissionRun of one plan. This is the dispatch/decision half of the
// capability: the runs themselves are dispatched by the existing MissionRunner
// (worktree-isolated by construction), and the result never merges anything.
func (n *Nexus) EvaluatePlanChallenge(ctx context.Context, planID string, criteria []AgentCompareCriterion) (*ChallengeResult, error) {
	if n == nil || n.Runner() == nil {
		return nil, fmt.Errorf("parallel challenge: mission runner is unavailable")
	}
	if strings.TrimSpace(planID) == "" {
		return nil, fmt.Errorf("parallel challenge: plan id is required")
	}
	runs, err := n.Runner().ListRuns(ctx)
	if err != nil {
		return nil, fmt.Errorf("parallel challenge: failed to list mission runs: %w", err)
	}
	planRuns := make([]runner.MissionRun, 0, len(runs))
	for _, run := range runs {
		if strings.EqualFold(strings.TrimSpace(run.PlanID), strings.TrimSpace(planID)) {
			planRuns = append(planRuns, *run)
		}
	}
	participants, err := MissionChallengeParticipants(planRuns)
	if err != nil {
		return nil, err
	}
	specification := fmt.Sprintf("plan:%s", planID)
	return EvaluateChallenge(specification, participants, criteria)
}
