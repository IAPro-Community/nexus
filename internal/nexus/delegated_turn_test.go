package nexus

import (
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/quota"
)

func delegatedTestAccount(provider, profile, status string, available bool) ProviderAccount {
	return ProviderAccount{
		ID:            provider + ":" + profile,
		Provider:      provider,
		Profile:       profile,
		DisplayName:   provider + " (" + profile + ")",
		Authenticated: true,
		Available:     available,
		Health:        "healthy",
		Capabilities: map[string]string{
			"headless":      "SUPPORTED",
			"submit_prompt": "SUPPORTED",
		},
		QuotaView: &quota.QuotaView{
			Status:    status,
			Available: available,
		},
	}
}

func TestDelegatedTurnPromptIsExplicitlyReadOnly(t *testing.T) {
	got := delegatedTurnPrompt("revise o módulo")
	for _, required := range []string{"READ_ONLY", "Do not modify files", "User request:", "revise o módulo"} {
		if !strings.Contains(got, required) {
			t.Fatalf("delegated prompt missing %q: %s", required, got)
		}
	}
}

func TestSelectDelegatedTurnResourceAcceptsOpaqueRegisteredCodex(t *testing.T) {
	account := delegatedTestAccount("codex", "work", "UNSUPPORTED", true)
	got, fallback, _, err := selectDelegatedTurnResource(
		[]ProviderAccount{account},
		TaskRequirements{TaskKind: "review", Role: "reviewer"},
		"codex",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Provider != "codex" || got.Profile != "work" || fallback {
		t.Fatalf("unexpected delegated selection: account=%+v fallback=%t", got, fallback)
	}
}

func TestSelectDelegatedTurnResourceUsesHealthyAccountWithinPreferredProvider(t *testing.T) {
	limited := delegatedTestAccount("codex", "limited", "RATE_LIMITED", false)
	limited.RateLimited = true
	healthy := delegatedTestAccount("codex", "healthy", "UNSUPPORTED", true)
	got, fallback, _, err := selectDelegatedTurnResource(
		[]ProviderAccount{limited, healthy},
		TaskRequirements{TaskKind: "review", Role: "reviewer"},
		"codex",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Profile != "healthy" || fallback {
		t.Fatalf("expected healthy preferred account, got %+v fallback=%t", got, fallback)
	}
}

func TestBoundedDelegatedTurnOutput(t *testing.T) {
	input := strings.Repeat("x", maxDelegatedTurnOutputBytes+100)
	got := boundedDelegatedTurnOutput(input)
	if len(got) <= maxDelegatedTurnOutputBytes || !strings.Contains(got, "truncated") {
		t.Fatalf("delegated output was not bounded: len=%d", len(got))
	}
}
