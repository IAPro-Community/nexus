package fallback

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/cooldown"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/scheduler"
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
	attempted := make(map[string]bool)

	currentProfileName := initialProfile
	if currentProfileName == "" {
		res, err := e.selector.SelectBestProfile(ctx, provider, workspace, candidates, accounts, nil)
		if err != nil {
			return err
		}
		currentProfileName = res.SelectedProfile.Name
	}

	for {
		if attempted[currentProfileName] {
			return fmt.Errorf("all usable %s profiles exhausted in this execution cycle", provider)
		}
		attempted[currentProfileName] = true

		var currentProfile model.Profile
		found := false
		for _, p := range candidates {
			if p.Provider == provider && p.Name == currentProfileName {
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

		// If fallback is not allowed or failure is not recoverable via switching accounts
		if !allowFallback || (failure.Kind != model.FailureRateLimit && failure.Kind != model.FailureQuota) {
			if err != nil {
				return err
			}
			return fmt.Errorf("%s", failure.Message)
		}

		// Record rate limit / cooldown
		retryDur := 15 * time.Minute
		if failure.RetryAfter != nil && *failure.RetryAfter > 0 {
			retryDur = *failure.RetryAfter
		}
		currentScope := currentProfile.AccountScope
		if currentScope.Verifiable() {
			e.cooldown.RecordRateLimitForScope(currentScope, retryDur, failure.ResetAt, failure.Message)
		} else {
			// Compatibility for in-memory callers that have not gone through
			// profile.List yet. Persisted profiles always carry a scope.
			e.cooldown.RecordRateLimit(provider, currentProfileName, retryDur, failure.ResetAt, failure.Message)
		}

		// Collect already attempted profile names
		var excluded []string
		for name := range attempted {
			excluded = append(excluded, name)
		}

		// Select next best profile
		nextRes, nextErr := e.selector.SelectBestProfile(ctx, provider, workspace, candidates, accounts, excluded)
		if nextErr != nil {
			return fmt.Errorf("%s failed (%s) and no fallback profiles available: %w", currentProfileName, failure.Message, nextErr)
		}

		fmt.Fprintf(os.Stderr, "⚠ Profile %s:%s hit rate-limit (%s). Automatically falling back to %s...\n", provider, currentProfileName, failure.Message, nextRes.SelectedProfile.Name)
		currentProfileName = nextRes.SelectedProfile.Name
	}
}
