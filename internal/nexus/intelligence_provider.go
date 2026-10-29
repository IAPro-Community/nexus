package nexus

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/core/classifier"
	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/profile"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

// IntelligenceStatus is the secret-free status exposed by the Nexus REST API.
type IntelligenceStatus struct {
	Mode       coreconfig.IntelligenceMode `json:"mode"`
	Provider   string                      `json:"provider,omitempty"`
	Profile    string                      `json:"profile,omitempty"`
	BaseURL    string                      `json:"base_url,omitempty"`
	Model      string                      `json:"model,omitempty"`
	APIKeyEnv  string                      `json:"api_key_env,omitempty"`
	APIKeyFile string                      `json:"api_key_file,omitempty"`
	Available  bool                        `json:"available"`
	Error      string                      `json:"error,omitempty"`
}

// GetIntelligenceConfig returns persisted non-secret configuration.
func (n *Nexus) GetIntelligenceConfig() (coreconfig.IntelligenceConfig, error) {
	cfg, err := coreconfig.LoadConfig()
	if err != nil {
		return coreconfig.IntelligenceConfig{}, err
	}
	return cfg.Intelligence, nil
}

// SetIntelligenceConfig persists only routing metadata. The API contract does not
// contain a raw API key field; secrets must be injected by environment or file.
func (n *Nexus) SetIntelligenceConfig(next coreconfig.IntelligenceConfig) error {
	cfg, err := coreconfig.LoadConfig()
	if err != nil {
		return err
	}
	cfg.Intelligence = next
	if issues := cfg.Validate(); len(issues) > 0 {
		return fmt.Errorf("invalid intelligence configuration: %s", strings.Join(issues, "; "))
	}
	return coreconfig.SaveConfig(cfg)
}

func (n *Nexus) IntelligenceStatus(ctx context.Context, projectID string) IntelligenceStatus {
	cfg, err := n.GetIntelligenceConfig()
	status := IntelligenceStatus{
		Mode: cfg.Mode, Provider: cfg.Provider, Profile: cfg.Profile, BaseURL: cfg.BaseURL,
		Model: cfg.Model, APIKeyEnv: cfg.APIKeyEnv, APIKeyFile: cfg.APIKeyFile,
	}
	if err != nil {
		status.Error = err.Error()
		return status
	}
	provider, err := n.ConfiguredIntelligenceProvider(ctx, projectID)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	if provider != nil {
		status.Available = provider.Available(ctx)
	}
	return status
}

// IntelligenceProbeResult is the secret-free outcome of a live provider round-trip.
type IntelligenceProbeResult struct {
	Provider string `json:"provider"`
}

// ProbeIntelligence proves the configured provider can answer before Composer Refinar.
func (n *Nexus) ProbeIntelligence(ctx context.Context, projectID string) (IntelligenceProbeResult, error) {
	provider, err := n.ConfiguredIntelligenceProvider(ctx, projectID)
	if err != nil {
		return IntelligenceProbeResult{}, err
	}
	engine := intelligence.NewNexusEngine(provider)
	if err := engine.Probe(ctx); err != nil {
		name := ""
		if provider != nil {
			name = provider.Name()
		}
		return IntelligenceProbeResult{Provider: name}, err
	}
	return IntelligenceProbeResult{Provider: provider.Name()}, nil
}

// ConfiguredIntelligenceProvider resolves the explicitly configured intelligence source.
// Direct work never calls this function; Composer analysis/planning does.
func (n *Nexus) ConfiguredIntelligenceProvider(ctx context.Context, projectID string) (intelligence.IntelligenceProvider, error) {
	if n != nil && n.intelligenceProviderForTest != nil {
		return n.intelligenceProviderForTest, nil
	}
	cfg, err := n.GetIntelligenceConfig()
	if err != nil {
		return nil, err
	}
	switch cfg.Mode {
	case "", coreconfig.IntelligenceOff:
		return nil, intelligence.ErrIntelligenceUnavailable
	case coreconfig.IntelligenceOpenAICompatible:
		return intelligence.ProviderFromConfig(cfg)
	case coreconfig.IntelligenceCLI:
		return n.cliIntelligenceProvider(ctx, projectID, cfg)
	default:
		return nil, fmt.Errorf("unsupported intelligence mode %q", cfg.Mode)
	}
}

