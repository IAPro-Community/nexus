package nexus

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
)

func TestIntelligenceProbeFailureDiagnosticsNeverLeakProviderOutput(t *testing.T) {
	secret := "sk-secret-private-do-not-expose"
	cases := []struct {
		name   string
		err    error
		reason string
	}{
		{"not configured", intelligence.ErrIntelligenceUnavailable, "not_configured"},
		{"bad credentials", errors.New("openai error (401): invalid key " + secret), "authentication"},
		{"quota", errors.New("openai error (429): quota exceeded " + secret), "rate_limit"},
		{"missing model", errors.New("model not found /private/path/" + secret), "model"},
		{"timeout", context.DeadlineExceeded, "timeout"},
		{"connection", errors.New("connection refused: " + secret), "network"},
		{"JSON format", errors.New("decode cli: invalid JSON from /secret/" + secret), "response_format"},
		{"unknown", errors.New("provider unknown stderr " + secret), "provider_failure"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if reason := IntelligenceProbeFailureReason(tc.err); reason != tc.reason {
				t.Fatalf("expected %s, got %s", tc.reason, reason)
			}
			message := IntelligenceProbeFailureMessage(tc.err)
			if strings.Contains(message, secret) {
				t.Fatalf("unsafe diagnostic leaked provider output: %s", message)
			}
			if strings.TrimSpace(message) == "" {
				t.Fatal("missing user-facing guidance")
			}
		})
	}
}

func TestIntelligenceProbeSuccessHasNoFailureReason(t *testing.T) {
	if got := IntelligenceProbeFailureReason(nil); got != "" {
		t.Fatalf("unexpected error reason: %s", got)
	}
}
