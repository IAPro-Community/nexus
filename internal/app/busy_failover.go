package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	sdkproviders "github.com/IAPro-Community/IAPro-SDK/providers"
	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/core/scheduler"
	"github.com/IAPro-Community/nexus/internal/profile"
)

type runtimeLauncher interface {
	Launch(context.Context, launcher.LaunchOptions) (*registry.RuntimeSession, error)
}

type profileSelector interface {
	SelectBestProfile(ctx context.Context, provider string, workspace string, candidates []model.Profile, accounts map[string]model.AccountInfo, excludeProfiles []string) (*scheduler.SelectionResult, error)
}

type balancedProfileSelector interface {
	SelectNextBalancedProfile(ctx context.Context, provider string, workspace string, candidates []model.Profile, accounts map[string]model.AccountInfo, excludeProfiles []string) (*scheduler.SelectionResult, error)
}

func resolveLaunchProfile(ctx context.Context, provider, explicit, workspace, configuredDefault string, candidates []model.Profile, accounts map[string]model.AccountInfo, selector profileSelector, excluded []string, stderr io.Writer, balanced ...bool) (string, error) {
	return resolveLaunchProfileTracked(ctx, provider, explicit, workspace, configuredDefault, candidates, accounts, selector, excluded, stderr, nil, balanced...)
}

func resolveLaunchProfileTracked(ctx context.Context, provider, explicit, workspace, configuredDefault string, candidates []model.Profile, accounts map[string]model.AccountInfo, selector profileSelector, excluded []string, stderr io.Writer, attemptedRefresh map[string]bool, balanced ...bool) (string, error) {
	if stderr == nil {
		stderr = os.Stderr
	}
	if attemptedRefresh == nil {
		attemptedRefresh = map[string]bool{}
	}
	names := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if profileNameExcluded(candidate.Name, excluded) {
			continue
		}
		names = append(names, candidate.Name)
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no usable %s profiles remaining", provider)
	}
	var selectionErr error
	var lastResult *scheduler.SelectionResult
	isBalanced := len(balanced) > 0 && balanced[0]
	selected := profile.ResolveDefault(provider, explicit, names, configuredDefault, func() (string, error) {
		if selector == nil {
			selectionErr = fmt.Errorf("profile selector unavailable")
			return "", selectionErr
		}
		remaining := remainingProfiles(candidates, excluded)
		result, err := selectLaunchProfile(ctx, selector, provider, workspace, remaining, accounts, nil, isBalanced)
		lastResult = result
		if err != nil {
			selectionErr = err
			return "", err
		}
		if result == nil || result.SelectedProfile == nil {
			selectionErr = fmt.Errorf("profile selector returned no usable %s profile", provider)
			return "", selectionErr
		}
		return result.SelectedProfile.Name, nil
	})
	// Refresh any stale/unknown authenticated candidates in parallel when automatic
	// selection ran (even if another candidate had fresh cache, so :balanced never
	// picks a wrong leader because a sibling's cache expired or rolled over).
	if strings.TrimSpace(explicit) == "" && selector != nil && lastResult != nil && len(lastResult.NeedsRefresh) > 0 {
		if refreshScopesInParallelTracked(ctx, provider, lastResult.NeedsRefresh, candidates, accounts, stderr, attemptedRefresh) {
			remaining := remainingProfiles(candidates, excluded)
			if result, err := selectLaunchProfile(ctx, selector, provider, workspace, remaining, accounts, nil, isBalanced); err == nil && result != nil && result.SelectedProfile != nil {
				lastResult = result
				selectionErr = nil
				selected = result.SelectedProfile.Name
			} else if err != nil {
				selectionErr = err
				lastResult = result
			}
		}
	}
	if strings.TrimSpace(explicit) == "" && selectionErr != nil {
		if isStaleOrUnknownQuotaError(selectionErr) {
			remaining := remainingProfiles(candidates, excluded)
			if refreshLaunchCandidatesTracked(ctx, provider, explicit, remaining, accounts, stderr, attemptedRefresh) && selector != nil {
				result, err := selectLaunchProfile(ctx, selector, provider, workspace, remaining, accounts, nil, isBalanced)
				if err == nil && result != nil && result.SelectedProfile != nil {
					if result.UsedAuxiliaryFallback {
						fmt.Fprintf(stderr, "⚠ [nexus] %s: todas as quotas principais chegaram a 0%%; usando fallback auxiliar (%s)\n", provider, result.SelectedProfile.Name)
					}
					return result.SelectedProfile.Name, nil
				}
			}
			if fallback := findHealthyFallbackProfile(names, configuredDefault, accounts); fallback != "" {
				if launchProfileOverrideRequested() {
					fmt.Fprintf(stderr, "⚠ [nexus] quotas em tempo real indisponíveis; iniciando com %s:%s por decisão explícita (fallback resiliente)\n", provider, fallback)
					return fallback, nil
				}
				fmt.Fprintf(stderr, "⚠ [nexus] quotas em tempo real indisponíveis; seleção automática recusada (fail-closed).\n")
				fmt.Fprintf(stderr, "  → use %s %s:<perfil> para escolher uma conta explicitamente, ou %s %s --accept-unverified-quota para aceitar o fallback %s.\n", progName(), provider, progName(), provider, fallback)
				return "", fmt.Errorf("quota não verificável para %s; seleção automática recusada (fail-closed)", provider)
			}
		}
		return "", selectionErr
	}
	if !containsProfileName(names, selected) {
		return "", fmt.Errorf("profile %q is not configured for provider %s", selected, provider)
	}
	if lastResult != nil && lastResult.UsedAuxiliaryFallback {
		fmt.Fprintf(stderr, "⚠ [nexus] %s: todas as quotas principais chegaram a 0%%; usando fallback auxiliar (%s)\n", provider, selected)
	}
	return selected, nil
}

