package nexus

import (
	"context"
	"fmt"
	"sort"
	"strings"
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
	detectionFor := func(provider string) providerDetect {
		if cached, ok := detectCache[provider]; ok {
			return cached
		}
		out := providerDetect{caps: map[string]string{}}
		d, derr := n.controlDrivers().Get(provider)
		if derr != nil {
			detectCache[provider] = out
			return out
		}
		out.ok = true
		if det, detectErr := d.Detect(ctx); detectErr == nil {
			out.installed = det.Installed
		}
		// EffectiveCaps is profile-specific; filled per profile below.
		detectCache[provider] = out
		return out
	}

	accounts := make([]ProviderAccount, 0, len(profiles))
	for _, p := range profiles {
		id := p.Provider + ":" + p.Name
		displayName := p.Provider + " (" + p.Name + ")"

		acc := profile.GetAccountInfo(p.Provider, p.Name)
		authenticated := false
		health := strings.ToLower(string(acc.Health))
		if health == "" {
			health = "unknown"
		}

		capabilities := map[string]string{}
		det := detectionFor(p.Provider)
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

		// Get quota view with availability. quota_remaining has one stable
		// semantic across providers: the best currently usable pool's literal
		// remaining quota, normalized to 0..1. Multi-pool breadth belongs to the
		// scheduler ranking and must never be encoded into this public field.
		usageSnapshot := profile.GetUsageSnapshot(p.Provider, p.Name)
		qv := quota.BuildQuotaView(usageSnapshot, acc.Email, acc.Plan)
		quotaRemaining := quotaRemainingRatio(qv)
		resourceChannels := subscriptionResourceChannels(usageSnapshot)

		accounts = append(accounts, ProviderAccount{
			ID:             id,
			Provider:       p.Provider,
			Profile:        p.Name,
			DisplayName:    displayName,
			Authenticated:  authenticated,
			Health:         health,
			IsDefault:      isDefault,
			Available:      qv.IsAvailable(),
			AvailReasons:   &qv.AvailReasons,
			QuotaView:        &qv,
			Capabilities:     capabilities,
			ResourceChannels: resourceChannels,
			QuotaRemaining:   quotaRemaining,
			QuotaTotal:     1.0,
			RateLimited:    qv.Status == "RATE_LIMITED",
			NetworkError:   strings.EqualFold(strings.TrimSpace(qv.Status), "ERROR"),
			Scope:          acc.AccountScope,
			LastChecked:    time.Now(),
		})
		accIdx := len(accounts) - 1
		elig := quota.EvaluateEligibility(accountEligibilityInput(accounts[accIdx], false))
		accounts[accIdx].Eligibility = &elig
	}

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
		"process":           string(caps.Process.Status),
		"terminal":          string(caps.Terminal.Status),
		"attach":            string(caps.Attach.Status),
		"structured_events": string(caps.StructuredEvents.Status),
		"sessions":          string(caps.Sessions.Status),
		"resume":            string(caps.Resume.Status),
		"fork":              string(caps.Fork.Status),
		"submit_prompt":     string(caps.SubmitPrompt.Status),
		"cancel_turn":       string(caps.CancelTurn.Status),
		"approvals":         string(caps.Approvals.Status),
		"native_ui_attach":  string(caps.NativeUIAttach.Status),
		"headless":          string(caps.Headless.Status),
		"slash_control":     string(caps.SlashControl.Status),
	}
}
