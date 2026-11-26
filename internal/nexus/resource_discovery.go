package nexus

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/profile"
)

// ResourceAllocation is the durable result of assigning a provider profile to
// an Agent. A scheduler recommendation alone never changes Agent state.
type ResourceAllocation struct {
	Decision  SchedulerDecision `json:"decision"`
	Impact    *ConfigImpact     `json:"impact,omitempty"`
	Persisted bool              `json:"persisted"`
}

func (n *Nexus) ListResources() ([]ProviderAccount, error) {
	return n.ListResourcesContext(context.Background())
}

// ListResourcesContext discovers provider accounts using the caller's
// cancellation context. ListResources remains the backwards-compatible
// context-free facade used by existing CLI/domain callers.
func (n *Nexus) ListResourcesContext(ctx context.Context) ([]ProviderAccount, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	profiles, err := profile.List()
	if err != nil {
		return nil, fmt.Errorf("list profiles: %w", err)
	}

	cfg, cfgErr := config.LoadConfig()

	type providerDetect struct {
		installed bool
		ok        bool
		caps      map[string]string
	}
	detectCache := map[string]providerDetect{}
	var detectMu sync.Mutex
	var detectWG sync.WaitGroup
	uniqueProviders := make(map[string]struct{}, len(profiles))
	for _, p := range profiles {
		uniqueProviders[p.Provider] = struct{}{}
	}
	for prov := range uniqueProviders {
		detectWG.Add(1)
		go func(provider string) {
			defer detectWG.Done()
			out := providerDetect{caps: map[string]string{}}
			d, derr := n.controlDrivers().Get(provider)
			if derr == nil {
				out.ok = true
				pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
				if det, detectErr := d.Detect(pctx); detectErr == nil {
					out.installed = det.Installed
				}
				cancel()
			}
			detectMu.Lock()
			detectCache[provider] = out
			detectMu.Unlock()
		}(prov)
	}
	detectWG.Wait()

	accounts := make([]ProviderAccount, len(profiles))
	sem := make(chan struct{}, 6)
	var accWG sync.WaitGroup
	now := time.Now()
	for i, prof := range profiles {
		accWG.Add(1)
		go func(idx int, p model.Profile) {
			defer accWG.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			id := p.Provider + ":" + p.Name
			displayName := p.Provider + " (" + p.Name + ")"

			acc, _ := profile.GetAccountInfoContext(ctx, p.Provider, p.Name)
			authenticated := false
			health := strings.ToLower(string(acc.Health))
			if health == "" {
				health = "unknown"
			}

			capabilities := map[string]string{}
			det := detectCache[p.Provider]
			if det.ok {
				authenticated = det.installed && acc.Authenticated
				if !det.installed {
					health = "unavailable"
				} else if !acc.Authenticated {
					health = "auth_required"
				}
				if d, derr := n.controlDrivers().Get(p.Provider); derr == nil {
					capabilities = effectiveCapabilityMap(d.EffectiveCaps(ctx, p))
				}
			} else {
				health = "unavailable"
			}

			isDefault := false
			if cfgErr == nil {
				isDefault = cfg.Defaults[p.Provider] == p.Name
			}

			usageSnapshot := profile.GetSanitizedUsageSnapshot(p.Provider, p.Name, acc.Email)
			qv := profile.GetQuotaView(p.Provider, p.Name, acc.Plan, acc.Email)
			available := reconcileResourceAvailability(&qv, det.ok, det.installed, authenticated)
			proj := qv.ProjectionAt(now, quota.DefaultTTL, authenticated, health)
			quotaRemaining := quotaRemainingRatio(qv)
			if proj.EffectiveRemaining != nil {
				ratio := *proj.EffectiveRemaining / 100.0
				if ratio < 0 {
					ratio = 0
				} else if ratio > 1 {
					ratio = 1
				}
				quotaRemaining = ratio
			}
			quotaTotal := quotaTotalRatio(qv)
			resourceChannels := subscriptionResourceChannels(usageSnapshot)

			item := ProviderAccount{
				ID:               id,
				Provider:         p.Provider,
				Profile:          p.Name,
				DisplayName:      displayName,
				Authenticated:    authenticated,
				Health:           health,
				IsDefault:        isDefault,
				Available:        available,
				AvailReasons:     &qv.AvailReasons,
				QuotaView:        &qv,
				QuotaProjection:  &proj,
				Capabilities:     capabilities,
				ResourceChannels: resourceChannels,
				QuotaRemaining:   quotaRemaining,
				QuotaTotal:       quotaTotal,
				RateLimited:      qv.Status == "RATE_LIMITED",
				NetworkError:     strings.EqualFold(strings.TrimSpace(qv.Status), "ERROR"),
				Scope:            acc.AccountScope,
				LastChecked:      now,
			}
			attachCredentialProvenance(&item, p.Provider, p.Name, authenticated, acc.ExternalAccountID)
			elig := quota.EvaluateEligibility(accountEligibilityInput(item, false))
			item.Eligibility = &elig
			accounts[idx] = item
		}(i, prof)
	}
	accWG.Wait()

	sort.Slice(accounts, func(i, j int) bool {
		if accounts[i].Provider != accounts[j].Provider {
			return accounts[i].Provider < accounts[j].Provider
		}
		return accounts[i].Profile < accounts[j].Profile
	})

	return accounts, nil
}

