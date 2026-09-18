package nexus

import (
	"context"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
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
	runner := func(runCtx context.Context, prompt string) (string, error) {
		args, err := intelligence.HeadlessPromptArgs(providerID, prompt)
		if err != nil {
			return "", err
		}
		args, err = driver.ApplyLaunchConfiguration(providerID, cfg.Model, nil, args)
		if err != nil {
			return "", err
		}
		bin, builtArgs, env, err := d.BuildCommand(runCtx, prof, args)
		if err != nil {
			return "", err
		}
		return runtime.RunCommandCapture(runCtx, bin, builtArgs, env, cwd)
	}
	return intelligence.NewCLIProvider(providerID, profileName, true, runner), nil
}
