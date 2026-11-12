package nexus

import (
	"fmt"
	"sort"
	"strings"
)

// Parallel challenge (capability parallel-challenge).
//
// Distributes one specification to multiple isolated agents (each in its own
// worktree) and compares their evidence with explicit criteria. The challenge
// NEVER merges: the result carries an explicit auto-merge denial with the
// reason, and the human decides what happens next. Unknown evidence —
// especially UNKNOWN cost — is never treated as zero: a participant without
// cost evidence cannot win a cost criterion and is listed as unknown.

// ChallengeParticipant is one isolated agent entry with optional evidence.
type ChallengeParticipant struct {
	Label        string
	AgentID      string
	WorktreePath string
	// Evidence fields follow agent-compare semantics.
	TestsPassed int
	TestsFailed int
	CostKnown   bool
	CostTotal   float64
}

// ChallengeResult is the deterministic, merge-free comparison outcome.
type ChallengeResult struct {
	Specification string
	Participants  int
	Ranking       []string // labels, best first; ties keep alphabetical order
	Scores        map[string]int
	Unknowns      []string
	// AutoMergeAllowed is always false with an explicit reason: the
	// challenge informs, the human merges.
	AutoMergeAllowed bool
	MergePolicyNote  string
	BlockReasons     []string
}

func toCompareSide(participant ChallengeParticipant) AgentCompareSide {
	return AgentCompareSide{
		Label:        participant.Label,
		WorktreePath: participant.WorktreePath,
		ChangedFiles: 0,
		TestsPassed:  participant.TestsPassed,
		TestsFailed:  participant.TestsFailed,
		CostKnown:    participant.CostKnown,
		CostTotal:    participant.CostTotal,
	}
}

// EvaluateChallenge compares every participant pair against the explicit
// criteria (round-robin, deterministic order) and ranks the results.
func EvaluateChallenge(specification string, participants []ChallengeParticipant, criteria []AgentCompareCriterion) (*ChallengeResult, error) {
	if strings.TrimSpace(specification) == "" {
		return nil, fmt.Errorf("parallel challenge: specification is required")
	}
	if len(participants) < 2 {
		return nil, fmt.Errorf("parallel challenge: at least two isolated agents are required")
	}
	seen := make(map[string]string)
	labels := make([]string, 0, len(participants))
	for _, participant := range participants {
		if strings.TrimSpace(participant.WorktreePath) == "" {
			return nil, fmt.Errorf("parallel challenge: participant %q requires an isolated worktree", participant.Label)
		}
		if other, clash := seen[participant.WorktreePath]; clash {
			return nil, fmt.Errorf(
				"parallel challenge: participants %q and %q share worktree %s; isolation is required",
				other, participant.Label, participant.WorktreePath,
			)
		}
		seen[participant.WorktreePath] = participant.Label
		labels = append(labels, participant.Label)
	}
	if len(criteria) == 0 {
		return nil, fmt.Errorf("parallel challenge: at least one explicit criterion is required")
	}

	result := &ChallengeResult{
		Specification:    strings.TrimSpace(specification),
		Participants:     len(participants),
		Scores:           make(map[string]int, len(participants)),
		AutoMergeAllowed: false,
		MergePolicyNote:  "no automatic merge; human review and explicit approval required",
	}
	sort.Strings(labels)

	byLabel := make(map[string]ChallengeParticipant, len(participants))
	for _, participant := range participants {
		byLabel[participant.Label] = participant
		result.Scores[participant.Label] = 0
	}

	// Round-robin in label order: deterministic regardless of input order.
	for i := 0; i < len(labels); i++ {
		for j := i + 1; j < len(labels); j++ {
			left, right := byLabel[labels[i]], byLabel[labels[j]]
			comparison, err := CompareAgentImplementations(toCompareSide(left), toCompareSide(right), criteria)
			if err != nil {
				return nil, err
			}
			for _, outcome := range comparison.Outcomes {
				if !outcome.Decided {
					if outcome.Note != "" {
						result.Unknowns = append(result.Unknowns, comparison.SideA+" vs "+comparison.SideB+": "+outcome.Note)
					}
					continue
				}
				switch outcome.Winner {
				case "a":
					result.Scores[left.Label]++
				case "b":
					result.Scores[right.Label]++
				}
			}
		}
	}

	sort.SliceStable(labels, func(x, y int) bool {
		if result.Scores[labels[x]] != result.Scores[labels[y]] {
			return result.Scores[labels[x]] > result.Scores[labels[y]]
		}
		return labels[x] < labels[y]
	})
	result.Ranking = labels

	if len(result.Unknowns) > 0 {
		sort.Strings(result.Unknowns)
		result.BlockReasons = append(result.BlockReasons,
			"unknown evidence present; cost or tests may be UNKNOWN and are not treated as zero")
	}
	result.BlockReasons = append(result.BlockReasons, result.MergePolicyNote)
	return result, nil
}
