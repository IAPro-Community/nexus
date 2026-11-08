package fallback

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/cooldown"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/core/scheduler"
)

func TestFallbackExecution(t *testing.T) {
	cfgDir := t.TempDir()
	dataDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_STATE_DIR", stateDir)

	cfg := config.NewDefaultConfig()
	qEng := quota.NewEngine(5 * time.Minute)
	cd := cooldown.NewTracker()
	sel := scheduler.NewSelector(cfg, qEng, cd)
	exec := NewExecutor(sel, cd)

	candidates := []model.Profile{
		{Provider: "codex", Name: "acc-1"},
		{Provider: "codex", Name: "acc-2"},
	}
	accounts := map[string]model.AccountInfo{
		"acc-1": {Authenticated: true, Health: model.HealthHealthy},
		"acc-2": {Authenticated: true, Health: model.HealthHealthy},
	}

	ctx := context.Background()
	attempts := 0
	lastProfileUsed := ""

	// Test 1: acc-1 fails with RateLimit -> automatically retries with acc-2
	err := exec.RunWithFallback(ctx, "codex", "/tmp", "acc-1", candidates, accounts, true, func(p model.Profile) (model.Failure, error) {
		attempts++
		lastProfileUsed = p.Name
		if p.Name == "acc-1" {
			return model.Failure{
				Kind:    model.FailureRateLimit,
				Message: "HTTP 429 Too Many Requests",
			}, errors.New("exit status 1")
		}
		return model.Failure{Kind: model.FailureNone}, nil
	})

	if err == nil {
		t.Fatal("UNKNOWN quota fallback must fail closed instead of selecting an unverified resource")
	}
	if attempts != 1 {
		t.Fatalf("expected only the explicitly selected attempt, got %d", attempts)
	}
	if lastProfileUsed != "acc-1" {
		t.Fatalf("expected acc-1 to be the only attempted profile, got %s", lastProfileUsed)
	}

	// Verify cooldown was registered for acc-1
	limited, _ := cd.IsRateLimited("codex", "acc-1")
	if !limited {
		t.Fatalf("expected acc-1 to be recorded in cooldown")
	}

	// Test 2: Both fail with RateLimit -> clean error without infinite loop
	attempts = 0
	err2 := exec.RunWithFallback(ctx, "codex", "/tmp", "acc-1", candidates, accounts, true, func(p model.Profile) (model.Failure, error) {
		attempts++
		return model.Failure{
			Kind:    model.FailureRateLimit,
			Message: "HTTP 429",
		}, errors.New("exit 1")
	})

	if err2 == nil {
		t.Fatalf("expected error when all candidates fail")
	}
	if !strings.Contains(err2.Error(), "no fallback profiles available") && !strings.Contains(err2.Error(), "exhausted") && !strings.Contains(err2.Error(), "no usable codex profiles") {
		t.Fatalf("unexpected error message: %v", err2)
	}
}

func TestRunWithFallbackStrategyRotatesAfterQuotaFailure(t *testing.T) {
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv("AI_CLI_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	cd := cooldown.NewTracker()
	executor := NewExecutor(scheduler.NewSelector(config.NewDefaultConfig(), quota.NewEngine(time.Minute), cd), cd)
	candidates := []model.Profile{{Provider: "agy", Name: "account-a"}, {Provider: "agy", Name: "account-b"}}
	accounts := map[string]model.AccountInfo{}
	for _, candidate := range candidates {
		remaining := 80.0
		accounts[candidate.Name] = model.AccountInfo{
			Authenticated: true,
			Health:        model.HealthHealthy,
			Usage: model.UsageSnapshot{
				ProviderID: "agy",
				ProfileID:  candidate.Name,
				Status:     model.UsageLive,
				Source:     model.SourceCLIOutput,
				FetchedAt:  time.Now(),
				Windows: []model.UsageWindow{
					{Kind: "5h", Group: "gemini", RemainingPercent: &remaining},
				},
			},
		}
	}
	var attempted []string
	err := executor.RunWithFallbackStrategy(context.Background(), "agy", "/tmp", "", candidates, accounts, true, true, func(profile model.Profile) (model.Failure, error) {
		attempted = append(attempted, profile.Name)
		if profile.Name == "account-a" {
			return model.Failure{Kind: model.FailureQuota, Message: "quota exhausted"}, errors.New("quota exhausted")
		}
		return model.Failure{Kind: model.FailureNone}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(attempted) != 2 || attempted[0] != "account-a" || attempted[1] != "account-b" {
		t.Fatalf("balanced fallback order = %v, want [account-a account-b]", attempted)
	}
}
