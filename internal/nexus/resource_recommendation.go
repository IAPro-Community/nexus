package nexus

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/telemetry"
)

// TaskRequirements specifies what an intelligence or execution task demands (§Gate 5, Phase B).
type TaskRequirements struct {
	TaskKind             string   `json:"task_kind"` // "coding" | "planning" | "review" | "verify" | "refactor" | "security"
	Role                 string   `json:"role"`      // "implementer" | "reviewer" | "architect" | "tester"
	RequiredCapabilities []string `json:"required_capabilities,omitempty"`
	EstimatedTokens      int      `json:"estimated_tokens,omitempty"`
	CurrentProvider      string   `json:"current_provider,omitempty"`
	CurrentProfile       string   `json:"current_profile,omitempty"`
	PreferProvider       string   `json:"prefer_provider,omitempty"`
	ProjectPolicy        string   `json:"project_policy,omitempty"`
	AgentPreference      string   `json:"agent_preference,omitempty"`
}

// ResourceCandidate represents an evaluated provider account scored for a specific task.
type ResourceCandidate struct {
	Account         ProviderAccount    `json:"account"`
	Rank            int                `json:"rank"`
	TotalScore      float64            `json:"total_score"`
	Confidence      string             `json:"confidence"` // "LIVE" | "CACHED" | "ESTIMATED" | "UNKNOWN"
	ScoreBreakdown  map[string]float64 `json:"score_breakdown"`
	Pros            []string           `json:"pros"`
	Cons            []string           `json:"cons"`
	Eligible        bool               `json:"eligible"`
	RejectionReason string             `json:"rejection_reason,omitempty"`
}

// RecommendationResult holds the ranked candidates and top selection with explainable reasoning.
type RecommendationResult struct {
	Requirements         TaskRequirements    `json:"requirements"`
	Policy               SchedulerPolicy     `json:"policy"`
	Recommended          *ResourceCandidate  `json:"recommended,omitempty"`
	Candidates           []ResourceCandidate `json:"candidates"`
	Explanation          string              `json:"explanation"`
	ResetInformation     string              `json:"reset_information,omitempty"`
	RejectedAlternatives []string            `json:"rejected_alternatives,omitempty"`
	ContinuityImpact     string              `json:"continuity_impact,omitempty"`
	Fingerprint          string              `json:"fingerprint,omitempty"`
}

