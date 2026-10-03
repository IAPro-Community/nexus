package nexus

import "strings"

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
