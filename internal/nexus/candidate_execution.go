package nexus

import (
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/flags"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

// CandidateExecutionDecision is a bounded, explainable decision about whether
// multiple independent implementations are worth their additional cost.
// Candidate execution is never selected merely because a cheaper model exists.
type CandidateExecutionDecision struct {
	Enabled                       bool     `json:"enabled"`
	CandidateCount                int      `json:"candidate_count"`
	Score                         int      `json:"score"`
	Reason                        string   `json:"reason"`
	Factors                       []string `json:"factors,omitempty"`
	RequiresIndependentEvaluation bool     `json:"requires_independent_evaluation,omitempty"`
}

func DecideCandidateExecution(req TaskRequirements) CandidateExecutionDecision {
	decision := CandidateExecutionDecision{CandidateCount: 1}
	taskKind := strings.ToLower(strings.TrimSpace(req.TaskKind))
	if taskKind == "review" || taskKind == "verify" || taskKind == "documentation" {
		decision.Reason = "task type does not benefit from competing implementation candidates"
		return decision
	}

	complexity := strings.ToLower(strings.TrimSpace(req.EstimatedComplexity))
	risk := strings.ToLower(strings.TrimSpace(req.EstimatedRisk))
	confidence := strings.ToLower(strings.TrimSpace(req.Confidence))

	score := 0
	switch complexity {
	case "high", "complex", "very_high":
		score += 2
		decision.Factors = append(decision.Factors, "high complexity")
	case "medium", "intermediate":
		score++
		decision.Factors = append(decision.Factors, "medium complexity")
	}
	switch risk {
	case "critical":
		score += 3
		decision.Factors = append(decision.Factors, "critical risk")
	case "high":
		score += 2
		decision.Factors = append(decision.Factors, "high risk")
	case "medium":
		score++
		decision.Factors = append(decision.Factors, "medium risk")
	}
	switch confidence {
	case "low":
		score += 2
		decision.Factors = append(decision.Factors, "low routing confidence")
	case "medium":
		score++
		decision.Factors = append(decision.Factors, "medium routing confidence")
	}
	if req.RequiresDecomposition {
		score++
		decision.Factors = append(decision.Factors, "requires decomposition")
	}
	if taskKind == "security" || taskKind == "architecture" {
		score++
		decision.Factors = append(decision.Factors, taskKind+" specialty")
	}

	decision.Score = score
	switch {
	case score >= 6:
		decision.CandidateCount = 3
	case score >= 4:
		decision.CandidateCount = 2
	default:
		decision.CandidateCount = 1
	}
	decision.Enabled = decision.CandidateCount > 1
	decision.RequiresIndependentEvaluation = decision.Enabled
	if decision.Enabled {
		decision.Reason = "independent candidates are justified by specialty/complexity/risk/confidence"
	} else {
		decision.Reason = "single execution is sufficient; candidate fan-out would add cost without enough expected value"
	}
	return decision
}


func constrainCandidateExecution(decision CandidateExecutionDecision, assignmentStrategy, agentAllocation string, autonomous bool) CandidateExecutionDecision {
	if !decision.Enabled {
		return decision
	}
	if !autonomous {
		decision.Enabled = false
		decision.CandidateCount = 1
		decision.RequiresIndependentEvaluation = false
		decision.Reason = "candidate fan-out is only available for autonomous approved Mission execution"
		return decision
	}
	strategy := strings.ToUpper(strings.TrimSpace(assignmentStrategy))
	if strategy == string(FlowAssignmentExisting) || strings.TrimSpace(agentAllocation) != "" {
		decision.Enabled = false
		decision.CandidateCount = 1
		decision.RequiresIndependentEvaluation = false
		decision.Reason = "explicit Agent assignment is authoritative; Nexus will not multiply a pinned Flow Step"
	}
	return decision
}

// expandCandidatePlanSpec turns an approved WorkPackage into N implementation
// packages plus one synthetic coordinator that preserves the original package
// ID for downstream dependencies. Only the approved Execution Preview may
// request this expansion.
func expandCandidatePlanSpec(spec runner.PlanSpec, preview *PlanExecutionPreview) (runner.PlanSpec, error) {
	if !flags.Enabled(flags.CandidateRuns) {
		return spec, nil
	}
	if !flags.Enabled(flags.ExecutionWorkspacesV2) {
		return runner.PlanSpec{}, fmt.Errorf("candidate_runs requires execution_workspaces_v2")
	}
	if preview == nil || preview.PlanID != spec.ID || preview.Revision != spec.Revision {
		return runner.PlanSpec{}, fmt.Errorf("candidate execution requires the approved execution preview for plan revision %d", spec.Revision)
	}

	counts := make(map[string]int, len(preview.Steps))
	for _, step := range preview.Steps {
		count := step.CandidateExecution.CandidateCount
		if count <= 0 {
			count = 1
		}
		if count > 3 {
			return runner.PlanSpec{}, fmt.Errorf("candidate count %d for package %s exceeds the supported maximum of 3", count, step.PackageID)
		}
		counts[step.PackageID] = count
	}

	known := make(map[string]struct{}, len(spec.Packages)*4)
	for _, pkg := range spec.Packages {
		known[pkg.ID] = struct{}{}
	}
	out := make([]runner.PackageSpec, 0, len(spec.Packages)*2)
	for _, original := range spec.Packages {
		count := counts[original.ID]
		if count <= 1 {
			out = append(out, original)
			continue
		}
		if strings.TrimSpace(original.AgentAllocation) != "" ||
			strings.EqualFold(strings.TrimSpace(original.AssignmentStrategy), string(FlowAssignmentExisting)) {
			return runner.PlanSpec{}, fmt.Errorf("package %s has an explicit Agent assignment and cannot fan out", original.ID)
		}

		groupID := "candidate:" + original.ID
		candidateIDs := make([]string, 0, count)
		for index := 1; index <= count; index++ {
			candidateID := fmt.Sprintf("%s__candidate_%d", original.ID, index)
			if _, exists := known[candidateID]; exists {
				return runner.PlanSpec{}, fmt.Errorf("candidate package id %s collides with an existing Flow Step", candidateID)
			}
			known[candidateID] = struct{}{}
			candidateIDs = append(candidateIDs, candidateID)

			candidate := original
			candidate.ID = candidateID
			candidate.Title = fmt.Sprintf("%s · Candidate %d", original.Title, index)
			candidate.Dependencies = append([]string(nil), original.Dependencies...)
			candidate.ParallelGroup = groupID
			candidate.ReviewRequired = true
			candidate.AgentAllocation = ""
			if strings.TrimSpace(candidate.AssignmentStrategy) == "" {
				candidate.AssignmentStrategy = string(FlowAssignmentAuto)
			}
			candidate.CandidateGroupID = groupID
			candidate.CandidateIndex = index
			candidate.CandidateCoordinator = false
			candidate.CandidateDisposition = runner.CandidatePending
			out = append(out, candidate)
		}

		coordinator := original
		coordinator.Dependencies = candidateIDs
		coordinator.ParallelGroup = ""
		coordinator.ReviewRequired = false
		coordinator.AgentAllocation = ""
		coordinator.AssignmentStrategy = ""
		coordinator.CandidateGroupID = groupID
		coordinator.CandidateIndex = 0
		coordinator.CandidateCoordinator = true
		coordinator.CandidateDisposition = runner.CandidateCoordinator
		out = append(out, coordinator)
	}
	spec.Packages = out
	return spec, nil
}