// RecommendResources evaluates available accounts against task requirements under the chosen policy.
func RecommendResources(accounts []ProviderAccount, req TaskRequirements, policy SchedulerPolicy) RecommendationResult {
	if policy == "" {
		if req.ProjectPolicy != "" {
			policy = SchedulerPolicy(strings.ToUpper(req.ProjectPolicy))
		} else {
			policy = PolicyBalanced
		}
	}

	result := RecommendationResult{
		Requirements: req,
		Policy:       policy,
		Candidates:   make([]ResourceCandidate, 0, len(accounts)),
	}

	for _, acc := range accounts {
		c := evaluateCandidate(acc, req, policy)
		result.Candidates = append(result.Candidates, c)
	}

	eligibleIdx := make([]int, 0, len(result.Candidates))
	allUnknown := true
	for i, c := range result.Candidates {
		if !c.Eligible {
			continue
		}
		eligibleIdx = append(eligibleIdx, i)
		if c.Confidence != "UNKNOWN" {
			allUnknown = false
		}
	}
	if allUnknown && len(eligibleIdx) > 0 {
		lastUsed := profileSelectedAt()
		sort.SliceStable(eligibleIdx, func(i, j int) bool {
			left := result.Candidates[eligibleIdx[i]].Account
			right := result.Candidates[eligibleIdx[j]].Account
			leftKey := left.Provider + "/" + left.Profile
			rightKey := right.Provider + "/" + right.Profile
			return lastUsed[leftKey].Before(lastUsed[rightKey])
		})
		best := result.Candidates[eligibleIdx[0]]
		result.Recommended = &best
		for i := range result.Candidates {
			result.Candidates[i].Rank = 0
		}
		for rank, idx := range eligibleIdx {
			result.Candidates[idx].Rank = rank + 1
		}
		result.Explanation = fmt.Sprintf("Recomendado %s (%s) por LRU entre perfis saudáveis com quota desconhecida.",
			best.Account.DisplayName, best.Account.Provider)
		return result
	}

	// Sort candidates: eligible first, then higher score, then lower rate limit risk
	sort.Slice(result.Candidates, func(i, j int) bool {
		if result.Candidates[i].Eligible != result.Candidates[j].Eligible {
			return result.Candidates[i].Eligible
		}
		if math.Abs(result.Candidates[i].TotalScore-result.Candidates[j].TotalScore) > 0.001 {
			return result.Candidates[i].TotalScore > result.Candidates[j].TotalScore
		}
		return result.Candidates[i].Account.DisplayName < result.Candidates[j].Account.DisplayName
	})

	// Assign ranks and identify recommended
	for i := range result.Candidates {
		result.Candidates[i].Rank = i + 1
		if result.Candidates[i].Eligible && result.Recommended == nil {
			cand := result.Candidates[i]
			result.Recommended = &cand
		}
	}

	for _, c := range result.Candidates {
		if !c.Eligible {
			reason := c.RejectionReason
			if reason == "" {
				reason = "inelegível"
			}
			result.RejectedAlternatives = append(result.RejectedAlternatives, fmt.Sprintf("%s (%s): %s", c.Account.DisplayName, c.Account.Provider, reason))
		}
	}

	if result.Recommended != nil {
		result.Explanation = fmt.Sprintf("Recomendado %s (%s) com score %.1f baseado na política %s.",
			result.Recommended.Account.DisplayName, result.Recommended.Account.Provider, result.Recommended.TotalScore, policy)

		if req.CurrentProvider != "" && result.Recommended.Account.Provider != req.CurrentProvider {
			result.ContinuityImpact = fmt.Sprintf("Troca de provedor (%s -> %s): novo contexto de runtime será iniciado", req.CurrentProvider, result.Recommended.Account.Provider)
		} else if req.CurrentProfile != "" && result.Recommended.Account.Profile != req.CurrentProfile {
			result.ContinuityImpact = fmt.Sprintf("Troca de perfil (%s -> %s): mesma sessão preservada", req.CurrentProfile, result.Recommended.Account.Profile)
		} else {
			result.ContinuityImpact = "Mesmo provedor e perfil: continuidade total preservada"
		}

		if result.Recommended.Account.QuotaView != nil {
			result.ResetInformation = result.Recommended.Account.QuotaView.Status
		}

		result.Fingerprint = fmt.Sprintf("rec-%s-%s-%s", result.Recommended.Account.Provider, result.Recommended.Account.Profile, policy)
	} else {
		result.Explanation = "Nenhum provedor elegível atende aos requisitos da tarefa (todos indisponíveis ou sem autenticação)."
	}

	return result
}

func profileSelectedAt() map[string]time.Time {
	result := make(map[string]time.Time)
	events, err := telemetry.ReadRecentEvents(0)
	if err != nil {
		return result
	}
	for _, ev := range events {
		if ev.Type != telemetry.EventProfileSelected {
			continue
		}
		key := ev.ProviderID + "/" + ev.ProfileID
		if old, ok := result[key]; !ok || ev.Timestamp.After(old) {
			result[key] = ev.Timestamp
		}
	}
	return result
}