func selectLaunchProfile(ctx context.Context, selector profileSelector, provider, workspace string, candidates []model.Profile, accounts map[string]model.AccountInfo, excluded []string, balanced bool) (*scheduler.SelectionResult, error) {
	if balanced {
		balancedSelector, ok := selector.(balancedProfileSelector)
		if !ok {
			return nil, fmt.Errorf("balanced profile selection is unavailable")
		}
		return balancedSelector.SelectNextBalancedProfile(ctx, provider, workspace, candidates, accounts, excluded)
	}
	return selector.SelectBestProfile(ctx, provider, workspace, candidates, accounts, excluded)
}

func isStaleOrUnknownQuotaError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "desatualizada") ||
		strings.Contains(lower, "stale") ||
		strings.Contains(lower, "unknown") ||
		strings.Contains(lower, "needs refresh") ||
		strings.Contains(lower, "5h quota") ||
		strings.Contains(lower, "atualize antes de selecionar automaticamente") ||
		strings.Contains(lower, "inelegível para alocação automática")
}

func refreshLaunchCandidates(ctx context.Context, providerName, explicitProfile string, candidates []model.Profile, accounts map[string]model.AccountInfo, stderr io.Writer) bool {
	return refreshLaunchCandidatesTracked(ctx, providerName, explicitProfile, candidates, accounts, stderr, nil)
}

