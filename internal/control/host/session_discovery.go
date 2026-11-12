package host

import (
	"context"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
)

// Session discovery pacing. The poll interval must stay well below a quota
// monitor tick so continuity preflight can rely on a discovered session id,
// while remaining cheap enough to run for the whole session lifetime.
var (
	sessionDiscoveryPollInterval = 2 * time.Second
	sessionDiscoveryProbeTimeout = 5 * time.Second
)

// SessionIDDiscovererFunc probes the provider's own session store for the live
// provider session id of this runtime. Implementations must be read-only
// observers: no PTY writes, no profile mutations, and no contention with the
// interactive TUI. An empty id means "not observed yet"; the host keeps
// polling until the id appears or the host lifecycle ends.
type SessionIDDiscovererFunc func(ctx context.Context, session registry.RuntimeSession, startedAt time.Time) (string, error)

// discoverProviderSessionID adopts the provider session id for PTY runtimes.
// Structured backends learn the id from the provider protocol and set it
// themselves; this watcher only fills the gap for terminal runtimes. It never
// touches the input path and never writes to the provider PTY.
func (sh *SessionHost) discoverProviderSessionID(startedAt time.Time) {
	for {
		select {
		case <-sh.stopChan:
			return
		case <-sh.doneChan:
			return
		default:
		}

		sh.mu.RLock()
		session := sh.session
		sh.mu.RUnlock()
		if strings.TrimSpace(session.ProviderSessionID) != "" {
			// A structured backend (or an earlier discovery) already owns it.
			return
		}

		// Tie the probe to the host lifecycle: a session store walk must never
		// outlive a stopped/terminated host and keep a goroutine waiting. The
		// probe goroutine is abandoned on stop (its ctx is canceled; bounded
		// work finishes on its own) — its local result is never read again.
		ctx, cancel := context.WithTimeout(context.Background(), sessionDiscoveryProbeTimeout)
		probeDone := make(chan struct{})
		var id string
		var err error
		go func() {
			defer close(probeDone)
			id, err = sh.cfg.SessionIDDiscoverer(ctx, session, startedAt)
		}()
		select {
		case <-probeDone:
		case <-sh.stopChan:
			cancel()
			return
		case <-sh.doneChan:
			cancel()
			return
		}
		cancel()

		if err == nil && strings.TrimSpace(id) != "" {
			sh.mu.Lock()
			adopt := strings.TrimSpace(sh.session.ProviderSessionID) == ""
			if adopt {
				sh.session.ProviderSessionID = id
			}
			sh.mu.Unlock()
			if adopt {
				nexusruntime.TraceTerminalBytes(sh.session.RuntimeID, "host.session_discovered", []byte(id))
				_ = sh.registry.UpdateProviderSessionID(sh.session.RuntimeID, id)
			}
			return
		}

		select {
		case <-sh.stopChan:
			return
		case <-sh.doneChan:
			return
		case <-time.After(sessionDiscoveryPollInterval):
		}
	}
}