func evaluateCandidate(acc ProviderAccount, req TaskRequirements, policy SchedulerPolicy) ResourceCandidate {
	c := ResourceCandidate{
		Account:        acc,
		ScoreBreakdown: make(map[string]float64),
		Pros:           make([]string, 0),
		Cons:           make([]string, 0),
		Eligible:       true,
		Confidence:     "UNKNOWN",
	}

	// Hard Gate 1: Must be authenticated
	if !acc.Authenticated {
		c.Eligible = false
		c.RejectionReason = "Conta não autenticada no sistema"
		c.Cons = append(c.Cons, "Não autenticada")
		return c
	}

	// Hard Gate 2: Must be available / not in active cooldown
	if !acc.Available {
		c.Eligible = false
		c.RejectionReason = "Provedor marcado como indisponível"
		c.Cons = append(c.Cons, "Indisponível")
		return c
	}

	if acc.RateLimited {
		c.Eligible = false
		c.RejectionReason = "Conta em rate limit / cooldown ativo"
		c.Cons = append(c.Cons, "Rate limit ativo")
		return c
	}

	if acc.CooldownUntil != nil && time.Now().Before(*acc.CooldownUntil) {
		c.Eligible = false
		c.RejectionReason = fmt.Sprintf("Em cooldown até %s", acc.CooldownUntil.Format(time.RFC3339))
		c.Cons = append(c.Cons, "Em cooldown")
		return c
	}

	// Hard Gate 3: every required capability must be explicitly SUPPORTED.
	// PARTIAL/UNKNOWN/NOT_TESTED are not enough for a hard requirement.
	for _, required := range req.RequiredCapabilities {
		key := normalizeCapabilityName(required)
		status := strings.ToUpper(strings.TrimSpace(acc.Capabilities[key]))
		if status != "SUPPORTED" {
			c.Eligible = false
			c.RejectionReason = fmt.Sprintf("Capacidade obrigatória %s não suportada (%s)", key, capabilityStatusLabel(status))
			c.Cons = append(c.Cons, c.RejectionReason)
			return c
		}
	}

	// Determine quota confidence from the actual usage status. A non-nil
	// QuotaView is not evidence that the data is LIVE.
	c.Confidence = quotaConfidence(acc)

	var score float64 = 50.0 // Baseline

	// 1. Quota & Capacity Scoring (0 to 30 pts)
	quotaScore := 0.0
	switch c.Confidence {
	case "LIVE":
		quotaScore = acc.QuotaRemaining * 30.0
		c.Pros = append(c.Pros, fmt.Sprintf("Quota em tempo real (%.0f%% restante)", acc.QuotaRemaining*100))
	case "CACHED":
		quotaScore = acc.QuotaRemaining * 20.0
		c.Pros = append(c.Pros, fmt.Sprintf("Quota em cache (%.0f%% restante)", acc.QuotaRemaining*100))
	case "ESTIMATED":
		quotaScore = acc.QuotaRemaining * 15.0
		c.Pros = append(c.Pros, fmt.Sprintf("Quota estimada (%.0f%% restante)", acc.QuotaRemaining*100))
	case "UNKNOWN":
		// Unknown quota is NOT treated as best (§Phase B: Unknown quota must not be treated as magically best)
		quotaScore = 10.0
		c.Cons = append(c.Cons, "Quota não aferida em tempo real")
	}
	c.ScoreBreakdown["quota"] = quotaScore
	score += quotaScore

	// 2. Health & Reliability (0 to 20 pts)
	healthScore := 10.0
	switch acc.Health {
	case "healthy":
		healthScore = 20.0
		c.Pros = append(c.Pros, "Saúde operacional excelente")
	case "degraded":
		healthScore = 5.0
		c.Cons = append(c.Cons, "Saúde degradada")
	case "unhealthy":
		healthScore = 0.0
		c.Cons = append(c.Cons, "Falhas recentes registradas")
	}
	c.ScoreBreakdown["health"] = healthScore
	score += healthScore

	// 3. Affinity & Continuity (0 to 20 pts)
	affinityScore := 0.0
	if req.CurrentProvider != "" && acc.Provider == req.CurrentProvider {
		affinityScore += 15.0
		c.Pros = append(c.Pros, "Afinidade com sessão/runtime atual (sem custo de troca de contexto)")
		if req.CurrentProfile != "" && acc.Profile == req.CurrentProfile {
			affinityScore += 5.0
			c.Pros = append(c.Pros, "Perfil idêntico ao runtime ativo")
		}
	} else if req.CurrentProvider != "" {
		c.Cons = append(c.Cons, "Exige troca de provedor (custo de novo contexto)")
	}
	c.ScoreBreakdown["affinity"] = affinityScore
	score += affinityScore

	// 4. Role & Capability Fit (0 to 20 pts)
	// Instead of hard-coding provider names, we reward accounts that advertise
	// the structural capabilities most valuable for the requested role.
	// Any driver (current or future) that declares those capabilities receives
	// the same bonus automatically.
	roleScore := 10.0
	switch req.Role {
	case "reviewer", "tester":
		// Reviewers benefit from structured event streams (richer feedback) and
		// headless approvals mode (safe read-only review). Both are already
		// hard-gated as required, so we give a bonus for extra quality signals.
		if capSupported(acc, "structured_events") || capSupported(acc, "approvals") {
			roleScore = 20.0
			c.Pros = append(c.Pros, "Capacidade avançada de revisão e feedback estruturado")
		}
	case "implementer":
		// Implementers benefit from headless execution and, optionally, Fork
		// (true parallelism within a session). Both headless and submit_prompt are
		// already required; Fork bumps the score for parallel-capable accounts.
		if capSupported(acc, "fork") || capSupported(acc, "sessions") {
			roleScore = 20.0
			c.Pros = append(c.Pros, "Alta performance para implementação paralela de código")
		} else if capSupported(acc, "headless") && capSupported(acc, "submit_prompt") {
			roleScore = 18.0
			c.Pros = append(c.Pros, "Suporte completo a execução autônoma")
		}
	case "architect", "planner":
		// Architects and planners benefit from long sessions with resume support
		// (continuous reasoning across multiple turns).
		if capSupported(acc, "sessions") && capSupported(acc, "resume") {
			roleScore = 20.0
			c.Pros = append(c.Pros, "Capacidade avançada de raciocínio e decomposição em sessão longa")
		} else if capSupported(acc, "sessions") || capSupported(acc, "resume") {
			roleScore = 16.0
			c.Pros = append(c.Pros, "Suporte a sessões estendidas de planejamento")
		}
	}
	c.ScoreBreakdown["role_fit"] = roleScore
	score += roleScore

	// 5. Policy Multipliers
	switch policy {
	case PolicyPreserveQuota:
		if acc.QuotaRemaining > 0.7 {
			score += 25.0
			c.Pros = append(c.Pros, "Preserva quotas escassas utilizando conta com alta disponibilidade")
		} else if acc.QuotaRemaining < 0.3 && c.Confidence == "LIVE" {
			score -= 20.0
			c.Cons = append(c.Cons, "Penalizado pela política Preserve Quota (quota < 30%)")
		}
	case PolicyPreferProvider:
		prefer := req.PreferProvider
		if prefer == "" {
			prefer = req.AgentPreference
		}
		if prefer != "" && (acc.Provider == prefer || acc.ID == prefer) {
			score += 35.0
			c.Pros = append(c.Pros, fmt.Sprintf("Provedor preferencial explícito (%s)", prefer))
		}
	case PolicyManual:
		if req.PreferProvider != "" && acc.Provider == req.PreferProvider {
			score += 50.0
		} else {
			score -= 50.0
		}
	case PolicyBalanced:
		// Default balanced weights
	}

	c.TotalScore = math.Round(score*10) / 10
	return c
}

func quotaConfidence(acc ProviderAccount) string {
	if acc.QuotaView != nil {
		switch strings.ToUpper(strings.TrimSpace(acc.QuotaView.Status)) {
		case "LIVE":
			return "LIVE"
		case "CACHED":
			return "CACHED"
		case "ESTIMATED":
			return "ESTIMATED"
		default:
			return "UNKNOWN"
		}
	}
	if acc.QuotaRemaining > 0 {
		return "CACHED"
	}
	return "UNKNOWN"
}

func normalizeCapabilityName(value string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "-", "_")
}

func capabilityStatusLabel(status string) string {
	if status == "" {
		return "UNKNOWN"
	}
	return status
}

// capSupported returns true when the account's capability map advertises
// the given capability (snake_case or hyphenated) as "supported".
func capSupported(acc ProviderAccount, capability string) bool {
	key := normalizeCapabilityName(capability)
	val := strings.ToUpper(strings.TrimSpace(acc.Capabilities[key]))
	return val == "SUPPORTED"
}
