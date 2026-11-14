package classifier

import (
	"errors"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestClassifyFailures(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		output       string
		wantKind     model.FailureKind
		wantDuration time.Duration
	}{
		{
			name:         "Rate limit 429 with minutes",
			err:          errors.New("exit status 1"),
			output:       "Error: 429 Too Many Requests. Rate limit reached. Try again in 37 minutes.",
			wantKind:     model.FailureRateLimit,
			wantDuration: 37 * time.Minute,
		},
		{
			name:         "Rate limit with seconds",
			err:          errors.New("exit status 1"),
			output:       "HTTP 429: You have exceeded your current quota. Retry in 45s.",
			wantKind:     model.FailureRateLimit,
			wantDuration: 45 * time.Second,
		},
		{
			name:         "Structured provider resource exhausted",
			err:          nil,
			output:       "RESOURCE_EXHAUSTED: request limit reached",
			wantKind:     model.FailureRateLimit,
			wantDuration: 0,
		},
		{
			name:         "Auth required",
			err:          errors.New("exit status 1"),
			output:       "Please run /login to authenticate with your Google account.",
			wantKind:     model.FailureAuth,
			wantDuration: 0,
		},
		{
			name:         "Quota out of credits",
			err:          errors.New("exit status 1"),
			output:       "Your credit balance is too low to complete this request.",
			wantKind:     model.FailureQuota,
			wantDuration: 0,
		},
		{
			name:         "Network failure",
			err:          errors.New("exit status 1"),
			output:       "dial tcp 142.250.190.46:443: connect: connection refused",
			wantKind:     model.FailureNetwork,
			wantDuration: 0,
		},
		{
			name:         "Provider 503 Overloaded",
			err:          errors.New("exit status 1"),
			output:       "HTTP 503 Service Unavailable: Model overloaded, please retry later.",
			wantKind:     model.FailureProvider,
			wantDuration: 0,
		},
		{
			name:         "Quota word in cache filename is not quota exhaustion",
			err:          errors.New("exit status 1"),
			output:       "failed to write usage.json quota cache",
			wantKind:     model.FailureUnknown,
			wantDuration: 0,
		},
		{
			// Exact AGY weekly-quota surface seen in production (2026-11-14):
			// previously classified as Unknown, so failover never rotated.
			name:         "AGY individual quota with weekly reset",
			err:          errors.New("exit status 1"),
			output:       "⚠ Individual quota reached. Please upgrade your subscription to increase your limits. Resets in 167h58m45s.\nError ID: 9afeb906-c1f0-4b24-8f6e-8382bc642753-353",
			wantKind:     model.FailureQuota,
			wantDuration: 167*time.Hour + 58*time.Minute + 45*time.Second,
		},
		{
			name:         "AGY individual quota without reset hint",
			err:          errors.New("exit status 1"),
			output:       "Individual quota reached. Please upgrade your subscription to increase your limits.",
			wantKind:     model.FailureQuota,
			wantDuration: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := Classify(tc.err, tc.output)
			if res.Kind != tc.wantKind {
				t.Fatalf("expected kind %s, got %s", tc.wantKind, res.Kind)
			}
			if tc.wantDuration > 0 {
				if res.RetryAfter == nil || *res.RetryAfter != tc.wantDuration {
					t.Fatalf("expected retry after %v, got %v", tc.wantDuration, res.RetryAfter)
				}
			}
		})
	}
}

func TestExtractResetDurationVariants(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want time.Duration
	}{
		{"weekly hours minutes seconds", "Resets in 167h58m45s", 167*time.Hour + 58*time.Minute + 45*time.Second},
		{"minutes seconds", "Resets in 58m45s", 58*time.Minute + 45*time.Second},
		{"only seconds", "Resets in 45s", 45 * time.Second},
		{"only hours", "Resets in 2h", 2 * time.Hour},
		{"no digits yields zero", "Resets in a while", 0},
		{"retry wording still parses", "Try again in 37 minutes", 37 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExtractRetryDuration(tc.in); got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestQuotaFailureCarriesResetAt(t *testing.T) {
	res := Classify(errors.New("exit status 1"), "Individual quota reached. Please upgrade your subscription to increase your limits. Resets in 167h58m45s.")
	if res.Kind != model.FailureQuota {
		t.Fatalf("expected quota, got %s", res.Kind)
	}
	if res.ResetAt == nil {
		t.Fatal("expected ResetAt to be set when the provider states a reset window")
	}
	if delta := time.Until(*res.ResetAt); delta <= 0 || delta > 168*time.Hour {
		t.Fatalf("expected reset within a weekly window, got %v", delta)
	}
}

func TestClassifyCodexTuiLimitBanners(t *testing.T) {
	cases := []struct {
		name   string
		output string
	}{
		{"hit usage limit", "✗ You've hit your usage limit. Your limit will reset at 5pm."},
		{"reached usage limit", "You've reached your usage limit for this plan."},
		{"reset signal only", "Your limit will reset in 2h 30m."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := Classify(errors.New("exit status 1"), tc.output)
			if res.Kind != model.FailureRateLimit {
				t.Fatalf("kind = %s, want rate limit", res.Kind)
			}
		})
	}
}
