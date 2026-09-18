package codex

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

// appServerProbeTTL bounds how often a profile may spawn an app-server probe.
// The TUI, the quota monitor and the scheduler all read usage independently, so
// without this the same account would start several subprocesses per second.
const appServerProbeTTL = 60 * time.Second

// appServerFailureTTL is the cooldown after a failed probe. Shorter than the
// success TTL so a login or a network recovery is picked up quickly, long enough
// that a permanently unavailable app-server does not spawn a process per render.
const appServerFailureTTL = 20 * time.Second

type probeResult struct {
	snap model.UsageSnapshot
	ok   bool
	at   time.Time
}

type probeSlot struct {
	// mu serializes probes for one profile so concurrent readers coalesce onto
	// a single subprocess instead of racing.
	mu     sync.Mutex
	result probeResult
}

var (
	probeSlotsMu sync.Mutex
	probeSlots   = map[string]*probeSlot{}
)

func probeSlotFor(key string) *probeSlot {
	probeSlotsMu.Lock()
	defer probeSlotsMu.Unlock()
	slot, ok := probeSlots[key]
	if !ok {
		slot = &probeSlot{}
		probeSlots[key] = slot
	}
	return slot
}

func (r probeResult) fresh(now time.Time) bool {
	if r.at.IsZero() {
		return false
	}
	ttl := appServerProbeTTL
	if !r.ok {
		ttl = appServerFailureTTL
	}
	return now.Sub(r.at) < ttl
}

// appServerUsage returns the official quota snapshot for a profile, reusing a
// recent probe when possible. Concurrent callers for the same profile wait for
// the in-flight probe rather than starting another one.
func (a *Adapter) appServerUsage(ctx context.Context, p model.Profile, force bool) (model.UsageSnapshot, bool) {
	key := string(a.ID()) + "/" + p.Name
	slot := probeSlotFor(key)

	if !force {
		slot.mu.Lock()
		cached := slot.result
		slot.mu.Unlock()
		if cached.fresh(time.Now()) {
			return a.serveProbeCache(ctx, p, cached)
		}
	}

	slot.mu.Lock()
	defer slot.mu.Unlock()

	// A concurrent caller may have just refreshed while we waited for the lock.
	if cached := slot.result; cached.fresh(time.Now()) && !cached.at.IsZero() {
		if !force || cached.at.After(time.Now().Add(-time.Second)) {
			return a.serveProbeCache(ctx, p, cached)
		}
	}

	snap, ok := a.fetchUsageFromAppServer(ctx, p)
	slot.result = probeResult{snap: snap, ok: ok, at: time.Now()}
	return snap, ok
}

// serveProbeCache revalidates auth before returning a memoized LIVE reading as
// CACHED. Logout or identity change must not keep serving another account's
// quota under this profile.
func (a *Adapter) serveProbeCache(ctx context.Context, p model.Profile, cached probeResult) (model.UsageSnapshot, bool) {
	if !cached.ok {
		return model.UsageSnapshot{}, false
	}
	info := a.InspectAuth(ctx, p)
	if !info.Authenticated {
		return model.UsageSnapshot{}, false
	}
	expected := strings.TrimSpace(info.ExternalAccountID)
	if expected == "" {
		return model.UsageSnapshot{}, false
	}
	if snapAccount := strings.TrimSpace(cached.snap.Account); snapAccount != "" && !strings.EqualFold(snapAccount, expected) {
		return model.UsageSnapshot{}, false
	}
	snap := cached.snap
	if snap.Status == model.UsageLive {
		snap.Status = model.UsageCached
	}
	return snap, true
}

// resetAppServerProbeCache clears memoized probes. Tests only.
func resetAppServerProbeCache() {
	probeSlotsMu.Lock()
	defer probeSlotsMu.Unlock()
	probeSlots = map[string]*probeSlot{}
}