func refreshLaunchCandidatesTracked(ctx context.Context, providerName, explicitProfile string, candidates []model.Profile, accounts map[string]model.AccountInfo, stderr io.Writer, attempted map[string]bool) bool {
	if strings.TrimSpace(explicitProfile) != "" {
		return false
	}
	if s, ok := spec.Get(providerName); ok && s.Quota == spec.QuotaUnsupported {
		return false
	}
	now := time.Now()
	var toProbe []model.Profile
	for _, p := range candidates {
		if attempted != nil && attempted[p.Name] {
			continue
		}
		acc, ok := accounts[p.Name]
		if !ok || !acc.Authenticated {
			continue
		}
		if acc.Usage.Status == model.UsageError && strings.EqualFold(strings.TrimSpace(acc.Usage.DiagnosticCode), "AUTH_REQUIRED") {
			continue
		}
		qv := quota.BuildQuotaView(acc.Usage, acc.Email, acc.Plan)
		proj := qv.ProjectionAt(now, quota.DefaultTTL, acc.Authenticated, string(acc.Health))
		if proj.TruthState == quota.TruthStale || proj.TruthState == quota.TruthUnknown || proj.NeedsResetReprobe {
			toProbe = append(toProbe, p)
			if attempted != nil {
				attempted[p.Name] = true
			}
		}
	}
	if len(toProbe) == 0 {
		return false
	}
	if stderr != nil {
		fmt.Fprintf(stderr, "⚡ [nexus] atualizando %d conta(s) de %s em paralelo…\n", len(toProbe), providerName)
	}
	outcomes := profile.RefreshUsageBatch(ctx, toProbe, sdkproviders.BatchOptions{
		MaxConcurrency:  4,
		PerProbeTimeout: 10 * time.Second,
		GlobalTimeout:   15 * time.Second,
	})
	refreshed := false
	for _, out := range outcomes {
		name := out.Account.Scope.ProfileID
		acc, ok := accounts[name]
		if !ok {
			continue
		}
		acc.Usage = profile.GetSanitizedUsageSnapshot(providerName, name, acc.Email)
		accounts[name] = acc
		refreshed = true
	}
	return refreshed
}

func refreshScopesInParallelTracked(ctx context.Context, providerName string, scopes []model.AccountScope, candidates []model.Profile, accounts map[string]model.AccountInfo, stderr io.Writer, attempted map[string]bool) bool {
	wanted := make(map[string]bool, len(scopes))
	for _, sc := range scopes {
		if sc.ProfileID != "" && (attempted == nil || !attempted[sc.ProfileID]) {
			wanted[sc.ProfileID] = true
		}
	}
	var toProbe []model.Profile
	for _, p := range candidates {
		if wanted[p.Name] {
			if acc, ok := accounts[p.Name]; ok && acc.Authenticated {
				if acc.Usage.Status == model.UsageError && strings.EqualFold(strings.TrimSpace(acc.Usage.DiagnosticCode), "AUTH_REQUIRED") {
					continue
				}
				toProbe = append(toProbe, p)
				if attempted != nil {
					attempted[p.Name] = true
				}
			}
		}
	}
	if len(toProbe) == 0 {
		return false
	}
	if stderr != nil {
		fmt.Fprintf(stderr, "⚡ [nexus] revalidando %d conta(s) %s desatualizada(s) em paralelo…\n", len(toProbe), providerName)
	}
	outcomes := profile.RefreshUsageBatch(ctx, toProbe, sdkproviders.BatchOptions{
		MaxConcurrency:  4,
		PerProbeTimeout: 10 * time.Second,
		GlobalTimeout:   15 * time.Second,
	})
	refreshed := false
	for _, out := range outcomes {
		name := out.Account.Scope.ProfileID
		if acc, ok := accounts[name]; ok {
			acc.Usage = profile.GetSanitizedUsageSnapshot(providerName, name, acc.Email)
			accounts[name] = acc
			refreshed = true
		}
	}
	return refreshed
}

func findHealthyFallbackProfile(names []string, configuredDefault string, accounts map[string]model.AccountInfo) string {
	if configuredDefault != "" && containsProfileName(names, configuredDefault) {
		if acc, ok := accounts[configuredDefault]; ok && isAccountHealthyForFallback(acc) {
			return configuredDefault
		}
	}
	for _, name := range names {
		if acc, ok := accounts[name]; ok && isAccountHealthyForFallback(acc) {
			return name
		}
	}
	return ""
}

func isAccountHealthyForFallback(acc model.AccountInfo) bool {
	if !acc.Authenticated {
		return false
	}
	if acc.Health != model.HealthHealthy {
		return false
	}
	if acc.Usage.Status == model.UsageRateLimited {
		return false
	}
	return true
}

