package nexus

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	nexusskills "github.com/kivervinicius/ai-cli/internal/nexus/skills"
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
	CreatedAt            time.Time                `json:"created_at"`
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
	decision := RuntimeRoutingDecision{Desired: policy, AffinityPolicy: policy, Requirements: req, TaskClass: req.TaskKind, CreatedAt: time.Now().UTC()}
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
	if decision.Reason == "" {
		decision.Reason = fmt.Sprintf("selected %s for task class %s using %s affinity", chosenModel.Model, req.TaskKind, affinitySummary(policy))
	}
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
		if account.Authenticated && account.Available && !account.RateLimited {
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
