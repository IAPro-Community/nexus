package web

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
	coreprovider "github.com/IAPro-Community/nexus/internal/core/provider"
)

func TestProviderRegistrationStatePrioritizesAuthenticatedAccount(t *testing.T) {
	ctx := context.Background()
	profiles := []model.Profile{
		{Provider: "codex", Name: "personal"},
		{Provider: "codex", Name: "work"},
		{Provider: "claude", Name: "other"},
	}
	// A pending account later in the list must never demote a verified one.
	got := providerRegistrationState(ctx, "codex", true, profiles, func(_ context.Context, p model.Profile) (bool, error) {
		return p.Name == "personal", nil
	})
	if got != string(coreprovider.RegisteredAuthenticated) {
		t.Fatalf("authenticated account demoted to %q", got)
	}

	got = providerRegistrationState(ctx, "codex", true, profiles, func(context.Context, model.Profile) (bool, error) { return false, nil })
	if got != string(coreprovider.PendingAuth) {
		t.Fatalf("pending account expected, got %q", got)
	}
	got = providerRegistrationState(ctx, "codex", false, profiles, func(context.Context, model.Profile) (bool, error) { return true, nil })
	if got != "" {
		t.Fatalf("missing CLI cannot be marked ready: %q", got)
	}
	got = providerRegistrationState(ctx, "codex", true, []model.Profile{
		{Provider: "codex", Name: "disabled", Disabled: true},
	}, func(context.Context, model.Profile) (bool, error) { return true, nil })
	if got != string(coreprovider.InstalledUnregistered) {
		t.Fatalf("disabled profile cannot be marked ready: %q", got)
	}
}

func TestProviderRegistrationStateDoesNotTurnUnverifiedProbeIntoLoginRequired(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	profiles := []model.Profile{{Provider: "cursor", Name: "slow"}}
	got := providerRegistrationState(ctx, "cursor", true, profiles, func(ctx context.Context, _ model.Profile) (bool, error) {
		<-ctx.Done()
		return false, ctx.Err()
	})
	if got != string(coreprovider.RegistrationUnknown) {
		t.Fatalf("timed-out probe must remain unknown, got %q", got)
	}
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("test context did not expire: %v", ctx.Err())
	}
}
