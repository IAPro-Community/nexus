package fallback

import (
	"context"
	"errors"
	"fmt"

	"github.com/IAPro-Community/nexus/internal/core/cooldown"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/scheduler"
)

// Typed fallback causes — never report everything as rate limit. Callers can
// use errors.Is to distinguish the terminal cause.
var (
	ErrFallbackExhausted       = errors.New("fallback exhausted: no usable profiles left in this execution cycle")
	ErrFallbackRateLimited     = errors.New("fallback rate limited")
	ErrFallbackQuotaExhausted  = errors.New("fallback quota exhausted")
	ErrFallbackUnauthenticated = errors.New("fallback unauthenticated")
	ErrFallbackUnknownQuota    = errors.New("fallback unknown quota")
	ErrFallbackNetwork         = errors.New("fallback network failure")
	ErrFallbackNoCandidates    = errors.New("fallback has no candidate profiles")
	ErrFallbackUnsafeReplay    = errors.New("automatic retry refused because provider execution outcome is ambiguous")
)

// Executor manages execution with automatic failover between accounts on rate-limit/quota errors.
type Executor struct {
	selector *scheduler.Selector
	cooldown *cooldown.Tracker
}

// NewExecutor creates a new Fallback Executor.
func NewExecutor(s *scheduler.Selector, cd *cooldown.Tracker) *Executor {
	if cd == nil {
		cd = cooldown.NewTracker()
	}
	return &Executor{
		selector: s,
		cooldown: cd,
	}
}

// RunWithFallback retains the historical call shape but does not replay a
// provider operation after dispatch. Quota/rate-limit output does not prove
// the provider produced no output or external effects. Initial account
// selection still uses the selector's quota and identity evidence.
func (e *Executor) RunWithFallback(
	ctx context.Context,
	provider string,
	workspace string,
	initialProfile string,
	candidates []model.Profile,
	accounts map[string]model.AccountInfo,
	allowFallback bool,
	runFn func(p model.Profile) (model.Failure, error),
) error {
	return e.RunWithFallbackStrategy(ctx, provider, workspace, initialProfile, candidates, accounts, allowFallback, false, runFn)
}

// RunWithFallbackStrategy optionally balances initial account selection.
// allowFallback is retained for compatibility: opted-in callers get a typed
// unsafe-replay error after a quota/rate-limit result, never a second dispatch.
func (e *Executor) RunWithFallbackStrategy(
	ctx context.Context,
	provider string,
	workspace string,
	initialProfile string,
	candidates []model.Profile,
	accounts map[string]model.AccountInfo,
	allowFallback bool,
	balanced bool,
	runFn func(p model.Profile) (model.Failure, error),
) error {
	pool := make([]model.Profile, 0, len(candidates))
	seen := make(map[string]bool)
	for _, p := range candidates {
		if p.Provider != provider {
			continue
		}
		if seen[p.Name] {
			continue
		}
		seen[p.Name] = true
		pool = append(pool, p)
	}
	if len(pool) == 0 {
		return fmt.Errorf("%w for provider %s", ErrFallbackNoCandidates, provider)
	}
	currentProfileName := initialProfile
	if currentProfileName == "" {
		res, err := e.selectProfile(ctx, provider, workspace, candidates, accounts, nil, balanced)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFallbackExhausted, err)
		}
		if res == nil || res.SelectedProfile == nil {
			return fmt.Errorf("%w for provider %s", ErrFallbackExhausted, provider)
		}
		currentProfileName = res.SelectedProfile.Name
	}

	var currentProfile model.Profile
	found := false
	for _, p := range pool {
		if p.Name == currentProfileName {
			currentProfile = p
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("profile %s:%s does not exist", provider, currentProfileName)
	}

	failure, err := runFn(currentProfile)
	if err == nil && (failure.Kind == model.FailureNone || failure.Kind == "") {
		return nil
	}
	if failure.Kind == model.FailureRateLimit || failure.Kind == model.FailureQuota {
		if currentProfile.AccountScope.Verifiable() {
			e.cooldown.RecordFailureForScope(currentProfile.AccountScope, failure, failure.Message)
		} else {
			e.cooldown.RecordFailure(provider, currentProfileName, failure, failure.Message)
		}
		if allowFallback {
			return fmt.Errorf("%w: %w", ErrFallbackUnsafeReplay, typedTerminalError(failure, err))
		}
	}
	return typedTerminalError(failure, err)
}

func (e *Executor) selectProfile(ctx context.Context, provider, workspace string, candidates []model.Profile, accounts map[string]model.AccountInfo, excluded []string, balanced bool) (*scheduler.SelectionResult, error) {
	if balanced {
		return e.selector.SelectNextBalancedProfile(ctx, provider, workspace, candidates, accounts, excluded)
	}
	return e.selector.SelectBestProfile(ctx, provider, workspace, candidates, accounts, excluded)
}

func causeForKind(kind model.FailureKind) error {
	switch kind {
	case model.FailureQuota:
		return ErrFallbackQuotaExhausted
	case model.FailureRateLimit:
		return ErrFallbackRateLimited
	case model.FailureAuth:
		return ErrFallbackUnauthenticated
	case model.FailureNetwork:
		return ErrFallbackNetwork
	case model.FailureUnknown:
		return ErrFallbackUnknownQuota
	default:
		return ErrFallbackExhausted
	}
}

func typedTerminalError(failure model.Failure, runErr error) error {
	cause := causeForKind(failure.Kind)
	// Preserve the provider message alongside the typed cause.
	if runErr != nil {
		// Keep both the typed cause and the underlying execution error.
		return fmt.Errorf("%w: %s: %w", cause, failure.Message, runErr)
	}
	if failure.Message != "" {
		return fmt.Errorf("%w: %s", cause, failure.Message)
	}
	return cause
}
