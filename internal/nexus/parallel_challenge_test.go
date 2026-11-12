package nexus

import (
	"strings"
	"testing"
)

func challengeParticipants() []ChallengeParticipant {
	return []ChallengeParticipant{
		{
			Label:        "alpha",
			WorktreePath: "/wt/alpha",
			TestsPassed:  12,
			TestsFailed:  0,
			CostKnown:    true,
			CostTotal:    2.00,
		},
		{
			Label:        "beta",
			WorktreePath: "/wt/beta",
			TestsPassed:  12,
			TestsFailed:  1,
			CostKnown:    true,
			CostTotal:    1.00,
		},
		{
			Label:        "gamma",
			WorktreePath: "/wt/gamma",
			TestsPassed:  10,
			TestsFailed:  0,
			CostKnown:    true,
			CostTotal:    0.50,
		},
	}
}

func challengeCriteria() []AgentCompareCriterion {
	return []AgentCompareCriterion{
		{Name: "tests", Kind: AgentCompareCriterionTests},
		{Name: "cost", Kind: AgentCompareCriterionCost},
	}
}

func TestEvaluateChallengeValidatesInputs(t *testing.T) {
	participants := challengeParticipants()
	if _, err := EvaluateChallenge("   ", participants, challengeCriteria()); err == nil ||
		!strings.Contains(err.Error(), "specification is required") {
		t.Fatalf("empty spec must fail: %v", err)
	}
	if _, err := EvaluateChallenge("spec", participants[:1], challengeCriteria()); err == nil ||
		!strings.Contains(err.Error(), "at least two") {
		t.Fatalf("single participant must fail: %v", err)
	}
	if _, err := EvaluateChallenge("spec", participants, nil); err == nil ||
		!strings.Contains(err.Error(), "explicit criterion") {
		t.Fatalf("no criteria must fail: %v", err)
	}
	clash := append([]ChallengeParticipant{}, participants...)
	clash[1] = ChallengeParticipant{Label: "beta2", WorktreePath: "/wt/alpha"}
	if _, err := EvaluateChallenge("spec", clash, challengeCriteria()); err == nil ||
		!strings.Contains(err.Error(), "isolation is required") {
		t.Fatalf("shared worktree must fail: %v", err)
	}
	missing := append([]ChallengeParticipant{}, participants...)
	missing[0] = ChallengeParticipant{Label: "alpha", WorktreePath: ""}
	if _, err := EvaluateChallenge("spec", missing, challengeCriteria()); err == nil ||
		!strings.Contains(err.Error(), "requires an isolated worktree") {
		t.Fatalf("missing worktree must fail: %v", err)
	}
}

func TestEvaluateChallengeRanksDeterministically(t *testing.T) {
	participants := challengeParticipants()
	first, err := EvaluateChallenge("Build the panel", participants, challengeCriteria())
	if err != nil {
		t.Fatal(err)
	}
	second, err := EvaluateChallenge("Build the panel", participants, challengeCriteria())
	if err != nil {
		t.Fatal(err)
	}
	// Pairwise scoring: alpha wins tests vs both (2 points); gamma wins cost
	// vs both (2 points, ties alpha on score, loses alphabetically); beta
	// wins cost only vs alpha (1 point) and ties gamma on tests.
	if len(first.Ranking) != 3 || first.Ranking[0] != "alpha" || first.Ranking[1] != "gamma" || first.Ranking[2] != "beta" {
		t.Fatalf("ranking = %v, want [alpha gamma beta]", first.Ranking)
	}
	if first.Scores["alpha"] != 2 || first.Scores["gamma"] != 2 || first.Scores["beta"] != 1 {
		t.Fatalf("scores = %v, want alpha=2 gamma=2 beta=1", first.Scores)
	}
	for i := range first.Ranking {
		if first.Ranking[i] != second.Ranking[i] || first.Scores[first.Ranking[i]] != second.Scores[first.Ranking[i]] {
			t.Fatalf("evaluation must be deterministic: %+v vs %+v", first, second)
		}
	}
}

func TestEvaluateChallengeUnknownCostNeverZero(t *testing.T) {
	participants := challengeParticipants()
	// gamma would win cost but has unknown cost evidence.
	participants[2].CostKnown = false
	result, err := EvaluateChallenge("spec", participants, challengeCriteria())
	if err != nil {
		t.Fatal(err)
	}
	foundUnknown := false
	for _, unknown := range result.Unknowns {
		if strings.Contains(unknown, "gamma") {
			foundUnknown = true
		}
	}
	if !foundUnknown {
		t.Fatalf("gamma's unknown cost must be listed: %v", result.Unknowns)
	}
	// With cost unknown, gamma cannot win the cost criterion; its score must
	// be lower than a fully evidenced participant.
	if result.Scores["gamma"] >= result.Scores["beta"] {
		t.Fatalf("unknown cost must not behave as zero cost: scores=%v", result.Scores)
	}
}

func TestEvaluateChallengeNeverAutoMerges(t *testing.T) {
	participants := challengeParticipants()
	result, err := EvaluateChallenge("spec", participants, challengeCriteria())
	if err != nil {
		t.Fatal(err)
	}
	if result.AutoMergeAllowed {
		t.Fatal("challenge must never authorize automatic merge")
	}
	if !strings.Contains(result.MergePolicyNote, "no automatic merge") {
		t.Fatalf("merge policy must be explicit: %s", result.MergePolicyNote)
	}
	hasPolicy := false
	for _, reason := range result.BlockReasons {
		if strings.Contains(reason, "no automatic merge") {
			hasPolicy = true
		}
	}
	if !hasPolicy {
		t.Fatalf("block reasons must carry the merge policy: %v", result.BlockReasons)
	}
}

func TestEvaluateChallengeOrderIndependence(t *testing.T) {
	participants := challengeParticipants()
	reversed := []ChallengeParticipant{participants[2], participants[1], participants[0]}
	a, err := EvaluateChallenge("spec", participants, challengeCriteria())
	if err != nil {
		t.Fatal(err)
	}
	b, err := EvaluateChallenge("spec", reversed, challengeCriteria())
	if err != nil {
		t.Fatal(err)
	}
	for i := range a.Ranking {
		if a.Ranking[i] != b.Ranking[i] {
			t.Fatalf("input order must not change ranking: %v vs %v", a.Ranking, b.Ranking)
		}
	}
}
