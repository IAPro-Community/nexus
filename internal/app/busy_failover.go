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

func resolveLaunchProfile(provider, explicit, workspace, configuredDefault string, candidates []model.Profile, accounts map[string]model.AccountInfo, selector profileSelector, excluded []string) (string, error) {
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
	selected := profile.ResolveDefault(provider, explicit, names, configuredDefault, func() (string, error) {
		if selector == nil {
			return "", fmt.Errorf("profile selector unavailable")
		}
		result, err := selector.SelectBestProfile(context.Background(), provider, workspace, candidates, accounts, excluded)
		if err == nil && result != nil && result.SelectedProfile != nil {
			return result.SelectedProfile.Name, nil
		}
		return "", err
	})
	if !containsProfileName(names, selected) {
		return "", fmt.Errorf("profile %q is not configured for provider %s", selected, provider)
	}
	return selected, nil
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
		selected, err := resolveLaunchProfile(provider, explicit, workspace, configuredDefault, candidates, accounts, selector, excluded)
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
