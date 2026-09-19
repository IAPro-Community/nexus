package nexus

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/classifier"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	nexusskills "github.com/IAPro-Community/nexus/internal/nexus/skills"
)

type AffinityMode string

const (
	AffinityAuto   AffinityMode = "AUTO"
	AffinityPrefer AffinityMode = "PREFER"
	AffinityPin    AffinityMode = "PIN"
)

type AffinityPreference struct {
	Mode  AffinityMode `json:"mode"`
	Value string       `json:"value,omitempty"`
}

type RuntimeAffinityPolicy struct {
	Engine   AffinityPreference `json:"engine"`
	Provider AffinityPreference `json:"provider"`
	Profile  AffinityPreference `json:"profile"`
	Model    AffinityPreference `json:"model"`
}

type ModelCandidate struct {
	Provider       string   `json:"provider"`
	Profile        string   `json:"profile"`
	Model          string   `json:"model"`
	Capabilities   []string `json:"capabilities,omitempty"`
	CostRank       int      `json:"cost_rank"`
	ReasoningRank  int      `json:"reasoning_rank"`
	Healthy        bool     `json:"healthy"`
	Authenticated  bool     `json:"authenticated"`
	QuotaAvailable bool     `json:"quota_available"`
}

type RuntimeRoutingDecision struct {
	TaskID               string                   `json:"task_id,omitempty"`
	PlanRevision         int                      `json:"work_plan_revision,omitempty"`
	AgentID              string                   `json:"agent_id,omitempty"`
	AgentScore           float64                  `json:"agent_score,omitempty"`
	AgentConfidence      string                   `json:"agent_confidence,omitempty"`
	AgentReason          string                   `json:"agent_reason,omitempty"`
	Requirements         TaskRequirements         `json:"task_requirements"`
	Desired              RuntimeAffinityPolicy    `json:"desired"`
	AffinityPolicy       RuntimeAffinityPolicy    `json:"affinity_policy"`
	Actual               ModelCandidate           `json:"actual"`
	SelectedEngine       string                   `json:"selected_engine,omitempty"`
	SelectedProvider     string                   `json:"selected_provider,omitempty"`
	SelectedProfile      string                   `json:"selected_profile,omitempty"`
	SelectedAccountScope model.AccountScope       `json:"selected_account_scope,omitempty"`
	SelectedModel        string                   `json:"selected_model,omitempty"`
	SelectedReasoning    string                   `json:"selected_reasoning,omitempty"`
	SkillRefs            []string                 `json:"skill_refs,omitempty"`
	SkillResolutions     []nexusskills.Resolution `json:"skill_resolutions,omitempty"`
	MaestroGuidanceRef   string                   `json:"maestro_guidance_ref,omitempty"`
	Alternatives         []ModelCandidate         `json:"alternatives,omitempty"`
	Fallback             bool                     `json:"fallback"`
	Reason               string                   `json:"reason"`
	Rejected             []string                 `json:"rejected,omitempty"`
	RejectedCandidates   []string                 `json:"rejected_candidates,omitempty"`
	TaskClass            string                   `json:"task_class,omitempty"`
	// Explainability extension (additive, no Engine≠Backend≠Transport refactor):
	// requested/selected/fallback/why/alternatives/rejected are projected from
	// the durable decision. SwitchReason carries the real typed switch cause.
	RequestedProvider string              `json:"requested_provider,omitempty"`
	RequestedProfile  string              `json:"requested_profile,omitempty"`
	RequestedModel    string              `json:"requested_model,omitempty"`
	SwitchReason      RoutingSwitchReason `json:"switch_reason,omitempty"`
	SwitchDetail      string              `json:"switch_detail,omitempty"`
	CreatedAt         time.Time           `json:"created_at"`
}

// RoutingSwitchReason is the real typed cause for a resource switch.
// Empty means no switch (selected == requested and no failover).
type RoutingSwitchReason string

const (
	SwitchReasonNone                 RoutingSwitchReason = ""
	SwitchReasonPreferredUnavailable RoutingSwitchReason = "preferred_unavailable"
	SwitchReasonQuotaExhausted       RoutingSwitchReason = "quota_exhausted"
	SwitchReasonRateLimited          RoutingSwitchReason = "rate_limited"
	SwitchReasonCooldown             RoutingSwitchReason = "cooldown"
	SwitchReasonUnauthenticated      RoutingSwitchReason = "unauthenticated"
	SwitchReasonUnhealthy            RoutingSwitchReason = "unhealthy"
	SwitchReasonModelIncapable       RoutingSwitchReason = "model_incapable"
)

