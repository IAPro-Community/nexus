package fallback

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"

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

// RunWithFallback executes runFn with automatic failover if rate-limited.
// Bound: at most one attempt per distinct provider candidate (derived from
// attempted/excluded), so A↔B and A→B→C→A ping-pong are impossible. Only
// RateLimit/Quota failures trigger failover; Auth/Network/Unknown/Provider/
// Command/User causes are returned immediately with their typed cause preserved.
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
	maxAttempts := len(pool)

	attempted := make(map[string]bool, len(pool))
	attemptOrder := make([]string, 0, len(pool))

	currentProfileName := initialProfile
	if currentProfileName == "" {
		res, err := e.selector.SelectBestProfile(ctx, provider, workspace, candidates, accounts, nil)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFallbackExhausted, err)
		}
		if res == nil || res.SelectedProfile == nil {
			return fmt.Errorf("%w for provider %s", ErrFallbackExhausted, provider)
		}
		currentProfileName = res.SelectedProfile.Name
	}

	var lastFailure model.Failure
	var lastErr error

	for len(attempted) < maxAttempts {
		if attempted[currentProfileName] {
			return fmt.Errorf("%w: %s cycle %v last cause %q: %w",
				ErrFallbackExhausted, provider, attemptOrder, lastFailure.Message, causeForKind(lastFailure.Kind))
		}
		attempted[currentProfileName] = true
		attemptOrder = append(attemptOrder, currentProfileName)

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
		lastFailure = failure
		lastErr = err

		// Non-retryable causes: preserve the typed cause, no cooldown, no fallback.
		if failure.Kind != model.FailureRateLimit && failure.Kind != model.FailureQuota {
			return typedTerminalError(failure, err)
		}
		if !allowFallback {
			if err != nil {
				return fmt.Errorf("%w: %w", causeForKind(failure.Kind), err)
			}
			return fmt.Errorf("%w: %s", causeForKind(failure.Kind), failure.Message)
		}
		// Single-resource bound: no alternative to try.
		if len(pool) == 1 {
			return fmt.Errorf("%s failed (%s) single resource exhausted: %w",
				currentProfileName, failure.Message, causeForKind(failure.Kind))
		}

		// Record rate limit / cooldown with provider precedence.
		currentScope := currentProfile.AccountScope
		if currentScope.Verifiable() {
			e.cooldown.RecordFailureForScope(currentScope, failure, failure.Message)
		} else {
			// Compatibility for in-memory callers that have not gone through
			// profile.List yet. Persisted profiles always carry a scope.
			e.cooldown.RecordFailure(provider, currentProfileName, failure, failure.Message)
		}

		// Collect already attempted profile names (sorted for determinism).
		excluded := make([]string, 0, len(attempted))
		for name := range attempted {
			excluded = append(excluded, name)
		}
		sort.Strings(excluded)

		// Select next best profile
		nextRes, nextErr := e.selector.SelectBestProfile(ctx, provider, workspace, candidates, accounts, excluded)
		if nextErr != nil {
			return fmt.Errorf("%s failed (%s) and no fallback profiles available: %w: %w",
				currentProfileName, failure.Message, nextErr, causeForKind(failure.Kind))
		}
		if nextRes == nil || nextRes.SelectedProfile == nil {
			return fmt.Errorf("%s failed (%s) and selector returned no profile: %w",
				currentProfileName, failure.Message, causeForKind(failure.Kind))
		}
		if attempted[nextRes.SelectedProfile.Name] {
			// Anti ping-pong: selector re-offered an attempted profile.
			return fmt.Errorf("%w: selector re-offered attempted %s cycle %v last cause %q: %w",
				ErrFallbackExhausted, nextRes.SelectedProfile.Name, attemptOrder, failure.Message, causeForKind(failure.Kind))
		}

		fmt.Fprintf(os.Stderr, "⚠ Profile %s:%s hit %s (%s). Automatically falling back to %s...\n", provider, currentProfileName, failure.Kind, failure.Message, nextRes.SelectedProfile.Name)
		currentProfileName = nextRes.SelectedProfile.Name
	}
	return fmt.Errorf("%w: %s cycle %v last cause %q err=%v: %w",
		ErrFallbackExhausted, provider, attemptOrder, lastFailure.Message, lastErr, causeForKind(lastFailure.Kind))
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
