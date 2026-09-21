package nexus

import (
	"context"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/nexus/capacitymonitor"
	"github.com/IAPro-Community/nexus/internal/profile"
)

// The hosted monitor consumes the canonical quota observation cache through a
// read-only ProbeRegistry. QuotaMonitorService remains the single live
// collector, avoiding duplicate provider/API/app-server probes.
// StartCapacityMonitor hosts the capacity Monitor loop in the Nexus process
// exactly once, mirroring StartQuotaMonitor. Consent-gated: it starts only
// when preferences exist with Enabled AND BackgroundEnabled (default OFF).
// Idempotent, lease-gated across processes, safe after restart (the store
// lease, not memory, decides leadership). Never autostarts OS login items:
// that is AutoStartManager's separate, explicit surface.
func (n *Nexus) StartCapacityMonitor(ctx context.Context) {
	n.mu.Lock()
	if n.capacityMonitorCancel != nil {
		n.mu.Unlock()
		return
	}
	st := n.st
	n.mu.Unlock()
	if st == nil {
		return
	}
	prefs, err := st.GetCapacityPreferences()
	if err != nil || !prefs.Enabled || !prefs.BackgroundEnabled {
		return
	}
	gate := capacitymonitor.NewCoordinator(nil)
	// Single dispatcher, process-wide: the legacy loop and the new loop
	// share one gate from the moment the new loop is hosted.
	DefaultQuotaDropMonitor().SetDispatchGate(gate)
	mon := capacitymonitor.NewMonitor(capacitymonitor.Dependencies{
		Profiles:      profile.List,
		ProbeRegistry: profile.NewCapacityProbeRegistry(),
		Store:         st,
		Bus:           events.DefaultBus(),
		Clock:         capacitymonitor.RealClock{},
		JitterSource:  capacitymonitor.RealJitter{},
		Recommender: &CapacityRecommender{Accounts: func() []ProviderAccount {
			accs, _ := NewResourceApplicationService(n).List(context.Background())
			return accs
		}},
		Dispatcher: gate,
	})
	runCtx, cancel := context.WithCancel(ctx)
	n.mu.Lock()
	if n.capacityMonitorCancel != nil {
		n.mu.Unlock()
		cancel()
		return
	}
	n.capacityMonitor = mon
	n.capacityMonitorCancel = cancel
	n.mu.Unlock()
	go mon.Run(runCtx)
}

// StopCapacityMonitor stops the hosted loop without affecting runtimes or
// the legacy quota service. Safe to call when never started.
func (n *Nexus) StopCapacityMonitor() {
	n.mu.Lock()
	cancel := n.capacityMonitorCancel
	n.capacityMonitorCancel = nil
	n.capacityMonitor = nil
	n.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// CapacityMonitorRunning reports whether this process hosts the loop.
func (n *Nexus) CapacityMonitorRunning() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.capacityMonitorCancel != nil
}
