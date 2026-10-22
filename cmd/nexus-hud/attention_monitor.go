package main

import (
	"context"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/notify"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus"
)

// hudAttentionMonitor reads only the canonical SessionHost registry. It keeps
// operating when both Wails windows are hidden or the browser is minimized.
// The OS notifier receives generic text without terminal output or secrets.
type hudAttentionMonitor struct {
	source  func() []registry.RuntimeSession
	deliver func(string, string) error
	seen    map[string]struct{}
}

func newHUDAttentionMonitor() *hudAttentionMonitor {
	return &hudAttentionMonitor{
		source: func() []registry.RuntimeSession { return registry.DefaultRegistry().List() },
		deliver: func(runtimeID, fingerprint string) error {
			return notify.Default().Notify(notify.Payload{
				Title: "Nexus · Intervenção necessária",
				Body:  "Um terminal ou agente aguarda uma resposta. Abra o Nexus Guardian para verificar.",
				Tag:   "hud-attention:" + runtimeID + ":" + fingerprint,
			})
		},
		seen: make(map[string]struct{}),
	}
}

func (m *hudAttentionMonitor) poll() {
	items := nexus.BuildHUDRuntimeAttention(m.source(), "")
	active := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.Fingerprint == "" || item.RuntimeID == "" {
			continue
		}
		key := item.RuntimeID + ":" + item.Fingerprint
		active[key] = struct{}{}
		if _, wasPresent := m.seen[key]; !wasPresent {
			// Failure to post an OS toast must not flood an inaccessible
			// notification center. The HUD badge persists independently.
			_ = m.deliver(item.RuntimeID, item.Fingerprint)
		}
	}
	m.seen = active
}

func (m *hudAttentionMonitor) run(ctx context.Context) {
	const interval = 5 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	m.poll()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.poll()
		}
	}
}
