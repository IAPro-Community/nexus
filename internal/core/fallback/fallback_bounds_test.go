package fallback

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/cooldown"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/core/scheduler"
)

func testSetup(t *testing.T) (*Executor, []model.Profile, map[string]model.AccountInfo) {
	t.Helper()
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
	return exec, nil, nil
}

func liveAccounts(names ...string) map[string]model.AccountInfo {
	out := map[string]model.AccountInfo{}
	for _, n := range names {
		pct := 80.0
		out[n] = model.AccountInfo{
			Authenticated: true,
			Health:        model.HealthHealthy,
			Usage: model.UsageSnapshot{
				ProviderID: "codex", ProfileID: n,
				Status: model.UsageLive, Source: model.SourceCLIOutput, FetchedAt: time.Now(),
				Windows: []model.UsageWindow{{Kind: "5h", RemainingPercent: &pct}},
			},
		}
	}
	return out
}

func profiles(provider string, names ...string) []model.Profile {
	out := []model.Profile{}
	for _, n := range names {
		out = append(out, model.Profile{Provider: provider, Name: n})
	}
	return out
}

func TestFallbackAllFailBounded(t *testing.T) {
	exec, _, _ := testSetup(t)
	candidates := profiles("codex", "a", "b", "c")
	accounts := liveAccounts("a", "b", "c")
	ctx := context.Background()
	attempts := 0
	order := []string{}
	err := exec.RunWithFallback(ctx, "codex", "/tmp", "a", candidates, accounts, true, func(p model.Profile) (model.Failure, error) {
		attempts++
		order = append(order, p.Name)
		return model.Failure{Kind: model.FailureRateLimit, Message: "429"}, errors.New("exit 1")
	})
	if err == nil {
		t.Fatalf("expected all-fail error")
	}
	if attempts != 3 {
		t.Fatalf("bound violated: attempts=%d order=%v want 3", attempts, order)
	}
	if !errors.Is(err, ErrFallbackRateLimited) && !errors.Is(err, ErrFallbackExhausted) {
		t.Fatalf("typed rate-limit cause lost: %v", err)
	}
	// Anti ping-pong: no repeats.
	seen := map[string]bool{}
	for _, n := range order {
		if seen[n] {
			t.Fatalf("ping-pong repeat: %v", order)
		}
		seen[n] = true
	}
}

func TestFallbackSingleResourceBound(t *testing.T) {
	exec, _, _ := testSetup(t)
	candidates := profiles("codex", "only")
	accounts := liveAccounts("only")
	ctx := context.Background()
	attempts := 0
	err := exec.RunWithFallback(ctx, "codex", "/tmp", "only", candidates, accounts, true, func(p model.Profile) (model.Failure, error) {
		attempts++
		return model.Failure{Kind: model.FailureRateLimit, Message: "429"}, errors.New("exit 1")
	})
	if err == nil {
		t.Fatalf("expected single-resource exhaustion")
	}
	if attempts != 1 {
		t.Fatalf("single resource must attempt once, got %d", attempts)
	}
	if !errors.Is(err, ErrFallbackRateLimited) {
		t.Fatalf("single resource must preserve rate-limit cause: %v", err)
	}
}

