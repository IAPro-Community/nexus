package nexus

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
)

// TaskRequirements specifies what an intelligence or execution task demands (§Gate 5, Phase B).
type TaskRequirements struct {
	TaskKind              string                          `json:"task_kind"` // "coding" | "planning" | "review" | "verify" | "refactor" | "security"
	Role                  string                          `json:"role"`      // "implementer" | "reviewer" | "architect" | "tester"
	PreferredRoles        []string                        `json:"preferred_roles,omitempty"`
	AcceptableRoles       []string                        `json:"acceptable_roles,omitempty"`
	Domains               []string                        `json:"domains,omitempty"`
	RequiredCapabilities  []string                        `json:"required_capabilities,omitempty"`
	PreferredCapabilities []string                        `json:"preferred_capabilities,omitempty"`
	DesiredStrengths      []string                        `json:"desired_strengths,omitempty"`
	Constraints           []string                        `json:"constraints,omitempty"`
	EstimatedComplexity   string                          `json:"estimated_complexity,omitempty"`
	EstimatedTokens       int                             `json:"estimated_tokens,omitempty"`
	RequiresDecomposition bool                            `json:"requires_decomposition,omitempty"`
	Confidence            string                          `json:"confidence,omitempty"`
	Source                string                          `json:"source,omitempty"`
	CurrentProvider       string                          `json:"current_provider,omitempty"`
	CurrentProfile        string                          `json:"current_profile,omitempty"`
	PreferProvider        string                          `json:"prefer_provider,omitempty"`
	ProjectPolicy         string                          `json:"project_policy,omitempty"`
	AgentPreference       string                          `json:"agent_preference,omitempty"`
	ProviderPriorities    map[string]int                  `json:"provider_priorities,omitempty"`
	RuntimeAffinity       *RuntimeAffinityPolicy          `json:"runtime_affinity,omitempty"`
	ModelCandidates       []ModelCandidate                `json:"model_candidates,omitempty"`
	EscalationLevel       int                             `json:"escalation_level,omitempty"`
	Guidance              *intelligence.ExecutionGuidance `json:"guidance,omitempty"`
}

// ResourceCandidate represents an evaluated provider account scored for a specific task.
type ResourceCandidate struct {
	Account          ProviderAccount    `json:"account"`
	Rank             int                `json:"rank"`
	TotalScore       float64            `json:"total_score"`
	Confidence       string             `json:"confidence"` // "LIVE" | "CACHED" | "ESTIMATED" | "UNSUPPORTED" | "UNKNOWN"
	ScoreBreakdown   map[string]float64 `json:"score_breakdown"`
	Pros             []string           `json:"pros"`
	Cons             []string           `json:"cons"`
	Eligible         bool               `json:"eligible"`
	RejectionReason  string             `json:"rejection_reason,omitempty"`
	ProviderPriority int                `json:"provider_priority,omitempty"`
}

// RecommendationResult holds the ranked candidates and top selection with explainable reasoning.
type RecommendationResult struct {
	Requirements TaskRequirements    `json:"requirements"`
	Policy       SchedulerPolicy     `json:"policy"`
	Recommended  *ResourceCandidate  `json:"recommended,omitempty"`
	Candidates   []ResourceCandidate `json:"candidates"`
	Explanation  string              `json:"explanation"`
}

// providerResourceRequirements keeps task/Agent specializations out of the
// provider control-capability hard gate. TaskRequirements.RequiredCapabilities
// historically carries both kinds of requirements (for example "security"
// and "code_review" alongside "headless" and "submit_prompt"). Only the
// capabilities actually advertised by a provider driver can constrain runtime
// selection; task specializations remain available to the Agent matcher.
func providerResourceRequirements(req TaskRequirements) TaskRequirements {
	filtered := make([]string, 0, len(req.RequiredCapabilities))
	for _, capability := range req.RequiredCapabilities {
		switch normalizeCapabilityName(capability) {
		case "process", "terminal", "attach", "structured_events", "sessions", "resume", "fork",
			"submit_prompt", "cancel_turn", "approvals", "native_ui_attach", "headless", "slash_control", "read_only_review":
			filtered = append(filtered, capability)
		}
	}
	req.RequiredCapabilities = filtered
	return req
}

// RecommendResources evaluates available accounts against task requirements under the chosen policy.
func RecommendResources(accounts []ProviderAccount, req TaskRequirements, policy SchedulerPolicy) RecommendationResult {
	if len(req.ProviderPriorities) == 0 {
		cfg, _ := config.LoadConfig()
		req.ProviderPriorities = cfg.ProviderPriorities
	}
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

	// Sort candidates: eligible first, then higher score, then lower rate limit risk
	sort.Slice(result.Candidates, func(i, j int) bool {
		if result.Candidates[i].Eligible != result.Candidates[j].Eligible {
			return result.Candidates[i].Eligible
		}
		leftPriority := result.Candidates[i].ProviderPriority
		rightPriority := result.Candidates[j].ProviderPriority
		if leftPriority > 0 && rightPriority > 0 && leftPriority != rightPriority {
			return leftPriority < rightPriority
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

	if result.Recommended != nil {
		result.Explanation = fmt.Sprintf("Recomendado %s (%s) com score %.1f baseado na política %s.",
			result.Recommended.Account.DisplayName, result.Recommended.Account.Provider, result.Recommended.TotalScore, policy)
	} else {
		result.Explanation = "Nenhum provedor elegível atende aos requisitos da tarefa (todos indisponíveis ou sem autenticação)."
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
	if req.ProviderPriorities != nil {
		c.ProviderPriority = req.ProviderPriorities[acc.Provider]
	}

	elig := quota.EvaluateEligibility(accountEligibilityInput(acc, false))
	c.Confidence = elig.Confidence
	if !elig.Eligible {
		c.Eligible = false
		c.RejectionReason = elig.Message
		if elig.Message != "" {
			c.Cons = append(c.Cons, elig.Message)
		}
		return c
	}

	// Hard Gate: every required capability must be explicitly SUPPORTED.
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

	var score = 50.0 // Baseline
	if c.ProviderPriority > 0 {
		c.ScoreBreakdown["provider_priority"] = float64(c.ProviderPriority)
		c.Pros = append(c.Pros, fmt.Sprintf("Prioridade de fallback do provider: %d", c.ProviderPriority))
	}

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
	case "UNSUPPORTED":
		// Provider is operationally usable but exposes no trustworthy capacity
		// metric. Keep it below measured resources without inventing a percent.
		quotaScore = 5.0
		c.Cons = append(c.Cons, "Quota não exposta pelo provedor; capacidade opaca")
	case "UNKNOWN":
		// UNKNOWN is normally rejected by EvaluateEligibility; keep this branch
		// defensive for legacy/in-memory callers and never score it as capacity.
		quotaScore = 0.0
		c.Cons = append(c.Cons, "Quota sem evidência suficiente")
	}
	c.ScoreBreakdown["quota"] = quotaScore
	score += quotaScore

	// 2. Health & Reliability (0 to 20 pts)
	healthScore := 10.0
	switch acc.Health {
	case "healthy":
		healthScore = 20.0
		c.Pros = append(c.Pros, "Saúde de operação excelente")
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
		case "UNSUPPORTED":
			return "UNSUPPORTED"
		case "OK":
			// Compatibility with pre-UsageStatus providers. These snapshots
			// carried measured aggregate quota, so preserve them as bounded
			// cache evidence instead of classifying them as UNKNOWN.
			return "CACHED"
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
