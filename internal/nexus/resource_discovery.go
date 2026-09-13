package nexus

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/driver"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/profile"
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
		d, derr := n.controlDrivers().Get(p.Provider)
		if derr == nil {
			det, detectErr := d.Detect(ctx)
			if detectErr == nil {
				authenticated = det.Installed && acc.Authenticated
				if !det.Installed {
					health = "unavailable"
				} else if !acc.Authenticated {
					health = "auth_required"
				}
			}
			capabilities = effectiveCapabilityMap(d.EffectiveCaps(ctx, p))
		} else {
			health = "unavailable"
		}

		isDefault := false
		if cfgErr == nil {
			isDefault = cfg.Defaults[p.Provider] == p.Name
		}

		// Get quota view with availability. UNKNOWN/unsupported data is never
		// converted into synthetic 100% capacity. Score prefers more usable
		// model families, then higher total group capacity (same as CLI selector).
		qv := profile.GetQuotaView(p.Provider, p.Name, acc.Plan, acc.Email)
		quotaRemaining := 0.0
		if quotaStatusKnown(qv.Status) {
			if ratio, ok := qv.EffectiveCapacityRatio(); ok {
				quotaRemaining = ratio
			}
		}

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
			QuotaView:      &qv,
			Capabilities:   capabilities,
			QuotaRemaining: quotaRemaining,
			QuotaTotal:     1.0,
			RateLimited:    qv.Status == "RATE_LIMITED",
			NetworkError:   strings.EqualFold(strings.TrimSpace(qv.Status), "ERROR"),
			LastChecked:    time.Now(),
		})
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
	switch {
	case !account.Authenticated:
		return fmt.Errorf("resource %s is not authenticated", account.ID)
	case account.RateLimited:
		return fmt.Errorf("resource %s is rate limited", account.ID)
	case account.CooldownUntil != nil && account.CooldownUntil.After(time.Now()):
		return fmt.Errorf("resource %s is in cooldown", account.ID)
	case account.Health == "unhealthy" || account.Health == "unavailable" || account.Health == "auth_required":
		return fmt.Errorf("resource %s is unavailable (%s)", account.ID, account.Health)
	case !account.Available:
		return fmt.Errorf("resource %s is unavailable", account.ID)
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
