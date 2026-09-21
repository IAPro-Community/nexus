package nexus

import (
	"fmt"
	"sort"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// AgentMatchRequirements describes the durable specialization needed by a
// task. It is deliberately provider-independent; provider/profile selection
// remains owned by ResourceScheduler.
type AgentMatchRequirements struct {
	PreferredRoles        []string
	AcceptableRoles       []string
	Domains               []string
	RequiredCapabilities  []string
	PreferredCapabilities []string
	DesiredStrengths      []string
	Constraints           []string
}

// AgentMatchCandidate is the only input shape accepted by the canonical
// matcher. The caller is responsible for scoping candidates to one project.
type AgentMatchCandidate struct {
	Agent store.Agent
	Spec  intelligence.AgentSpec
}

type AgentMatchEvaluation struct {
	Agent           store.Agent            `json:"agent"`
	Spec            intelligence.AgentSpec `json:"spec"`
	Eligible        bool                   `json:"eligible"`
	Score           float64                `json:"score"`
	Confidence      string                 `json:"confidence"`
	ScoreBreakdown  map[string]float64     `json:"score_breakdown"`
	Pros            []string               `json:"pros"`
	Cons            []string               `json:"cons"`
	RejectionReason string                 `json:"rejection_reason,omitempty"`
}

type AgentMatchResult struct {
	Requirements AgentMatchRequirements `json:"requirements"`
	Recommended  *AgentMatchEvaluation  `json:"recommended,omitempty"`
	Candidates   []AgentMatchEvaluation `json:"candidates"`
	Explanation  string                 `json:"explanation"`
}

// MatchAgents is the canonical persistent-Agent matcher. It applies hard
// lifecycle/capability gates first, then a deterministic specialization score.
func MatchAgents(candidates []AgentMatchCandidate, req AgentMatchRequirements) AgentMatchResult {
	result := AgentMatchResult{Requirements: req, Candidates: make([]AgentMatchEvaluation, 0, len(candidates))}
	for _, candidate := range candidates {
		result.Candidates = append(result.Candidates, evaluateAgentMatch(candidate, req))
	}
	sort.SliceStable(result.Candidates, func(i, j int) bool {
		left, right := result.Candidates[i], result.Candidates[j]
		if left.Eligible != right.Eligible {
			return left.Eligible
		}
		if left.Score != right.Score {
			return left.Score > right.Score
		}
		return left.Agent.ID < right.Agent.ID
	})
	for i := range result.Candidates {
		if result.Candidates[i].Eligible && result.Recommended == nil {
			selected := result.Candidates[i]
			result.Recommended = &selected
		}
	}
	if result.Recommended == nil {
		result.Explanation = "Nenhum Agent elegível atende aos requisitos da tarefa."
		return result
	}
	result.Explanation = fmt.Sprintf("Agent recomendado: %s (score %.1f, confiança %s).", result.Recommended.Agent.Name, result.Recommended.Score, result.Recommended.Confidence)
	return result
}

func evaluateAgentMatch(candidate AgentMatchCandidate, req AgentMatchRequirements) AgentMatchEvaluation {
	spec := candidate.Spec
	if strings.TrimSpace(spec.Role) == "" {
		spec.Role = candidate.Agent.Role
	}
	evaluation := AgentMatchEvaluation{
		Agent:          candidate.Agent,
		Spec:           spec,
		Eligible:       true,
		Confidence:     "ESTIMATED",
		ScoreBreakdown: map[string]float64{},
		Pros:           []string{},
		Cons:           []string{},
	}

	switch candidate.Agent.Status {
	case store.AgentStopped, store.AgentRecoverable, "":
	default:
		return rejectAgentMatch(evaluation, "Agent está ocupado, desabilitado ou não recuperável")
	}

	role := normalizeAgentTaxonomy(spec.Role)
	if len(req.AcceptableRoles) > 0 && !containsNormalized(req.AcceptableRoles, role) {
		return rejectAgentMatch(evaluation, fmt.Sprintf("role %s não é aceitável para a tarefa", role))
	}
	score := 0.0
	if containsNormalized(req.PreferredRoles, role) {
		score += 50
		evaluation.ScoreBreakdown["preferred_role"] = 50
		evaluation.Pros = append(evaluation.Pros, "Role preferida")
	} else if containsNormalized(req.AcceptableRoles, role) {
		score += 25
		evaluation.ScoreBreakdown["acceptable_role"] = 25
		evaluation.Pros = append(evaluation.Pros, "Role aceitável")
	}

	for _, required := range req.RequiredCapabilities {
		if !containsNormalized(spec.Capabilities, required) {
			return rejectAgentMatch(evaluation, fmt.Sprintf("capability obrigatória %s ausente", required))
		}
		score += 15
	}
	evaluation.ScoreBreakdown["required_capabilities"] = float64(len(req.RequiredCapabilities)) * 15

	score += overlapScore(spec.Domains, req.Domains, 10, &evaluation, "domains")
	score += overlapScore(spec.Capabilities, req.PreferredCapabilities, 5, &evaluation, "preferred_capabilities")
	score += overlapScore(spec.Strengths, req.DesiredStrengths, 5, &evaluation, "strengths")
	evaluation.Score = score
	if len(req.PreferredRoles) == 0 && len(req.AcceptableRoles) == 0 && len(req.RequiredCapabilities) == 0 && len(req.Domains) == 0 {
		evaluation.Confidence = "UNKNOWN"
	}
	return evaluation
}

func rejectAgentMatch(evaluation AgentMatchEvaluation, reason string) AgentMatchEvaluation {
	evaluation.Eligible = false
	evaluation.RejectionReason = reason
	evaluation.Cons = append(evaluation.Cons, reason)
	return evaluation
}

func overlapScore(values, wanted []string, points float64, evaluation *AgentMatchEvaluation, key string) float64 {
	matched := 0
	for _, item := range wanted {
		if containsNormalized(values, item) {
			matched++
		}
	}
	if matched == 0 {
		return 0
	}
	score := float64(matched) * points
	evaluation.ScoreBreakdown[key] = score
	evaluation.Pros = append(evaluation.Pros, fmt.Sprintf("Correspondência em %s (%d)", key, matched))
	return score
}

func containsNormalized(values []string, wanted string) bool {
	wanted = normalizeAgentTaxonomy(wanted)
	for _, value := range values {
		if normalizeAgentTaxonomy(value) == wanted {
			return true
		}
	}
	return false
}

func normalizeAgentTaxonomy(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	switch value {
	case "frontend", "front-end", "frontend-dev", "frontend-developer":
		return "frontend-engineer"
	case "backend", "back-end", "backend-dev", "backend-developer":
		return "backend-engineer"
	case "fullstack", "full-stack", "fullstack-developer":
		return "fullstack-engineer"
	case "qa-engineer", "test-engineer", "tester":
		return "qa"
	case "code-reviewer", "review":
		return "reviewer"
	case "security", "security-auditor", "appsec", "security-review":
		return "security-reviewer"
	case "software-architect", "solution-architect":
		return "architect"
	case "debugger", "root-cause-investigator":
		return "investigator"
	case "performance", "performance-engineer":
		return "performance-investigator"
	case "senior", "senior-engineer":
		return "senior-developer"
	case "devops-release":
		return "devops"
	}
	return value
}