// DeriveRoutingSwitchReason maps a human reason + failure evidence to the
// typed switch cause. Typed failure kinds win; free prose is classified
// through internal/core/classifier (the only place allowed to match failure
// text for quota/rate-limit — provider guardrails enforce this). Unknown
// prose falls back to preferred_unavailable only when a fallback happened.
func DeriveRoutingSwitchReason(fallback bool, reason string, failureKind model.FailureKind, failureMessage string) RoutingSwitchReason {
	if !fallback {
		return SwitchReasonNone
	}
	// Typed failure evidence first — never report everything as rate limit.
	switch failureKind {
	case model.FailureQuota:
		return SwitchReasonQuotaExhausted
	case model.FailureRateLimit:
		if mentionsCooldown(reason + " " + failureMessage) {
			return SwitchReasonCooldown
		}
		return SwitchReasonRateLimited
	case model.FailureAuth:
		return SwitchReasonUnauthenticated
	case model.FailureNetwork, model.FailureProvider:
		return SwitchReasonUnhealthy
	}
	prose := reason + " " + failureMessage
	switch classifier.Classify(nil, prose).Kind {
	case model.FailureQuota:
		return SwitchReasonQuotaExhausted
	case model.FailureRateLimit:
		if mentionsCooldown(prose) {
			return SwitchReasonCooldown
		}
		return SwitchReasonRateLimited
	case model.FailureAuth:
		return SwitchReasonUnauthenticated
	case model.FailureNetwork, model.FailureProvider:
		return SwitchReasonUnhealthy
	}
	lower := strings.ToLower(prose)
	switch {
	case mentionsCooldown(prose):
		return SwitchReasonCooldown
	case strings.Contains(lower, "incapable") || (strings.Contains(lower, "model") && strings.Contains(lower, "fallback")):
		return SwitchReasonModelIncapable
	case strings.Contains(lower, "unhealthy") || (strings.Contains(lower, "unavailable") && !strings.Contains(lower, "preferred")) || strings.Contains(lower, "health"):
		return SwitchReasonUnhealthy
	case strings.Contains(lower, "preferred"):
		return SwitchReasonPreferredUnavailable
	case strings.Contains(lower, "model"):
		return SwitchReasonModelIncapable
	default:
		return SwitchReasonPreferredUnavailable
	}
}

// mentionsCooldown is the only inline failure-text match allowed here:
// "cooldown" is not a quota/rate-limit substring, so the provider substring
// guardrail does not apply to it.
func mentionsCooldown(prose string) bool {
	return strings.Contains(strings.ToLower(prose), "cooldown")
}

type ExecutionRoutingDecision = RuntimeRoutingDecision

var ErrBlockedResource = errors.New("blocked resource")

// Typed Mission allocation failures — never blame quota/rate-limit when the
// pool is unregistered, unauthenticated, or UNKNOWN.
var (
	ErrAllocationUnregistered    = errors.New("ALLOCATION_UNREGISTERED")
	ErrAllocationUnauthenticated = errors.New("ALLOCATION_UNAUTHENTICATED")
	ErrAllocationUnknownQuota    = errors.New("ALLOCATION_UNKNOWN_QUOTA")
	ErrAllocationQuotaExhausted  = errors.New("ALLOCATION_QUOTA_EXHAUSTED")
	ErrAllocationRateLimited     = errors.New("ALLOCATION_RATE_LIMITED")
	ErrAllocationNetwork         = errors.New("ALLOCATION_NETWORK")
)

// ResolveTaskModel selects a model for one WorkUnit after the provider/profile
// scheduler has selected an account. Model candidates are configuration
// metadata; health, authentication and quota are always taken from the live
// account/candidate evidence supplied by the caller.
func ResolveTaskModel(req TaskRequirements, preference AffinityPreference, account ProviderAccount, models []ModelCandidate) (ModelCandidate, bool, string, error) {
	pool, fallback, fallbackReason, err := resolveTaskModelPool(req, preference, account, models)
	if err != nil {
		return ModelCandidate{}, fallback, "", err
	}
	selected := chooseTaskModel(pool, req)
	reason := fmt.Sprintf("selected %s for task class %s at escalation level %d", selected.Model, req.TaskKind, req.EscalationLevel)
	if fallbackReason != "" {
		reason = fmt.Sprintf("%s: %s", fallbackReason, reason)
	}
	return selected, fallback, reason, nil
}