func (n *Nexus) cliIntelligenceProvider(ctx context.Context, projectID string, cfg coreconfig.IntelligenceConfig) (intelligence.IntelligenceProvider, error) {
	providerID := strings.ToLower(strings.TrimSpace(cfg.Provider))
	profileName := strings.TrimSpace(cfg.Profile)
	if providerID == "" || profileName == "" {
		return nil, fmt.Errorf("CLI intelligence requires provider and profile")
	}
	prof, err := profile.Get(providerID, profileName)
	if err != nil {
		return nil, fmt.Errorf("resolve intelligence profile: %w", err)
	}
	if prof.Disabled {
		return nil, fmt.Errorf("intelligence profile %s:%s is disabled", providerID, profileName)
	}
	d, err := n.controlDrivers().Get(providerID)
	if err != nil {
		return nil, err
	}
	detection, err := d.Detect(ctx)
	if err != nil {
		return nil, fmt.Errorf("detect intelligence CLI %s: %w", providerID, err)
	}
	if !detection.Installed {
		return nil, fmt.Errorf("intelligence CLI %s is not installed: %s", providerID, detection.Error)
	}
	caps := d.EffectiveCaps(ctx, prof)
	validated := caps.Headless.Status == driver.CapabilitySupported && caps.SubmitPrompt.Status == driver.CapabilitySupported
	if !validated {
		return nil, fmt.Errorf("provider %s cannot be used for Nexus Intelligence: headless=%s submit_prompt=%s", providerID, caps.Headless.Status, caps.SubmitPrompt.Status)
	}

	cwd := ""
	if projectID != "" {
		if st, openErr := n.OpenProject(); openErr == nil {
			if project, projectErr := st.GetProject(projectID); projectErr == nil {
				cwd = project.CanonicalPath
			}
		}
	}

	// Discover candidate profiles within the same provider family for automatic failover
	// when the primary profile experiences quota exhaustion or rate limits.
	candidates := []model.Profile{prof}
	seen := map[string]struct{}{
		strings.ToLower(prof.Name): {},
	}
	if allProfiles, listErr := profile.List(); listErr == nil {
		for _, p := range allProfiles {
			if !strings.EqualFold(p.Provider, providerID) || p.Disabled {
				continue
			}
			lowerName := strings.ToLower(p.Name)
			if _, already := seen[lowerName]; already {
				continue
			}
			pCaps := d.EffectiveCaps(ctx, p)
			if pCaps.Headless.Status == driver.CapabilitySupported && pCaps.SubmitPrompt.Status == driver.CapabilitySupported {
				candidates = append(candidates, p)
				seen[lowerName] = struct{}{}
			}
		}
	}

	// If the primary profile is known to have exhausted quota, prioritize an available candidate from the family.
	if len(candidates) > 1 && isProfileQuotaExhausted(providerID, candidates[0].Name) {
		for i := 1; i < len(candidates); i++ {
			if !isProfileQuotaExhausted(providerID, candidates[i].Name) {
				slog.Info("primary intelligence profile quota is exhausted; prioritizing available profile in family",
					"provider", providerID, "primary", candidates[0].Name, "fallback", candidates[i].Name)
				available := candidates[i]
				candidates = append([]model.Profile{available}, append(candidates[:i], candidates[i+1:]...)...)
				break
			}
		}
	}

	runner := func(runCtx context.Context, prompt string) (string, error) {
		args, err := intelligence.HeadlessPromptArgs(providerID, prompt)
		if err != nil {
			return "", err
		}
		args, err = driver.ApplyLaunchConfiguration(providerID, cfg.Model, nil, args)
		if err != nil {
			return "", err
		}
		var lastErr error
		for i, cand := range candidates {
			bin, builtArgs, env, err := d.BuildCommand(runCtx, cand, args)
			if err != nil {
				lastErr = fmt.Errorf("profile %s:%s build command: %w", providerID, cand.Name, err)
				continue
			}
			out, err := runtime.RunCommandCapture(runCtx, bin, builtArgs, env, cwd)
			if err == nil {
				trimmed := strings.TrimSpace(out)
				failure := classifier.Classify(nil, trimmed)
				if trimmed != "" && failure.Kind != model.FailureQuota && failure.Kind != model.FailureRateLimit {
					return out, nil
				}
				if trimmed == "" {
					lastErr = fmt.Errorf("profile %s:%s returned empty output", providerID, cand.Name)
				} else {
					lastErr = fmt.Errorf("profile %s:%s reported quota/limit: %s", providerID, cand.Name, trimmed)
				}
			} else {
				lastErr = fmt.Errorf("profile %s:%s error: %w", providerID, cand.Name, err)
			}
			slog.Warn("intelligence profile execution failed, attempting failover",
				"provider", providerID,
				"profile", cand.Name,
				"error", lastErr,
				"has_more_candidates", i+1 < len(candidates))
		}
		return "", lastErr
	}
	return intelligence.NewCLIProvider(providerID, profileName, true, runner), nil
}

func isProfileQuotaExhausted(provider, profileName string) bool {
	snap, found := quota.NewEngine(quota.LastKnownTTL).GetCachedUsage(provider, profileName)
	if !found {
		return false
	}
	for _, w := range snap.Windows {
		if (w.RemainingPercent != nil && *w.RemainingPercent <= 0) ||
			(w.UsedPercent != nil && *w.UsedPercent >= 100) {
			return true
		}
	}
	return false
}