// AllocateResource validates an exact provider/profile choice against the
// current discovery result and persists it as the Agent's current revision.
func (n *Nexus) AllocateResource(ctx context.Context, agentID, provider, profileName string, policy SchedulerPolicy) (*ResourceAllocation, error) {
	accounts, err := n.ListResourcesContext(ctx)
	if err != nil {
		return nil, err
	}
	return n.allocateResourceFromAccounts(ctx, agentID, provider, profileName, accounts, policy)
}

// ValidateResource verifies that an exact provider/profile is currently
// discoverable and eligible, without changing Agent configuration.
func (n *Nexus) ValidateResource(provider, profileName string) (ProviderAccount, error) {
	accounts, err := n.ListResources()
	if err != nil {
		return ProviderAccount{}, err
	}
	return validateResourceFromAccounts(provider, profileName, accounts)
}

func (n *Nexus) allocateResourceFromAccounts(ctx context.Context, agentID, provider, profileName string, accounts []ProviderAccount, policy SchedulerPolicy) (*ResourceAllocation, error) {
	provider = strings.TrimSpace(provider)
	profileName = strings.TrimSpace(profileName)
	if provider == "" || profileName == "" {
		return nil, fmt.Errorf("provider and profile are required")
	}
	if policy == "" {
		policy = PolicyBalanced
	}

	selected, err := validateResourceFromAccounts(provider, profileName, accounts)
	if err != nil {
		return nil, err
	}

	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return nil, err
	}
	var config AgentConfig
	if agent.CurrentRevisionID != "" {
		if revision, revisionErr := st.GetRevision(agent.CurrentRevisionID); revisionErr == nil {
			config, _ = ParseAgentConfig(revision.Config)
		}
	}
	config.Provider = provider
	config.Profile = profileName

	impact, err := n.SafeApply(ctx, agentID, config)
	if err != nil {
		return nil, err
	}
	return &ResourceAllocation{
		Decision: SchedulerDecision{
			Selected:    selected,
			Policy:      policy,
			Reason:      "manually selected eligible resource",
			Score:       1,
			ExplainPath: []string{"manual exact provider/profile selection", "resource validated", "agent configuration revision persisted"},
		},
		Impact:    impact,
		Persisted: true,
	}, nil
}

func validateResourceFromAccounts(provider, profileName string, accounts []ProviderAccount) (ProviderAccount, error) {
	provider = strings.TrimSpace(provider)
	profileName = strings.TrimSpace(profileName)
	if provider == "" || profileName == "" {
		return ProviderAccount{}, fmt.Errorf("provider and profile are required")
	}
	for _, account := range accounts {
		if account.Provider == provider && account.Profile == profileName {
			if err := validateResourceAccount(account); err != nil {
				return ProviderAccount{}, err
			}
			return account, nil
		}
	}
	return ProviderAccount{}, fmt.Errorf("resource %s:%s was not found", provider, profileName)
}