func ResolveRuntimeRouting(req TaskRequirements, policy RuntimeAffinityPolicy, accounts []ProviderAccount, models []ModelCandidate) (RuntimeRoutingDecision, error) {
	decision := RuntimeRoutingDecision{
		Desired: policy, AffinityPolicy: policy, Requirements: req, TaskClass: req.TaskKind,
		RequestedProvider: policy.Provider.Value, RequestedProfile: policy.Profile.Value, RequestedModel: policy.Model.Value,
		CreatedAt: time.Now().UTC(),
	}
	accountCandidates := eligibleAccounts(accounts)
	if policy.Provider.Mode == AffinityPin && !containsAccount(accountCandidates, "provider", policy.Provider.Value) {
		return decision, fmt.Errorf("%w: pinned provider %q unavailable", ErrBlockedResource, policy.Provider.Value)
	}
	if policy.Profile.Mode == AffinityPin && !containsAccount(accountCandidates, "profile", policy.Profile.Value) {
		return decision, fmt.Errorf("%w: pinned profile %q unavailable", ErrBlockedResource, policy.Profile.Value)
	}
	if policy.Model.Mode == AffinityPin && !containsModel(models, policy.Model.Value) {
		return decision, fmt.Errorf("%w: pinned model %q unavailable", ErrBlockedResource, policy.Model.Value)
	}

	filtered := filterAccounts(accountCandidates, policy)
	if len(filtered) == 0 {
		if policy.Provider.Mode == AffinityPin || policy.Profile.Mode == AffinityPin {
			return decision, fmt.Errorf("%w: pinned runtime has no eligible account", ErrBlockedResource)
		}
		filtered = accountCandidates
		decision.Fallback = policy.Provider.Mode == AffinityPrefer || policy.Profile.Mode == AffinityPrefer
		if decision.Fallback {
			decision.Reason = "preferred provider/profile unavailable; scheduler fallback permitted"
		}
	}
	if len(filtered) == 0 {
		return decision, fmt.Errorf("%w: no eligible provider profile", ErrBlockedResource)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].Available != filtered[j].Available {
			return filtered[i].Available
		}
		if filtered[i].QuotaRemaining != filtered[j].QuotaRemaining {
			return filtered[i].QuotaRemaining > filtered[j].QuotaRemaining
		}
		return filtered[i].Provider+":"+filtered[i].Profile < filtered[j].Provider+":"+filtered[j].Profile
	})
	chosenAccount := filtered[0]

	modelPool, modelFallback, modelReason, err := resolveTaskModelPool(req, policy.Model, chosenAccount, models)
	if err != nil {
		return decision, err
	}
	decision.Fallback = decision.Fallback || modelFallback
	if modelReason != "" {
		decision.Reason = firstNonEmpty(decision.Reason, modelReason)
	}
	chosenModel := chooseTaskModel(modelPool, req)
	decision.Actual = chosenModel
	decision.SelectedProvider = chosenModel.Provider
	decision.SelectedProfile = chosenModel.Profile
	decision.SelectedAccountScope = chosenAccount.Scope
	decision.SelectedModel = chosenModel.Model
	decision.SelectedReasoning = fmt.Sprintf("cost_rank=%d; reasoning_rank=%d", chosenModel.CostRank, chosenModel.ReasoningRank)
	for _, candidate := range modelPool {
		if candidate.Provider == chosenModel.Provider && candidate.Profile == chosenModel.Profile && candidate.Model == chosenModel.Model {
			continue
		}
		decision.Alternatives = append(decision.Alternatives, candidate)
	}
	// Rejected records the eligible-but-unselected accounts with the real
	// reason they lost (score vs requested preference), not a generic message.
	for _, acc := range filtered[1:] {
		decision.Rejected = append(decision.Rejected,
			fmt.Sprintf("%s:%s not selected (quota=%.2f available=%t)", acc.Provider, acc.Profile, acc.QuotaRemaining, acc.Available))
	}
	decision.RejectedCandidates = append([]string(nil), decision.Rejected...)
	if decision.Reason == "" {
		decision.Reason = fmt.Sprintf("selected %s for task class %s using %s affinity", chosenModel.Model, req.TaskKind, affinitySummary(policy))
	}
	decision.SwitchReason = DeriveRoutingSwitchReason(decision.Fallback, decision.Reason, "", "")
	decision.SwitchDetail = decision.Reason
	return decision, nil
}

