package codex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	sdkproviders "github.com/IAPro-Community/IAPro-SDK/providers"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

const appServerProbeTTL = 60 * time.Second
const appServerFailureTTL = 20 * time.Second

type quotaCacheAdapter struct{}

func quotaAccount(home, profile string) (sdkproviders.Account, error) {
	scope, err := readProfileScope(home, profile)
	if err != nil {
		return sdkproviders.Account{}, err
	}
	// Scope the local observation cache to the host's authorized profile
	// location; this is not the owner ID of a cross-product lease.
	owner := sha256.Sum256([]byte(home))
	return sdkproviders.Account{OwnerID: hex.EncodeToString(owner[:]), Scope: scope}, nil
}

func (quotaCacheAdapter) InspectAuth(ctx context.Context, account sdkproviders.Account) (sdkproviders.Authentication, error) {
	if err := ctx.Err(); err != nil {
		return sdkproviders.Authentication{}, err
	}
	home, err := config.ProfileHome(account.Scope.ProviderID, account.Scope.ProfileID)
	if err != nil {
		return sdkproviders.Authentication{}, sdkproviders.ErrIdentityMismatch
	}
	current, err := quotaAccount(home, account.Scope.ProfileID)
	if err != nil || current != account {
		return sdkproviders.Authentication{}, sdkproviders.ErrIdentityMismatch
	}
	info := New().InspectAuth(ctx, model.Profile{Provider: account.Scope.ProviderID, Name: account.Scope.ProfileID})
	if !info.Authenticated || info.ExternalAccountID != account.Scope.AccountID {
		return sdkproviders.Authentication{}, sdkproviders.ErrIdentityMismatch
	}
	return sdkproviders.Authentication{Scope: current.Scope, Authenticated: true, ObservedAt: time.Now()}, nil
}

func (quotaCacheAdapter) GetUsage(ctx context.Context, account sdkproviders.Account) (model.UsageSnapshot, error) {
	snap, ok := New().fetchUsageFromAppServer(ctx, model.Profile{Provider: account.Scope.ProviderID, Name: account.Scope.ProfileID})
	if !ok {
		return model.UsageSnapshot{}, sdkproviders.ErrProbeFailed
	}
	if snap.AccountScope != account.Scope {
		return model.UsageSnapshot{}, sdkproviders.ErrIdentityMismatch
	}
	return snap, nil
}

func newAppServerUsageCache() *sdkproviders.UsageCache {
	adapter := quotaCacheAdapter{}
	cache, err := sdkproviders.NewUsageCache(adapter, adapter, appServerProbeTTL, appServerFailureTTL, 256, time.Now)
	if err != nil {
		panic(err) // static configuration; not dependent on runtime credentials
	}
	return cache
}

var appServerUsageCache = newAppServerUsageCache()

// appServerUsage delegates coalescing, TTL, backoff, explicit refresh and
// defensive snapshot copies to IAPro-SDK.
func (a *Adapter) appServerUsage(ctx context.Context, p model.Profile, force bool) (model.UsageSnapshot, bool) {
	if !appServerEnabled() {
		return model.UsageSnapshot{}, false
	}
	home, err := config.ProfileHome(string(a.ID()), p.Name)
	if err != nil {
		return model.UsageSnapshot{}, false
	}
	account, err := quotaAccount(home, p.Name)
	if err != nil {
		return model.UsageSnapshot{}, false
	}
	var snap model.UsageSnapshot
	if force {
		snap, err = appServerUsageCache.RefreshUsage(ctx, account, time.Second)
	} else {
		snap, err = appServerUsageCache.GetUsage(ctx, account)
	}
	if err != nil {
		return model.UsageSnapshot{}, false
	}
	// The SDK cache has already verified the local scope. The Nexus profile
	// layer binds it to its own persisted account identity, as before.
	snap.AccountScope = model.AccountScope{}
	return snap, true
}

// Test-only reset; callers install fake servers before starting readers.
func resetAppServerProbeCache() {
	appServerUsageCache = newAppServerUsageCache()
}
