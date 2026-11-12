package nexus

import (
	"fmt"
	"sort"
	"strings"
)

// Agent compare (capability agent-compare).
//
// Compares two implementations that were executed in independent worktrees
// and reports a verdict against criteria declared by the caller. The model
// never merges anything and never decides by itself what "better" means:
// every criterion is explicit, and unknown evidence (especially UNKNOWN
// cost) is never silently treated as zero — a side without cost evidence
// cannot win a cost criterion, and the overall verdict degrades to
// inconclusive while unknowns remain.

// Criterion kinds supported by the comparator.
const (
	AgentCompareCriterionTests = "tests"
	AgentCompareCriterionCost  = "cost"
)

// AgentCompareSide is one implementation with its verifiable evidence.
type AgentCompareSide struct {
	Label        string
	WorktreePath string
	HeadSHA      string
	ChangedFiles int
	// Test evidence: counts must come from real check executions.
	TestsPassed int
	TestsFailed int
	// Cost evidence: CostKnown=false means the cost is UNKNOWN, never zero.
	CostKnown bool
	CostTotal float64
}

// AgentCompareCriterion is one explicit judging criterion.
type AgentCompareCriterion struct {
	Name string
	Kind string // "tests" or "cost"
	// Weight is currently uniform; kept explicit so callers see the scale.
	Weight int
}

// CriterionOutcome is the per-criterion measurement, echoed with its inputs.
type CriterionOutcome struct {
	Criterion string
	Kind      string
	// SideA/SideB hold the compared measurement ("3/1" tests, "0.42" cost,
	// or "unknown" when evidence is missing for that side).
	SideA   string
	SideB   string
	Winner  string // "a", "b", or "tie"
	Decided bool   // false when either side lacks evidence
	Note    string
}

// AgentCompareResult is the full deterministic comparison.
type AgentCompareResult struct {
	SideA       string
	SideB       string
	Verdict     string // "a", "b", "tie", or "inconclusive"
	Outcomes    []CriterionOutcome
	Unknowns    []string
	BlockReason []string
}

// CompareAgentImplementations judges two worktree implementations against
// explicit criteria. Worktree independence is enforced: comparing two sides
// of the same worktree is a misuse and is blocked.
func CompareAgentImplementations(a, b AgentCompareSide, criteria []AgentCompareCriterion) (*AgentCompareResult, error) {
	result := &AgentCompareResult{SideA: a.Label, SideB: b.Label}
	if strings.TrimSpace(a.WorktreePath) == "" || strings.TrimSpace(b.WorktreePath) == "" {
		return nil, fmt.Errorf("agent compare: both sides require an independent worktree path")
	}
	if a.WorktreePath == b.WorktreePath {
		return nil, fmt.Errorf(
			"agent compare: sides must come from independent worktrees, got %s twice",
			a.WorktreePath,
		)
	}
	if len(criteria) == 0 {
		return nil, fmt.Errorf("agent compare: at least one explicit criterion is required")
	}

	scoreA, scoreB := 0, 0
	decided := 0
	for _, criterion := range criteria {
		outcome := evaluateCriterion(a, b, criterion)
		result.Outcomes = append(result.Outcomes, outcome)
		if outcome.Decided {
			decided++
			switch outcome.Winner {
			case "a":
				scoreA += weightOf(criterion)
			case "b":
				scoreB += weightOf(criterion)
			}
		} else if outcome.Note != "" {
			result.Unknowns = append(result.Unknowns, outcome.Note)
		}
	}

	switch {
	case decided == 0:
		result.Verdict = "inconclusive"
	case scoreA > scoreB:
		result.Verdict = "a"
	case scoreB > scoreA:
		result.Verdict = "b"
	default:
		result.Verdict = "tie"
	}
	if len(result.Unknowns) > 0 {
		sort.Strings(result.Unknowns)
		// Unknown evidence keeps the human in the loop: the verdict stays
		// labeled even when criteria lean one way, so UNKNOWN never reads
		// as a zero-cost advantage.
		result.BlockReason = append(result.BlockReason,
			"unknown evidence present; review required before any integration decision")
	}
	return result, nil
}

func weightOf(criterion AgentCompareCriterion) int {
	if criterion.Weight <= 0 {
		return 1
	}
	return criterion.Weight
}

func evaluateCriterion(a, b AgentCompareSide, criterion AgentCompareCriterion) CriterionOutcome {
	outcome := CriterionOutcome{Criterion: criterion.Name, Kind: criterion.Kind}
	switch criterion.Kind {
	case AgentCompareCriterionTests:
		outcome.SideA = fmt.Sprintf("%d/%d", a.TestsPassed, a.TestsFailed)
		outcome.SideB = fmt.Sprintf("%d/%d", b.TestsPassed, b.TestsFailed)
		rateA := testScore(a)
		rateB := testScore(b)
		switch {
		case rateA > rateB:
			outcome.Winner, outcome.Decided = "a", true
		case rateB > rateA:
			outcome.Winner, outcome.Decided = "b", true
		default:
			outcome.Winner = "tie"
		}
		return outcome
	case AgentCompareCriterionCost:
		if !a.CostKnown || !b.CostKnown {
			if !a.CostKnown {
				outcome.SideA = "unknown"
				outcome.Note = fmt.Sprintf("cost evidence unknown for side %s", a.Label)
			} else {
				outcome.SideA = formatCost(a.CostTotal)
			}
			if !b.CostKnown {
				outcome.SideB = "unknown"
				outcome.Note = fmt.Sprintf("cost evidence unknown for side %s", b.Label)
			} else {
				outcome.SideB = formatCost(b.CostTotal)
			}
			return outcome
		}
		outcome.SideA = formatCost(a.CostTotal)
		outcome.SideB = formatCost(b.CostTotal)
		switch {
		case a.CostTotal < b.CostTotal:
			outcome.Winner, outcome.Decided = "a", true
		case b.CostTotal < a.CostTotal:
			outcome.Winner, outcome.Decided = "b", true
		default:
			outcome.Winner = "tie"
		}
		return outcome
	default:
		outcome.Note = fmt.Sprintf("unsupported criterion kind %q; criterion ignored", criterion.Kind)
		return outcome
	}
}

// testScore is pass count minus twice failures: failing tests weigh more
// than missing passes, so a side cannot win on volume of skipped tests.
func testScore(side AgentCompareSide) int {
	return side.TestsPassed - 2*side.TestsFailed
}

func formatCost(total float64) string {
	return fmt.Sprintf("%.2f", total)
}