func resolveTaskModelPool(req TaskRequirements, preference AffinityPreference, account ProviderAccount, models []ModelCandidate) ([]ModelCandidate, bool, string, error) {
	allPool := filterModels(models, account, RuntimeAffinityPolicy{Model: AffinityPreference{Mode: AffinityAuto}})
	preferredPool := filterModels(models, account, RuntimeAffinityPolicy{Model: preference})
	pool := allPool
	fallback := false
	switch preference.Mode {
	case AffinityPin:
		if len(preferredPool) == 0 || !hasTaskCapableModel(preferredPool, req) {
			return nil, false, "", fmt.Errorf("%w: pinned model %q unavailable or incapable for task", ErrBlockedResource, preference.Value)
		}
		pool = preferredPool
	case AffinityPrefer:
		if len(preferredPool) > 0 && hasTaskCapableModel(preferredPool, req) {
			pool = preferredPool
		} else {
			fallback = true
		}
	}
	if len(pool) == 0 {
		return nil, fallback, "", fmt.Errorf("%w: no model candidate for %s:%s", ErrBlockedResource, account.Provider, account.Profile)
	}
	if !hasTaskCapableModel(pool, req) {
		return nil, fallback, "", fmt.Errorf("%w: no capable model candidate for task %s", ErrBlockedResource, req.TaskKind)
	}
	reason := ""
	if fallback && preference.Mode == AffinityPrefer {
		reason = fmt.Sprintf("preferred model %q unavailable or incapable; capable fallback selected", preference.Value)
	}
	return pool, fallback, reason, nil
}

func eligibleAccounts(accounts []ProviderAccount) []ProviderAccount {
	result := make([]ProviderAccount, 0, len(accounts))
	for _, account := range accounts {
		if quota.EvaluateEligibility(accountEligibilityInput(account, false)).Eligible {
			result = append(result, account)
		}
	}
	return result
}

func filterAccounts(accounts []ProviderAccount, policy RuntimeAffinityPolicy) []ProviderAccount {
	result := append([]ProviderAccount(nil), accounts...)
	if policy.Provider.Value != "" && policy.Provider.Mode != AffinityAuto {
		result = filterAccountField(result, "provider", policy.Provider.Value)
	}
	if policy.Profile.Value != "" && policy.Profile.Mode != AffinityAuto {
		result = filterAccountField(result, "profile", policy.Profile.Value)
	}
	return result
}

func filterAccountField(accounts []ProviderAccount, field, value string) []ProviderAccount {
	result := make([]ProviderAccount, 0, len(accounts))
	for _, account := range accounts {
		if (field == "provider" && account.Provider == value) || (field == "profile" && account.Profile == value) {
			result = append(result, account)
		}
	}
	return result
}

func filterModels(models []ModelCandidate, account ProviderAccount, policy RuntimeAffinityPolicy) []ModelCandidate {
	result := make([]ModelCandidate, 0, len(models))
	for _, model := range models {
		if model.Provider != account.Provider || model.Profile != account.Profile || !model.Healthy || !model.Authenticated || !model.QuotaAvailable {
			continue
		}
		if policy.Model.Value != "" && policy.Model.Mode != AffinityAuto && model.Model != policy.Model.Value {
			continue
		}
		result = append(result, model)
	}
	return result
}

func chooseTaskModel(models []ModelCandidate, req TaskRequirements) ModelCandidate {
	sort.SliceStable(models, func(i, j int) bool {
		if models[i].CostRank != models[j].CostRank {
			return models[i].CostRank < models[j].CostRank
		}
		if models[i].ReasoningRank != models[j].ReasoningRank {
			return models[i].ReasoningRank < models[j].ReasoningRank
		}
		return models[i].Model < models[j].Model
	})
	minimumReasoning := minimumReasoningForTask(req)
	capable := make([]ModelCandidate, 0, len(models))
	for _, model := range models {
		if model.ReasoningRank >= minimumReasoning {
			capable = append(capable, model)
		}
	}
	if len(capable) == 0 {
		capable = models
	}
	level := req.EscalationLevel
	if level < 0 {
		level = 0
	}
	if level >= len(capable) {
		level = len(capable) - 1
	}
	return capable[level]
}

func minimumReasoningForTask(req TaskRequirements) int {
	if req.TaskKind == "architecture" || req.TaskKind == "security" || req.EstimatedComplexity == "high" {
		return 3
	}
	return 1
}

func hasTaskCapableModel(models []ModelCandidate, req TaskRequirements) bool {
	minimum := minimumReasoningForTask(req)
	for _, model := range models {
		if model.ReasoningRank >= minimum {
			return true
		}
	}
	return false
}

func containsAccount(accounts []ProviderAccount, field, value string) bool {
	for _, account := range accounts {
		if (field == "provider" && account.Provider == value) || (field == "profile" && account.Profile == value) {
			return true
		}
	}
	return false
}

func containsModel(models []ModelCandidate, value string) bool {
	for _, model := range models {
		if model.Model == value && model.Healthy && model.Authenticated && model.QuotaAvailable {
			return true
		}
	}
	return false
}

func affinitySummary(policy RuntimeAffinityPolicy) string {
	parts := []string{string(policy.Provider.Mode), string(policy.Profile.Mode), string(policy.Model.Mode)}
	return strings.Join(parts, "/")
}
