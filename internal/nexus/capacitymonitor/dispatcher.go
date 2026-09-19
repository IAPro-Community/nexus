package capacitymonitor

import (
	"strings"
	"sync"
	"time"
)

// DispatchClass collapses monitor-specific states into the cross-loop
// identity used by the single dispatcher. WARNING and CRITICAL are both
// "low": the legacy QuotaDropMonitor only knows LOW vs EXHAUSTED, so the
// shared key must not split what the other loop cannot distinguish. Exact
// states stay in each loop's own dedup (ShouldDeliver / milestones).
type DispatchClass string

const (
	DispatchLow       DispatchClass = "low"
	DispatchExhausted DispatchClass = "exhausted"
	DispatchRecovered DispatchClass = "recovered"
)

// Loop owners known to the single dispatcher.
const (
	OwnerCapacityMonitor = "capacity-monitor"
	OwnerLegacyQuota     = "legacy-quota"
)

// ClassForState maps a monitor state to its dispatch class.
// HEALTHY and UNKNOWN never announce, so they map to "" (no claim).
func ClassForState(state State) DispatchClass {
	switch state {
	case StateWarning, StateCritical:
		return DispatchLow
	case StateExhausted:
		return DispatchExhausted
	case StateRecovered:
		return DispatchRecovered
	default:
		return ""
	}
}

// DispatchKey is the cross-loop announcement identity: provider/profile +
// group/window + severity class. Scope and reset descriptions are excluded
// on purpose (same rationale as DeriveFingerprint): credential rotation or
// a ticking countdown must not fork the identity.
type DispatchKey struct {
	Provider string
	Profile  string
	Group    string
	Window   string
	Class    DispatchClass
}

func (k DispatchKey) String() string {
	return strings.Join([]string{k.Provider, k.Profile, k.Group, k.Window, string(k.Class)}, "|")
}

type dispatchClaim struct {
	owner string
	at    time.Time
}

// Coordinator is the single dispatcher between the new capacity Monitor and
// the legacy QuotaDropMonitor. Each loop claims with its owner id before
// announcing; a claim loses only when the OTHER loop claimed the same key
// within ttl. Same-owner repeats always win (each loop owns its repeat
// policy: ShouldDeliver / milestones), so the gate can never silence a
// loop's own step-down or reminder sequence. A nil *Coordinator allows
// (never suppresses): legacy call sites and tests that do not wire it keep
// exact previous behavior.
type Coordinator struct {
	mu    sync.Mutex
	clock Clock
	last  map[string]dispatchClaim
}

// NewCoordinator builds an idle coordinator. A nil clock uses RealClock.
func NewCoordinator(clock Clock) *Coordinator {
	if clock == nil {
		clock = RealClock{}
	}
	return &Coordinator{clock: clock, last: make(map[string]dispatchClaim)}
}

// Claim records an announcement intent and reports whether the caller owns
// this cycle. An empty class never claims.
func (c *Coordinator) Claim(owner string, key DispatchKey, ttl time.Duration) bool {
	if c == nil || key.Class == "" {
		return true
	}
	now := c.clock.Now()
	k := key.String()
	c.mu.Lock()
	defer c.mu.Unlock()
	if prev, ok := c.last[k]; ok && prev.owner != owner && now.Sub(prev.at) < ttl {
		return false
	}
	c.last[k] = dispatchClaim{owner: owner, at: now}
	return true
}

// Forget releases a key when a claim did not materialize (e.g. the event
// row write failed after claiming): the next pass must be free to retry
// instead of waiting out the TTL for an announcement that never happened.
// Tests and explicit repeat policy only.
func (c *Coordinator) Forget(key DispatchKey) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.last, key.String())
}
