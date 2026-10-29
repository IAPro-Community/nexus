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
