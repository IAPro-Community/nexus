package nexus

import (
	"strings"
	"testing"
)

func baseCompareSides() (AgentCompareSide, AgentCompareSide) {
	return AgentCompareSide{
			Label:        "variant-a",
			WorktreePath: "/wt/a",
			HeadSHA:      "aaa1111aaa",
			ChangedFiles: 3,
			TestsPassed:  10,
			TestsFailed:  0,
			CostKnown:    true,
			CostTotal:    1.50,
		},
		AgentCompareSide{
			Label:        "variant-b",
			WorktreePath: "/wt/b",
			HeadSHA:      "bbb2222bbb",
			ChangedFiles: 5,
			TestsPassed:  10,
			TestsFailed:  1,
			CostKnown:    true,
			CostTotal:    1.00,
		}
}

func TestCompareAgentImplementationsRejectsSharedWorktree(t *testing.T) {
	a, b := baseCompareSides()
	b.WorktreePath = a.WorktreePath
	criteria := []AgentCompareCriterion{{Name: "tests", Kind: AgentCompareCriterionTests}}
	if _, err := CompareAgentImplementations(a, b, criteria); err == nil ||
		!strings.Contains(err.Error(), "independent worktrees") {
		t.Fatalf("shared worktree must be blocked: %v", err)
	}
	empty := b
	empty.WorktreePath = ""
	if _, err := CompareAgentImplementations(a, empty, criteria); err == nil ||
		!strings.Contains(err.Error(), "independent worktree path") {
		t.Fatalf("missing worktree must be blocked: %v", err)
	}
}

func TestCompareAgentImplementationsRequiresExplicitCriteria(t *testing.T) {
	a, b := baseCompareSides()
	if _, err := CompareAgentImplementations(a, b, nil); err == nil ||
		!strings.Contains(err.Error(), "explicit criterion") {
		t.Fatalf("empty criteria must be blocked: %v", err)
	}
}

func TestCompareAgentImplementationsTestsCriterionWeightsFailures(t *testing.T) {
	a, b := baseCompareSides()
	result, err := CompareAgentImplementations(a, b, []AgentCompareCriterion{
		{Name: "tests", Kind: AgentCompareCriterionTests},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != "a" {
		t.Fatalf("10/0 must beat 10/1: %+v", result)
	}
	if result.Outcomes[0].Winner != "a" || !result.Outcomes[0].Decided {
		t.Fatalf("tests outcome wrong: %+v", result.Outcomes[0])
	}
	if result.Outcomes[0].SideA != "10/0" || result.Outcomes[0].SideB != "10/1" {
		t.Fatalf("measurements must be echoed: %+v", result.Outcomes[0])
	}
}

func TestCompareAgentImplementationsUnknownCostIsNeverZero(t *testing.T) {
	a, b := baseCompareSides()
	b.CostKnown = false
	result, err := CompareAgentImplementations(a, b, []AgentCompareCriterion{
		{Name: "cost", Kind: AgentCompareCriterionCost},
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := result.Outcomes[0]
	if outcome.Decided || outcome.Winner != "" {
		t.Fatalf("unknown cost must not decide: %+v", outcome)
	}
	if outcome.SideB != "unknown" {
		t.Fatalf("unknown must stay unknown, got %q", outcome.SideB)
	}
	if len(result.Unknowns) != 1 || !strings.Contains(result.Unknowns[0], "variant-b") {
		t.Fatalf("unknown must be listed: %v", result.Unknowns)
	}
	if result.Verdict != "inconclusive" || len(result.BlockReason) != 1 {
		t.Fatalf("verdict must stay inconclusive with unknowns: %+v", result)
	}
}

func TestCompareAgentImplementationsCostDecidesOnlyWithEvidence(t *testing.T) {
	a, b := baseCompareSides()
	result, err := CompareAgentImplementations(a, b, []AgentCompareCriterion{
		{Name: "cost", Kind: AgentCompareCriterionCost},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != "b" {
		t.Fatalf("cheaper side with full evidence wins: %+v", result)
	}
}

func TestCompareAgentImplementationsMixedCriteriaAndTie(t *testing.T) {
	a, b := baseCompareSides()
	// tests favor a, cost favors b: tie overall.
	result, err := CompareAgentImplementations(a, b, []AgentCompareCriterion{
		{Name: "tests", Kind: AgentCompareCriterionTests},
		{Name: "cost", Kind: AgentCompareCriterionCost},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != "tie" || len(result.Outcomes) != 2 {
		t.Fatalf("balanced criteria must tie: %+v", result)
	}
}

func TestCompareAgentImplementationsUnsupportedKindIsReported(t *testing.T) {
	a, b := baseCompareSides()
	result, err := CompareAgentImplementations(a, b, []AgentCompareCriterion{
		{Name: "style", Kind: "vibes"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != "inconclusive" || len(result.Outcomes) != 1 {
		t.Fatalf("unsupported criterion must not decide: %+v", result)
	}
	if !strings.Contains(result.Outcomes[0].Note, "unsupported criterion kind") {
		t.Fatalf("unsupported kind must be explicit: %+v", result.Outcomes[0])
	}
}