func validateResourceAccount(account ProviderAccount) error {
	elig := quota.EvaluateEligibility(accountEligibilityInput(account, true))
	if !elig.Eligible {
		if elig.Message != "" {
			return fmt.Errorf("resource %s: %s", account.ID, elig.Message)
		}
		return fmt.Errorf("resource %s is ineligible (%s)", account.ID, elig.Reason)
	}
	return nil
}

func (n *Nexus) ResolveStartParams(agentID, provider, profile string) (string, string, error) {
	if strings.TrimSpace(provider) != "" || strings.TrimSpace(profile) != "" {
		return "", "", fmt.Errorf("start only accepts a persisted resource; allocate provider and profile to the agent first")
	}

	st, err := n.OpenProject()
	if err != nil {
		return "", "", err
	}

	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return "", "", err
	}

	if agent.CurrentRevisionID != "" {
		rev, rerr := st.GetRevision(agent.CurrentRevisionID)
		if rerr == nil {
			cfg, _ := ParseAgentConfig(rev.Config)
			if cfg.Provider != "" {
				p := cfg.Profile
				if p == "" {
					p = "default"
				}
				return cfg.Provider, p, nil
			}
		}
	}

	return "", "", fmt.Errorf("REQUIRED_RESOURCE_SELECTION: agent %s has no configured provider; select a resource first", agentID)
}

func subscriptionResourceChannels(snap model.UsageSnapshot) []model.ResourceChannelState {
	if snap.ProviderID == "" && snap.ProfileID == "" {
		return nil
	}
	copy := snap
	return []model.ResourceChannelState{{
		Channel: model.ResourceChannelSubscription,
		Quota:   &copy,
	}}
}

func reconcileResourceAvailability(qv *quota.QuotaView, detectionKnown, installed, authenticated bool) bool {
	if qv == nil {
		return false
	}
	available := qv.IsAvailable() && detectionKnown && installed && authenticated
	if !detectionKnown || !installed {
		qv.Available = false
		qv.AvailReasons.AllOK = false
		return false
	}
	if !authenticated {
		qv.Available = false
		qv.AvailReasons.AuthRequired = true
		qv.AvailReasons.AllOK = false
		return false
	}
	return available
}

func quotaTotalRatio(qv quota.QuotaView) float64 {
	if !quotaStatusKnown(qv.Status) {
		return 0
	}
	if _, ok := qv.BestGroupRemaining(); !ok {
		return 0
	}
	return 1
}

func quotaRemainingRatio(qv quota.QuotaView) float64 {
	if !quotaStatusKnown(qv.Status) {
		return 0
	}
	remaining, ok := qv.BestGroupRemaining()
	if !ok {
		return 0
	}
	ratio := remaining / 100.0
	if ratio < 0 {
		return 0
	}
	if ratio > 1 {
		return 1
	}
	return ratio
}

func quotaStatusKnown(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "LIVE", "CACHED", "ESTIMATED":
		return true
	default:
		return false
	}
}

func effectiveCapabilityMap(caps driver.EffectiveCapabilities) map[string]string {
	return map[string]string{
		"process":           projectedCapabilityStatus(caps.Process),
		"terminal":          projectedCapabilityStatus(caps.Terminal),
		"attach":            projectedCapabilityStatus(caps.Attach),
		"structured_events": projectedCapabilityStatus(caps.StructuredEvents),
		"sessions":          projectedCapabilityStatus(caps.Sessions),
		"resume":            projectedCapabilityStatus(caps.Resume),
		"fork":              projectedCapabilityStatus(caps.Fork),
		"submit_prompt":     projectedCapabilityStatus(caps.SubmitPrompt),
		"cancel_turn":       projectedCapabilityStatus(caps.CancelTurn),
		"approvals":         projectedCapabilityStatus(caps.Approvals),
		"native_ui_attach":  projectedCapabilityStatus(caps.NativeUIAttach),
		"headless":          projectedCapabilityStatus(caps.Headless),
		"slash_control":     projectedCapabilityStatus(caps.SlashControl),
		"read_only_review":  projectedCapabilityStatus(caps.ReadOnlyReview),
	}
}

func projectedCapabilityStatus(evidence driver.CapabilityEvidence) string {
	if evidence.Status == driver.CapabilitySupported && !evidence.Tested {
		return string(driver.CapabilityNotTested)
	}
	return string(evidence.Status)
}
