package nexus

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/core/classifier"
	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
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

func intelligenceCLICapable(caps driver.EffectiveCapabilities) bool {
	return caps.Headless.IsSupportedAndTested() && caps.SubmitPrompt.IsSupportedAndTested()
}

func readOnlyReviewSpec(providerID string, caps driver.EffectiveCapabilities) (spec.Spec, bool) {
	if !caps.ReadOnlyReview.IsSupportedAndTested() {
		return spec.Spec{}, false
	}
	providerSpec, ok := spec.Get(strings.ToLower(strings.TrimSpace(providerID)))
	if !ok || providerSpec.ReviewIsolationLevel() != spec.IsolationReadOnlyNoNetwork || len(providerSpec.ReviewMode) == 0 {
		return spec.Spec{}, false
	}
	return providerSpec, true
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
	validated := intelligenceCLICapable(caps)
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
			if intelligenceCLICapable(pCaps) {
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
		return executeCLIProfileFailover(runCtx, d, providerID, cwd, candidates, args)
	}

	if providerSpec, ok := readOnlyReviewSpec(providerID, caps); ok {
		readOnlyRunner := func(runCtx context.Context, prompt string) (string, error) {
			args, err := readOnlyNexPromptArgs(providerID, cfg.Model, prompt, providerSpec)
			if err != nil {
				return "", err
			}
			return executeCLIProfileFailover(runCtx, d, providerID, cwd, candidates, args)
		}
		return intelligence.NewCLIProviderWithReadOnly(providerID, profileName, true, runner, readOnlyRunner), nil
	}
	return intelligence.NewCLIProvider(providerID, profileName, true, runner), nil
}

func readOnlyNexPromptArgs(providerID, modelName, prompt string, providerSpec spec.Spec) ([]string, error) {
	args, err := intelligence.HeadlessPromptArgs(providerID, prompt)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(strings.TrimSpace(providerID), "codex") && len(args) >= 2 && args[0] == "exec" {
		// Codex expects sandbox/approval flags before the exec subcommand and model flags before the positional prompt.
		out := append([]string(nil), providerSpec.ReviewMode...)
		out = append(out, "exec")
		if trimmedModel := strings.TrimSpace(modelName); trimmedModel != "" {
			out = append(out, "--model", trimmedModel)
		}
		out = append(out, args[1:]...)
		return out, nil
	}
	args, err = driver.ApplyLaunchConfiguration(providerID, modelName, nil, args)
	if err != nil {
		return nil, err
	}
	return append(args, providerSpec.ReviewMode...), nil
}

