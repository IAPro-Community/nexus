package nexus

import "testing"

func TestDecideCandidateExecutionAvoidsEasyHighConfidenceFanout(t *testing.T) {
	decision := DecideCandidateExecution(TaskRequirements{
		TaskKind: "coding", EstimatedComplexity: "low", EstimatedRisk: "low", Confidence: "HIGH",
	})
	if decision.Enabled || decision.CandidateCount != 1 {
		t.Fatalf("easy/high-confidence work must stay single-run: %+v", decision)
	}
}

func TestDecideCandidateExecutionUsesTwoForHighRiskMediumComplexity(t *testing.T) {
	decision := DecideCandidateExecution(TaskRequirements{
		TaskKind: "coding", EstimatedComplexity: "medium", EstimatedRisk: "high", Confidence: "HIGH",
	})
	if !decision.Enabled || decision.CandidateCount != 2 || !decision.RequiresIndependentEvaluation {
		t.Fatalf("high-risk work should justify two candidates: %+v", decision)
	}
}

func TestDecideCandidateExecutionCapsAtThree(t *testing.T) {
	decision := DecideCandidateExecution(TaskRequirements{
		TaskKind: "security", EstimatedComplexity: "high", EstimatedRisk: "critical", Confidence: "LOW", RequiresDecomposition: true,
	})
	if decision.CandidateCount != 3 {
		t.Fatalf("candidate fan-out must be capped at three: %+v", decision)
	}
}

func TestDecideCandidateExecutionDoesNotFanoutReview(t *testing.T) {
	decision := DecideCandidateExecution(TaskRequirements{
		TaskKind: "review", EstimatedComplexity: "high", EstimatedRisk: "critical", Confidence: "LOW",
	})
	if decision.Enabled || decision.CandidateCount != 1 {
		t.Fatalf("review must remain evaluation work, not competing implementation: %+v", decision)
	}
}
