package nexus

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

type CapacityDomainID string

type CapacityDomain struct {
	ID            CapacityDomainID
	Provider      string
	AccountScope  model.AccountScope
	Evidence      string
	CapacityGroup string
}

type ModelCatalogEntry struct {
	Candidate        ModelCandidate
	Source           string
	ObservedAt       time.Time
	ExpiresAt        time.Time
	CapacityDomainID CapacityDomainID
}

// ModelCatalog contains only observed/configured model evidence. It never
// manufactures provider model names. Entries may be refreshed by adapters,
// doctor probes or explicit configuration in later integrations.
type ModelCatalog struct {
	mu      sync.RWMutex
	entries []ModelCatalogEntry
}

func NewModelCatalog(entries []ModelCatalogEntry) *ModelCatalog {
	catalog := &ModelCatalog{}
	catalog.Replace(entries)
	return catalog
}

func (c *ModelCatalog) Replace(entries []ModelCatalogEntry) {
	if c == nil {
		return
	}
	now := time.Now().UTC()
	clean := make([]ModelCatalogEntry, 0, len(entries))
	seen := map[string]struct{}{}
	for _, entry := range entries {
		entry.Candidate.Provider = strings.ToLower(strings.TrimSpace(entry.Candidate.Provider))
		entry.Candidate.Profile = strings.TrimSpace(entry.Candidate.Profile)
		entry.Candidate.Model = strings.TrimSpace(entry.Candidate.Model)
		if entry.Candidate.Provider == "" || entry.Candidate.Model == "" {
			continue
		}
		if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
			continue
		}
		key := entry.Candidate.Provider + "|" + strings.ToLower(entry.Candidate.Profile) + "|" + strings.ToLower(entry.Candidate.Model)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		clean = append(clean, entry)
	}
	sort.SliceStable(clean, func(i, j int) bool {
		left := clean[i].Candidate.Provider + "|" + clean[i].Candidate.Profile + "|" + clean[i].Candidate.Model
		right := clean[j].Candidate.Provider + "|" + clean[j].Candidate.Profile + "|" + clean[j].Candidate.Model
		return left < right
	})
	c.mu.Lock()
	c.entries = clean
	c.mu.Unlock()
}

func (c *ModelCatalog) Entries() []ModelCatalogEntry {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ModelCatalogEntry, len(c.entries))
	copy(out, c.entries)
	return out
}

func (c *ModelCatalog) Has(provider, profileName, modelName string) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	profileName = strings.TrimSpace(profileName)
	modelName = strings.TrimSpace(modelName)
	for _, entry := range c.Entries() {
		if entry.Candidate.Provider != provider || !strings.EqualFold(entry.Candidate.Model, modelName) {
			continue
		}
		if entry.Candidate.Profile == "" || profileName == "" || strings.EqualFold(entry.Candidate.Profile, profileName) {
			return true
		}
	}
	return false
}

func (c *ModelCatalog) Candidates(provider, profileName string) []ModelCandidate {
	provider = strings.ToLower(strings.TrimSpace(provider))
	profileName = strings.TrimSpace(profileName)
	out := make([]ModelCandidate, 0)
	for _, entry := range c.Entries() {
		if entry.Candidate.Provider != provider {
			continue
		}
		if entry.Candidate.Profile != "" && profileName != "" && !strings.EqualFold(entry.Candidate.Profile, profileName) {
			continue
		}
		out = append(out, entry.Candidate)
	}
	return out
}

type ExecutionTarget struct {
	Backend          string
	Profile          string
	AccountScope     model.AccountScope
	Model            string
	Capabilities     []string
	RuntimeReady     bool
	Authenticated    bool
	Healthy          bool
	CapacityEvidence string
	CapacityDomainID CapacityDomainID
}

func BuildExecutionTargets(accounts []ProviderAccount, catalog *ModelCatalog) []ExecutionTarget {
	targets := make([]ExecutionTarget, 0)
	entries := catalog.Entries()
	for _, account := range accounts {
		elig := quota.EvaluateEligibility(accountEligibilityInput(account, false))
		baseDomain := capacityDomainForAccount(account)
		matched := false
		for _, entry := range entries {
			if !strings.EqualFold(entry.Candidate.Provider, account.Provider) {
				continue
			}
			if entry.Candidate.Profile != "" && !strings.EqualFold(entry.Candidate.Profile, account.Profile) {
				continue
			}
			matched = true
			domain := baseDomain.ID
			if entry.CapacityDomainID != "" {
				domain = entry.CapacityDomainID
			}
			targets = append(targets, ExecutionTarget{
				Backend:          account.Provider,
				Profile:          account.Profile,
				AccountScope:     account.Scope,
				Model:            entry.Candidate.Model,
				Capabilities:     append([]string(nil), entry.Candidate.Capabilities...),
				RuntimeReady:     elig.Eligible,
				Authenticated:    account.Authenticated,
				Healthy:          account.Health == "" || strings.EqualFold(account.Health, "healthy"),
				CapacityEvidence: elig.CapacityEvidence,
				CapacityDomainID: domain,
			})
		}
		if !matched {
			targets = append(targets, ExecutionTarget{
				Backend:          account.Provider,
				Profile:          account.Profile,
				AccountScope:     account.Scope,
				RuntimeReady:     elig.Eligible,
				Authenticated:    account.Authenticated,
				Healthy:          account.Health == "" || strings.EqualFold(account.Health, "healthy"),
				CapacityEvidence: elig.CapacityEvidence,
				CapacityDomainID: baseDomain.ID,
			})
		}
	}
	sort.SliceStable(targets, func(i, j int) bool {
		left := targets[i].Backend + "|" + targets[i].Profile + "|" + targets[i].Model
		right := targets[j].Backend + "|" + targets[j].Profile + "|" + targets[j].Model
		return left < right
	})
	return targets
}