func executeCLIProfileFailover(
	runCtx context.Context,
	d driver.ControlDriver,
	providerID string,
	cwd string,
	candidates []model.Profile,
	args []string,
) (string, error) {
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

// NexChatSetupCandidate describes an inspected local CLI or endpoint candidate for Nex chat.
type NexChatSetupCandidate struct {
	Mode          coreconfig.IntelligenceMode `json:"mode"`
	Provider      string                      `json:"provider"`
	Profile       string                      `json:"profile,omitempty"`
	Model         string                      `json:"model,omitempty"`
	Ready         bool                        `json:"ready"`
	ReadOnly      bool                        `json:"read_only"`
	Installed     bool                        `json:"installed"`
	Authenticated bool                        `json:"authenticated"`
	QuotaOK       bool                        `json:"quota_ok"`
	Reason        string                      `json:"reason,omitempty"`
}

// NexChatSetupAssistance returns actionable zero-friction setup hints when Nex
// does not have a live chat model ready yet.
type NexChatSetupAssistance struct {
	Reason              string                  `json:"reason"`
	CanAutoConfigure    bool                    `json:"can_auto_configure"`
	RecommendedMode     string                  `json:"recommended_mode,omitempty"`
	RecommendedProvider string                  `json:"recommended_provider,omitempty"`
	RecommendedProfile  string                  `json:"recommended_profile,omitempty"`
	RecommendedAction   string                  `json:"recommended_action,omitempty"`
	Candidates          []NexChatSetupCandidate `json:"candidates,omitempty"`
}

// AutoConfigureIntelligenceRequest drives 1-click or inline Nex setup without secrets in the body.
type AutoConfigureIntelligenceRequest struct {
	Mode       coreconfig.IntelligenceMode `json:"mode,omitempty"`
	Provider   string                      `json:"provider,omitempty"`
	Profile    string                      `json:"profile,omitempty"`
	BaseURL    string                      `json:"base_url,omitempty"`
	Model      string                      `json:"model,omitempty"`
	APIKeyEnv  string                      `json:"api_key_env,omitempty"`
	APIKeyFile string                      `json:"api_key_file,omitempty"`
}

// ResolveNexChatProvider returns a configured OpenAI/read-only CLI provider,
// automatically falls back to a ready read-only CLI profile when possible, or
// returns structured setup assistance so the UI can guide/auto-fix configuration.
func (n *Nexus) ResolveNexChatProvider(ctx context.Context, projectID string) (intelligence.IntelligenceProvider, string, *NexChatSetupAssistance, error) {
	if n != nil && n.intelligenceProviderForTest != nil {
		return n.intelligenceProviderForTest, "configured_model", nil, nil
	}
	cfg, err := n.GetIntelligenceConfig()
	if err != nil {
		return nil, "", n.InspectNexSetupAssistance(ctx, "intelligence_config_unreadable"), err
	}

	switch cfg.Mode {
	case coreconfig.IntelligenceOpenAICompatible:
		provider, provErr := intelligence.ProviderFromConfig(cfg)
		if provErr == nil && provider != nil && provider.Available(ctx) {
			return provider, "configured_model", nil, nil
		}
		return nil, "", n.InspectNexSetupAssistance(ctx, "openai_config_invalid"), firstError(provErr, intelligence.ErrIntelligenceUnavailable)

	case coreconfig.IntelligenceCLI:
		provider, provErr := n.cliIntelligenceProvider(ctx, projectID, cfg)
		if provErr == nil && provider != nil {
			if cliProv, ok := provider.(*intelligence.CLIProvider); ok && cliProv.SupportsNexReadOnly() {
				return cliProv, "configured_model", nil, nil
			}
		}
		if fallback, found := n.findReadyNexCLIProvider(ctx, projectID, cfg.Model); found {
			return fallback, "auto_cli_readonly", nil, nil
		}
		reason := "cli_missing_readonly_sandbox"
		if provErr != nil {
			reason = "configured_cli_unavailable"
		}
		return nil, "", n.InspectNexSetupAssistance(ctx, reason), firstError(provErr, intelligence.ErrNexChatRequiresReadOnly)

	case "", coreconfig.IntelligenceOff:
		if autoProv, found := n.findReadyNexCLIProvider(ctx, projectID, cfg.Model); found {
			return autoProv, "auto_cli_readonly", nil, nil
		}
		return nil, "", n.InspectNexSetupAssistance(ctx, "intelligence_unconfigured"), intelligence.ErrIntelligenceUnavailable

	default:
		return nil, "", n.InspectNexSetupAssistance(ctx, "unsupported_intelligence_mode"), fmt.Errorf("unsupported intelligence mode %q", cfg.Mode)
	}
}

func firstError(primary, fallback error) error {
	if primary != nil {
		return primary
	}
	return fallback
}

func (n *Nexus) findReadyNexCLIProvider(ctx context.Context, projectID, modelName string) (intelligence.IntelligenceProvider, bool) {
	candidates := n.inspectNexCLICandidates(ctx)
	for _, cand := range candidates {
		if !cand.Ready || !cand.ReadOnly {
			continue
		}
		prov, err := n.cliIntelligenceProvider(ctx, projectID, coreconfig.IntelligenceConfig{
			Mode:     coreconfig.IntelligenceCLI,
			Provider: cand.Provider,
			Profile:  cand.Profile,
			Model:    strings.TrimSpace(modelName),
		})
		if err != nil {
			continue
		}
		if cliProv, ok := prov.(*intelligence.CLIProvider); ok && cliProv.SupportsNexReadOnly() {
			return cliProv, true
		}
	}
	return nil, false
}

// InspectNexSetupAssistance inspects local CLI profiles and returns safe,
// secret-free guidance explaining what can be auto-activated or configured.
func (n *Nexus) InspectNexSetupAssistance(ctx context.Context, reason string) *NexChatSetupAssistance {
	candidates := n.inspectNexCLICandidates(ctx)
	assistance := &NexChatSetupAssistance{
		Reason:            strings.TrimSpace(reason),
		RecommendedAction: "configure_openai_or_cli",
		Candidates:        candidates,
	}
	if assistance.Reason == "" {
		assistance.Reason = "intelligence_unconfigured"
	}
	for _, cand := range candidates {
		if cand.Ready && cand.ReadOnly {
			assistance.CanAutoConfigure = true
			assistance.RecommendedMode = string(coreconfig.IntelligenceCLI)
			assistance.RecommendedProvider = cand.Provider
			assistance.RecommendedProfile = cand.Profile
			assistance.RecommendedAction = "auto_enable_readonly_cli"
			return assistance
		}
	}
	for _, cand := range candidates {
		if cand.Installed && cand.ReadOnly && !cand.Authenticated {
			assistance.RecommendedMode = string(coreconfig.IntelligenceCLI)
			assistance.RecommendedProvider = cand.Provider
			assistance.RecommendedProfile = cand.Profile
			assistance.RecommendedAction = "authenticate_cli_profile"
			return assistance
		}
	}
	return assistance
}

// AutoConfigureNexIntelligence configures Nex intelligence either from the best
// ready read-only CLI profile ("AUTO"), a specific CLI profile, or an OpenAI-compatible URL.
func (n *Nexus) AutoConfigureNexIntelligence(ctx context.Context, req AutoConfigureIntelligenceRequest) (IntelligenceStatus, error) {
	mode := coreconfig.IntelligenceMode(strings.ToUpper(strings.TrimSpace(string(req.Mode))))
	switch mode {
	case "", "AUTO":
		candidates := n.inspectNexCLICandidates(ctx)
		for _, cand := range candidates {
			if cand.Ready && cand.ReadOnly {
				next := coreconfig.IntelligenceConfig{
					Mode:     coreconfig.IntelligenceCLI,
					Provider: cand.Provider,
					Profile:  cand.Profile,
					Model:    strings.TrimSpace(req.Model),
				}
				if err := n.SetIntelligenceConfig(next); err != nil {
					return IntelligenceStatus{}, err
				}
				return n.IntelligenceStatus(ctx, ""), nil
			}
		}
		return IntelligenceStatus{}, fmt.Errorf("no authenticated read-only CLI profile ready for auto-configuration")

	case coreconfig.IntelligenceCLI:
		providerID := strings.ToLower(strings.TrimSpace(req.Provider))
		profileName := strings.TrimSpace(req.Profile)
		if providerID != "" && profileName == "" {
			for _, cand := range n.inspectNexCLICandidates(ctx) {
				if cand.Provider == providerID && cand.Ready {
					profileName = cand.Profile
					break
				}
			}
		}
		next := coreconfig.IntelligenceConfig{
			Mode:     coreconfig.IntelligenceCLI,
			Provider: providerID,
			Profile:  profileName,
			Model:    strings.TrimSpace(req.Model),
		}
		if err := n.SetIntelligenceConfig(next); err != nil {
			return IntelligenceStatus{}, err
		}
		return n.IntelligenceStatus(ctx, ""), nil

	case coreconfig.IntelligenceOpenAICompatible:
		next := coreconfig.IntelligenceConfig{
			Mode:       coreconfig.IntelligenceOpenAICompatible,
			BaseURL:    strings.TrimSpace(req.BaseURL),
			Model:      strings.TrimSpace(req.Model),
			APIKeyEnv:  strings.TrimSpace(req.APIKeyEnv),
			APIKeyFile: strings.TrimSpace(req.APIKeyFile),
		}
		if _, err := intelligence.ProviderFromConfig(next); err != nil {
			return IntelligenceStatus{}, fmt.Errorf("openai-compatible intelligence requires valid base_url and model: %w", err)
		}
		if err := n.SetIntelligenceConfig(next); err != nil {
			return IntelligenceStatus{}, err
		}
		return n.IntelligenceStatus(ctx, ""), nil

	default:
		return IntelligenceStatus{}, fmt.Errorf("unsupported auto-configure mode %q", req.Mode)
	}
}

func (n *Nexus) inspectNexCLICandidates(ctx context.Context) []NexChatSetupCandidate {
	profiles, err := profile.List()
	if err != nil {
		return nil
	}
	detections := make(map[string]model.DetectionResult)
	drivers := n.controlDrivers()
	out := make([]NexChatSetupCandidate, 0, len(profiles))
	for _, prof := range profiles {
		if prof.Disabled {
			continue
		}
		providerID := strings.ToLower(strings.TrimSpace(prof.Provider))
		d, err := drivers.Get(providerID)
		if err != nil {
			continue
		}
		det, ok := detections[providerID]
		if !ok {
			det, _ = d.Detect(ctx)
			detections[providerID] = det
		}
		cand := NexChatSetupCandidate{
			Mode:      coreconfig.IntelligenceCLI,
			Provider:  providerID,
			Profile:   prof.Name,
			Installed: det.Installed,
		}
		if !det.Installed {
			cand.Reason = "cli_not_installed"
			out = append(out, cand)
			continue
		}
		caps := d.EffectiveCaps(ctx, prof)
		if !intelligenceCLICapable(caps) {
			cand.Reason = "headless_unsupported"
			out = append(out, cand)
			continue
		}
		_, cand.ReadOnly = readOnlyReviewSpec(providerID, caps)
		cand.Authenticated = profile.GetAccountInfo(providerID, prof.Name).Authenticated
		cand.QuotaOK = !isProfileQuotaExhausted(providerID, prof.Name)
		cand.Ready = cand.Installed && cand.ReadOnly && cand.Authenticated && cand.QuotaOK
		switch {
		case !cand.ReadOnly:
			cand.Reason = "missing_readonly_sandbox"
		case !cand.Authenticated:
			cand.Reason = "not_authenticated"
		case !cand.QuotaOK:
			cand.Reason = "quota_exhausted"
		default:
			cand.Reason = "ready"
		}
		out = append(out, cand)
	}
	sort.Slice(out, func(i, j int) bool {
		if (out[i].Ready && out[i].ReadOnly) != (out[j].Ready && out[j].ReadOnly) {
			return out[i].Ready && out[i].ReadOnly
		}
		if (out[i].Provider == "codex") != (out[j].Provider == "codex") {
			return out[i].Provider == "codex"
		}
		if out[i].Provider != out[j].Provider {
			return out[i].Provider < out[j].Provider
		}
		return out[i].Profile < out[j].Profile
	})
	return out
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
