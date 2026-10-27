package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
	"github.com/IAPro-Community/nexus/internal/core/scheduler"
	"github.com/IAPro-Community/nexus/internal/profile"
)

type runtimeLauncher interface {
	Launch(context.Context, launcher.LaunchOptions) (*registry.RuntimeSession, error)
}

type profileSelector interface {
	SelectBestProfile(ctx context.Context, provider string, workspace string, candidates []model.Profile, accounts map[string]model.AccountInfo, excludeProfiles []string) (*scheduler.SelectionResult, error)
}

func resolveLaunchProfile(ctx context.Context, provider, explicit, workspace, configuredDefault string, candidates []model.Profile, accounts map[string]model.AccountInfo, selector profileSelector, excluded []string, stderr io.Writer) (string, error) {
	if stderr == nil {
		stderr = os.Stderr
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
	selected := profile.ResolveDefault(provider, explicit, names, configuredDefault, func() (string, error) {
		if selector == nil {
			selectionErr = fmt.Errorf("profile selector unavailable")
			return "", selectionErr
		}
		remaining := remainingProfiles(candidates, excluded)
		result, err := selector.SelectBestProfile(ctx, provider, workspace, remaining, accounts, nil)
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
	// Automatic routing must fail closed when accounts are exhausted or unavailable.
	// However, if the selector rejected candidates solely because quota evidence was
	// stale or unverified, treat the exception by refreshing remaining authenticated
	// candidates in real time.
	if strings.TrimSpace(explicit) == "" && selectionErr != nil {
		if isStaleOrUnknownQuotaError(selectionErr) {
			remaining := remainingProfiles(candidates, excluded)
			refreshedAny := false
			for _, cand := range remaining {
				acc, ok := accounts[cand.Name]
				if !ok || !acc.Authenticated {
					continue
				}
				fmt.Fprintf(stderr, "⚡ [nexus] quota desatualizada em %s:%s; consultando atualização oficial…\n", provider, cand.Name)
				_ = profile.RefreshUsageSnapshotContext(ctx, provider, cand.Name)
				acc.Usage = profile.GetSanitizedUsageSnapshot(provider, cand.Name, acc.Email)
				accounts[cand.Name] = acc
				refreshedAny = true
			}
			if refreshedAny && selector != nil {
				result, err := selector.SelectBestProfile(ctx, provider, workspace, remaining, accounts, nil)
				if err == nil && result != nil && result.SelectedProfile != nil {
					return result.SelectedProfile.Name, nil
				}
			}
			// If live probe could not update (offline/timeout/lock), but healthy authenticated
			// profiles exist that are not rate-limited, fall back safely.
			if fallback := findHealthyFallbackProfile(names, configuredDefault, accounts); fallback != "" {
				fmt.Fprintf(stderr, "⚠ [nexus] quotas em tempo real indisponíveis; iniciando com %s:%s (fallback resiliente)\n", provider, fallback)
				return fallback, nil
			}
		}
		return "", selectionErr
	}
	if !containsProfileName(names, selected) {
		return "", fmt.Errorf("profile %q is not configured for provider %s", selected, provider)
	}
	return selected, nil
}

func isStaleOrUnknownQuotaError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "desatualizada") ||
		strings.Contains(msg, "atualize antes de selecionar automaticamente") ||
		(strings.Contains(msg, "UNKNOWN") && strings.Contains(msg, "inelegível para alocação automática"))
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

func launchWithBusyFailover(
	ctx context.Context,
	l runtimeLauncher,
	selector profileSelector,
	provider, explicit, workspace, configuredDefault string,
	candidates []model.Profile,
	accounts map[string]model.AccountInfo,
	base launcher.LaunchOptions,
	stderr io.Writer,
) (*registry.RuntimeSession, string, error) {
	if stderr == nil {
		stderr = os.Stderr
	}
	allowFailover := strings.TrimSpace(explicit) == ""
	var excluded []string
	var lastBusy string
	attempted := map[string]bool{}

	for {
		selected, err := resolveLaunchProfile(ctx, provider, explicit, workspace, configuredDefault, candidates, accounts, selector, excluded, stderr)
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