func TestFallbackRecoverySucceeds(t *testing.T) {
	exec, _, _ := testSetup(t)
	candidates := profiles("codex", "a", "b")
	accounts := liveAccounts("a", "b")
	ctx := context.Background()
	attempts := 0
	err := exec.RunWithFallback(ctx, "codex", "/tmp", "a", candidates, accounts, true, func(p model.Profile) (model.Failure, error) {
		attempts++
		if p.Name == "a" {
			return model.Failure{Kind: model.FailureRateLimit, Message: "429"}, errors.New("exit 1")
		}
		return model.Failure{Kind: model.FailureNone}, nil
	})
	if err != nil {
		t.Fatalf("recovery must succeed: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestFallbackQuotaExhaustedPreservesCause(t *testing.T) {
	exec, _, _ := testSetup(t)
	candidates := profiles("codex", "a", "b")
	accounts := liveAccounts("a", "b")
	ctx := context.Background()
	err := exec.RunWithFallback(ctx, "codex", "/tmp", "a", candidates, accounts, true, func(p model.Profile) (model.Failure, error) {
		return model.Failure{Kind: model.FailureQuota, Message: "quota exhausted"}, errors.New("exit 1")
	})
	if err == nil {
		t.Fatalf("expected quota error")
	}
	if !errors.Is(err, ErrFallbackQuotaExhausted) {
		t.Fatalf("quota cause must be preserved, got: %v", err)
	}
	if errors.Is(err, ErrFallbackRateLimited) {
		t.Fatalf("quota must not be reported as rate limit: %v", err)
	}
}

func TestFallbackUnauthenticatedNoRetry(t *testing.T) {
	exec, _, _ := testSetup(t)
	candidates := profiles("codex", "a", "b")
	accounts := liveAccounts("a", "b")
	ctx := context.Background()
	attempts := 0
	err := exec.RunWithFallback(ctx, "codex", "/tmp", "a", candidates, accounts, true, func(p model.Profile) (model.Failure, error) {
		attempts++
		return model.Failure{Kind: model.FailureAuth, Message: "login required"}, errors.New("exit 1")
	})
	if err == nil {
		t.Fatalf("expected auth error")
	}
	if attempts != 1 {
		t.Fatalf("unauthenticated must not retry, attempts=%d", attempts)
	}
	if !errors.Is(err, ErrFallbackUnauthenticated) {
		t.Fatalf("auth cause lost: %v", err)
	}
}

func TestFallbackUnknownQuotaPreservesCause(t *testing.T) {
	exec, _, _ := testSetup(t)
	candidates := profiles("codex", "a", "b")
	accounts := liveAccounts("a", "b")
	ctx := context.Background()
	attempts := 0
	err := exec.RunWithFallback(ctx, "codex", "/tmp", "a", candidates, accounts, true, func(p model.Profile) (model.Failure, error) {
		attempts++
		return model.Failure{Kind: model.FailureUnknown, Message: "unknown failure"}, errors.New("exit 1")
	})
	if err == nil {
		t.Fatalf("expected unknown error")
	}
	if attempts != 1 {
		t.Fatalf("unknown must not retry, attempts=%d", attempts)
	}
	if !errors.Is(err, ErrFallbackUnknownQuota) {
		t.Fatalf("unknown cause lost: %v", err)
	}
}

func TestFallbackNetworkFailureNoRetry(t *testing.T) {
	exec, _, _ := testSetup(t)
	candidates := profiles("codex", "a", "b")
	accounts := liveAccounts("a", "b")
	ctx := context.Background()
	attempts := 0
	err := exec.RunWithFallback(ctx, "codex", "/tmp", "a", candidates, accounts, true, func(p model.Profile) (model.Failure, error) {
		attempts++
		return model.Failure{Kind: model.FailureNetwork, Message: "dial tcp"}, errors.New("exit 1")
	})
	if err == nil {
		t.Fatalf("expected network error")
	}
	if attempts != 1 {
		t.Fatalf("network must not retry, attempts=%d", attempts)
	}
	if !errors.Is(err, ErrFallbackNetwork) {
		t.Fatalf("network cause lost: %v", err)
	}
}

func TestFallbackAntiPingPongThreeNodes(t *testing.T) {
	exec, _, _ := testSetup(t)
	candidates := profiles("codex", "a", "b", "c")
	accounts := liveAccounts("a", "b", "c")
	ctx := context.Background()
	order := []string{}
	err := exec.RunWithFallback(ctx, "codex", "/tmp", "a", candidates, accounts, true, func(p model.Profile) (model.Failure, error) {
		order = append(order, p.Name)
		return model.Failure{Kind: model.FailureRateLimit, Message: "429"}, errors.New("exit 1")
	})
	if err == nil {
		t.Fatalf("expected exhaustion")
	}
	if len(order) != 3 {
		t.Fatalf("expected exactly 3 attempts, got %v", order)
	}
	// Must be a permutation of a,b,c with no A→B→C→A cycle back to A.
	if order[0] != "a" {
		t.Fatalf("must start at requested a, got %v", order)
	}
	seen := map[string]int{}
	for i, n := range order {
		if _, dup := seen[n]; dup {
			t.Fatalf("cycle detected A→B→C→A: %v", order)
		}
		seen[n] = i
	}
}
