package cooldown

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func testScope(provider, profile, account string) model.AccountScope {
	return model.AccountScope{ProviderID: provider, ProfileID: profile, AccountID: account, IdentityVersion: "1"}
}

func durationPointer(d time.Duration) *time.Duration { return &d }

func timePointer(t time.Time) *time.Time { return &t }

func TestRetryAfterPrecedenceOverHeuristic(t *testing.T) {
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	tracker := NewTracker()
	retry := 90 * time.Second
	tracker.RecordFailure("codex", "work", model.Failure{
		Kind:       model.FailureRateLimit,
		Message:    "429",
		RetryAfter: durationPointer(retry),
	}, "429")
	limited, rec := tracker.IsRateLimited("codex", "work")
	if !limited || rec == nil {
		t.Fatalf("expected rate limited")
	}
	// Provider signal must win: reset ~90s, not 15m heuristic.
	remaining := time.Until(rec.ResetAt)
	if remaining > 3*time.Minute || remaining < 30*time.Second {
		t.Fatalf("Retry-After precedence violated: remaining=%v reset=%v retry=%v", remaining, rec.ResetAt, rec.RetryAfter)
	}
	if rec.RetryAfter != retry {
		t.Fatalf("expected RetryAfter=%v got %v", retry, rec.RetryAfter)
	}
}

func TestResetAtPrecedenceOverRetryAfter(t *testing.T) {
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	tracker := NewTracker()
	reset := time.Now().Add(2 * time.Hour)
	retry := 30 * time.Second
	tracker.RecordFailure("codex", "work", model.Failure{
		Kind:       model.FailureRateLimit,
		Message:    "429",
		RetryAfter: durationPointer(retry),
		ResetAt:    timePointer(reset),
	}, "429")
	_, rec := tracker.IsRateLimited("codex", "work")
	if rec == nil {
		t.Fatalf("expected record")
	}
	// Absolute ResetAt wins over relative Retry-After.
	if rec.ResetAt.Sub(reset) > time.Minute || reset.Sub(rec.ResetAt) > time.Minute {
		t.Fatalf("ResetAt precedence violated: got %v want %v", rec.ResetAt, reset)
	}
}

func TestFallbackWithoutMetadataUsesHeuristic(t *testing.T) {
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	tracker := NewTracker()
	tracker.RecordFailure("codex", "work", model.Failure{
		Kind:    model.FailureRateLimit,
		Message: "429 without metadata",
	}, "429")
	_, rec := tracker.IsRateLimited("codex", "work")
	if rec == nil {
		t.Fatalf("expected record")
	}
	if rec.RetryAfter != DefaultFallbackCooldown {
		t.Fatalf("expected heuristic %v got %v", DefaultFallbackCooldown, rec.RetryAfter)
	}
}

func TestKnownResetNeverReplacedByArbitraryCooldown(t *testing.T) {
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	tracker := NewTracker()
	knownReset := time.Now().Add(2 * time.Hour)
	tracker.RecordFailure("codex", "work", model.Failure{
		Kind:    model.FailureRateLimit,
		Message: "429 with reset",
		ResetAt: timePointer(knownReset),
	}, "first")
	// Second failure without metadata must not shorten the known reset.
	tracker.RecordFailure("codex", "work", model.Failure{
		Kind:    model.FailureRateLimit,
		Message: "429 without metadata",
	}, "second")
	_, rec := tracker.IsRateLimited("codex", "work")
	if rec == nil {
		t.Fatalf("expected record")
	}
	if rec.ResetAt.Before(knownReset.Add(-time.Minute)) {
		t.Fatalf("known reset was replaced by arbitrary cooldown: got %v want ~%v", rec.ResetAt, knownReset)
	}
}

func TestExpiredResetIsNotRateLimited(t *testing.T) {
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	tracker := NewTracker()
	past := time.Now().Add(-10 * time.Minute)
	tracker.RecordFailure("codex", "work", model.Failure{
		Kind:    model.FailureRateLimit,
		Message: "old",
		ResetAt: timePointer(past),
	}, "old")
	limited, _ := tracker.IsRateLimited("codex", "work")
	if limited {
		t.Fatalf("expired reset must not be rate limited")
	}
}

func TestMalformedMetadataFallsBackToHeuristic(t *testing.T) {
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	tracker := NewTracker()
	neg := -5 * time.Second
	zero := time.Time{}
	tracker.RecordFailure("codex", "bad1", model.Failure{
		Kind:       model.FailureRateLimit,
		Message:    "bad retry",
		RetryAfter: durationPointer(neg),
	}, "bad")
	_, rec := tracker.IsRateLimited("codex", "bad1")
	if rec == nil || rec.RetryAfter != DefaultFallbackCooldown {
		t.Fatalf("negative Retry-After must fall back to heuristic, got %+v", rec)
	}
	tracker.RecordFailure("codex", "bad2", model.Failure{
		Kind:    model.FailureRateLimit,
		Message: "zero reset",
		ResetAt: &zero,
	}, "bad")
	_, rec2 := tracker.IsRateLimited("codex", "bad2")
	if rec2 == nil || rec2.RetryAfter != DefaultFallbackCooldown {
		t.Fatalf("zero ResetAt must fall back to heuristic, got %+v", rec2)
	}
}

func TestAccountScopedIsolationAndRestart(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("AI_CLI_STATE_DIR", stateDir)
	tracker := NewTracker()
	a := testScope("codex", "work", "a")
	b := testScope("codex", "work", "b")
	reset := time.Now().Add(time.Hour)
	tracker.RecordFailureForScope(a, model.Failure{
		Kind:    model.FailureRateLimit,
		Message: "429",
		ResetAt: timePointer(reset),
	}, "429")
	if limited, _ := tracker.IsRateLimitedForScope(b); limited {
		t.Fatalf("account B inherited account A cooldown")
	}
	if limited, _ := tracker.IsRateLimitedForScope(a); !limited {
		t.Fatalf("account A cooldown missing")
	}
	// Restart: new tracker must reload the scoped record from disk.
	reloaded := NewTracker()
	if limited, rec := reloaded.IsRateLimitedForScope(a); !limited || rec == nil {
		t.Fatalf("scoped cooldown did not survive restart")
	} else if rec.ResetAt.Before(reset.Add(-time.Minute)) {
		t.Fatalf("restarted reset mismatch: got %v want %v", rec.ResetAt, reset)
	}
	if limited, _ := reloaded.IsRateLimitedForScope(b); limited {
		t.Fatalf("restarted tracker leaked scope")
	}
}

func TestEffectiveCooldownPrecedenceUnit(t *testing.T) {
	reset := time.Now().Add(time.Hour)
	retry, gotReset, signal := EffectiveCooldown(model.Failure{
		RetryAfter: durationPointer(time.Minute),
		ResetAt:    timePointer(reset),
	})
	if !signal {
		t.Fatalf("expected provider signal")
	}
	if gotReset.Sub(reset) > time.Second || reset.Sub(gotReset) > time.Second {
		t.Fatalf("ResetAt must win, got %v want %v (retry %v)", gotReset, reset, retry)
	}
	_, _, signal = EffectiveCooldown(model.Failure{Kind: model.FailureRateLimit})
	if signal {
		t.Fatalf("empty metadata must not be a provider signal")
	}
}