func capacityDomainForAccount(account ProviderAccount) CapacityDomain {
	identity := ""
	evidence := "PROFILE_FALLBACK"
	if account.Scope.Verifiable() {
		identity = account.Scope.Key()
		evidence = "ACCOUNT_SCOPE"
	} else {
		identity = strings.ToLower(strings.TrimSpace(account.Provider)) + "/" + strings.ToLower(strings.TrimSpace(account.Profile))
	}
	return CapacityDomain{
		ID:           CapacityDomainID("account:" + identity),
		Provider:     account.Provider,
		AccountScope: account.Scope,
		Evidence:     evidence,
	}
}

func applySpecialistExecutionPreferences(req TaskRequirements, affinity RuntimeAffinityPolicy, catalog *ModelCatalog) RuntimeAffinityPolicy {
	mode, ok := AgentModeForRole(req.Role)
	if !ok {
		return affinity
	}
	prefs := AgentModeExecutionPreferences(mode, catalog)
	for _, target := range prefs.Targets {
		if affinity.Provider.Mode == AffinityAuto && strings.TrimSpace(target.Backend) != "" {
			affinity.Provider = AffinityPreference{Mode: AffinityPrefer, Value: target.Backend}
		}
		if affinity.Model.Mode == AffinityAuto && strings.TrimSpace(target.Model) != "" {
			if target.RequireCatalogEvidence && (catalog == nil || !catalog.Has(target.Backend, "", target.Model)) {
				continue
			}
			affinity.Model = AffinityPreference{Mode: AffinityPrefer, Value: target.Model}
			if affinity.Provider.Mode == AffinityAuto && strings.TrimSpace(target.Backend) != "" {
				affinity.Provider = AffinityPreference{Mode: AffinityPrefer, Value: target.Backend}
			}
			break
		}
	}
	return affinity
}

func ResolveExecutionTarget(req TaskRequirements, affinity RuntimeAffinityPolicy, accounts []ProviderAccount, catalog *ModelCatalog) (ExecutionTarget, RuntimeRoutingDecision, error) {
	affinity = applySpecialistExecutionPreferences(req, affinity, catalog)
	models := make([]ModelCandidate, 0)
	if catalog != nil {
		for _, entry := range catalog.Entries() {
			models = append(models, entry.Candidate)
		}
	}
	if len(models) == 0 {
		if affinity.Model.Mode == AffinityPin && strings.TrimSpace(affinity.Model.Value) != "" {
			return ExecutionTarget{}, RuntimeRoutingDecision{}, fmt.Errorf("pinned model %q is absent from ModelCatalog", affinity.Model.Value)
		}
		rec := RecommendResources(accounts, req, PolicyBalanced)
		if affinity.Provider.Mode == AffinityPin && affinity.Provider.Value != "" {
			pinned := make([]ProviderAccount, 0)
			for _, account := range accounts {
				if strings.EqualFold(account.Provider, affinity.Provider.Value) {
					pinned = append(pinned, account)
				}
			}
			rec = RecommendResources(pinned, req, PolicyBalanced)
		}
		if rec.Recommended == nil {
			return ExecutionTarget{}, RuntimeRoutingDecision{}, fmt.Errorf("no eligible execution target: %s", rec.Explanation)
		}
		account := rec.Recommended.Account
		elig := quota.EvaluateEligibility(accountEligibilityInput(account, false))
		target := ExecutionTarget{
			Backend:          account.Provider,
			Profile:          account.Profile,
			AccountScope:     account.Scope,
			RuntimeReady:     elig.Eligible,
			Authenticated:    account.Authenticated,
			Healthy:          account.Health == "" || strings.EqualFold(account.Health, "healthy"),
			CapacityEvidence: elig.CapacityEvidence,
			CapacityDomainID: capacityDomainForAccount(account).ID,
		}
		decision := RuntimeRoutingDecision{
			Requirements:         req,
			Desired:              affinity,
			AffinityPolicy:       affinity,
			SelectedProvider:     account.Provider,
			SelectedProfile:      account.Profile,
			SelectedAccountScope: account.Scope,
			Reason:               rec.Explanation,
			CreatedAt:            time.Now().UTC(),
		}
		return target, decision, nil
	}

	decision, err := ResolveRuntimeRouting(req, affinity, accounts, models)
	if err != nil {
		return ExecutionTarget{}, decision, err
	}
	for _, target := range BuildExecutionTargets(accounts, catalog) {
		if !strings.EqualFold(target.Backend, decision.SelectedProvider) ||
			!strings.EqualFold(target.Profile, decision.SelectedProfile) ||
			!strings.EqualFold(target.Model, decision.SelectedModel) {
			continue
		}
		return target, decision, nil
	}
	return ExecutionTarget{}, decision, fmt.Errorf("selected runtime route has no ExecutionTarget projection")
}
