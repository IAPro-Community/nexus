package web

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
	coreprovider "github.com/IAPro-Community/nexus/internal/core/provider"
)

func TestProviderRegistrationStatePrioritizesAuthenticatedAccount(t *testing.T) {
	profiles := []model.Profile{
		{Provider: "codex", Name: "personal"},
		{Provider: "codex", Name: "work"},
		{Provider: "claude", Name: "other"},
	}
	// A pending account later in the list must never demote a verified one.
	got := providerRegistrationState("codex", true, profiles, func(p model.Profile) bool {
		return p.Name == "personal"
	})
	if got != string(coreprovider.RegisteredAuthenticated) {
		t.Fatalf("authenticated account demoted to %q", got)
	}

	got = providerRegistrationState("codex", true, profiles, func(model.Profile) bool { return false })
	if got != string(coreprovider.PendingAuth) {
		t.Fatalf("pending account expected, got %q", got)
	}
	got = providerRegistrationState("codex", false, profiles, func(model.Profile) bool { return true })
	if got != "" {
		t.Fatalf("missing CLI cannot be marked ready: %q", got)
	}
	got = providerRegistrationState("codex", true, []model.Profile{
		{Provider: "codex", Name: "disabled", Disabled: true},
	}, func(model.Profile) bool { return true })
	if got != string(coreprovider.InstalledUnregistered) {
		t.Fatalf("disabled profile cannot be marked ready: %q", got)
	}
}