// launchProfileOverrideRequested reports whether the user explicitly accepted
// launching without verified quota evidence. The flag is opt-in per launch:
// automatic selection stays fail-closed by default.
func launchProfileOverrideRequested() bool {
	if os.Getenv("NEXUS_ACCEPT_UNVERIFIED_QUOTA") == "1" {
		return true
	}
	for _, arg := range os.Args[1:] {
		if arg == "--accept-unverified-quota" {
			return true
		}
	}
	return false
}

func launchWithBusyFailover(
	ctx context.Context,
	l runtimeLauncher,
	selector profileSelector,
	provider, explicit, workspace, configuredDefault string,
	candidates []model.Profile,
	accounts map[string]model.AccountInfo,
	base launcher.LaunchOptions,
	stderr io.Writer,
	automaticFallback bool,
	balanced ...bool,
) (*registry.RuntimeSession, string, error) {
	return launchWithBusyFailoverTracked(ctx, l, selector, provider, explicit, workspace, configuredDefault, candidates, accounts, base, stderr, automaticFallback, nil, balanced...)
}

func launchWithBusyFailoverTracked(
	ctx context.Context,
	l runtimeLauncher,
	selector profileSelector,
	provider, explicit, workspace, configuredDefault string,
	candidates []model.Profile,
	accounts map[string]model.AccountInfo,
	base launcher.LaunchOptions,
	stderr io.Writer,
	automaticFallback bool,
	attemptedRefresh map[string]bool,
	balanced ...bool,
) (*registry.RuntimeSession, string, error) {
	if stderr == nil {
		stderr = os.Stderr
	}
	if attemptedRefresh == nil {
		attemptedRefresh = map[string]bool{}
	}
	allowFailover := automaticFallback && strings.TrimSpace(explicit) == ""
	var excluded []string
	var lastBusy string
	attempted := map[string]bool{}

	for {
		selected, err := resolveLaunchProfileTracked(ctx, provider, explicit, workspace, configuredDefault, candidates, accounts, selector, excluded, stderr, attemptedRefresh, balanced...)
		if err != nil {
			if lastBusy != "" {
				return nil, lastBusy, fmt.Errorf("%s profile %s is busy and no fallback profiles are available", provider, lastBusy)
			}
			return nil, "", err
		}
		if attempted[selected] {
			return nil, selected, fmt.Errorf("%s profile %s is busy and no fallback profiles are available", provider, selected)
		}
		attempted[selected] = true

		opts := base
		opts.ProviderID = provider
		opts.ProfileID = selected
		if strings.TrimSpace(opts.Workspace) == "" {
			opts.Workspace = workspace
		}
		session, err := l.Launch(ctx, opts)
		if err == nil {
			return session, selected, nil
		}
		if !errors.Is(err, codex.ErrTUIBusy) {
			return nil, selected, err
		}
		if !allowFailover {
			return nil, selected, fmt.Errorf("%w", err)
		}
		lastBusy = selected
		excluded = append(excluded, selected)
		fmt.Fprintf(stderr, "⚠ Profile %s:%s is busy. Automatically falling back...\n", provider, selected)
	}
}

func busyProfileError(provider, profileName string) error {
	return fmt.Errorf("%s profile %s is busy (profile lock held by another session); stop the existing session or retry shortly", provider, profileName)
}

func remainingProfiles(candidates []model.Profile, excluded []string) []model.Profile {
	out := make([]model.Profile, 0, len(candidates))
	for _, candidate := range candidates {
		if profileNameExcluded(candidate.Name, excluded) {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

func profileNameExcluded(name string, excluded []string) bool {
	for _, item := range excluded {
		if item == name {
			return true
		}
	}
	return false
}

func containsProfileName(names []string, wanted string) bool {
	for _, name := range names {
		if name == wanted {
			return true
		}
	}
	return false
}
