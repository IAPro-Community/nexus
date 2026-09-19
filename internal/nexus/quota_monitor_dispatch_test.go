package nexus

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/notify"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/nexus/capacitymonitor"
)

func lowQuotaAccount() ProviderAccount {
	return ProviderAccount{
		Provider:       "claude",
		Profile:        "work",
		DisplayName:    "Claude Pro Work",
		QuotaRemaining: 0.10,
		QuotaTotal:     1.0,
		QuotaView: &quota.QuotaView{
			Status: "LIVE",
			ModelGroups: []quota.ModelGroup{
				{Windows: []quota.Window{{Kind: "5h", Remaining: 10, ResetDesc: "2h"}}},
			},
		},
	}
}

// Without a gate the legacy loop announces exactly as before.
func TestDispatchGateNilPreservesLegacy(t *testing.T) {
	rec := &notify.Recorder{}
	bus := events.NewBus(20)
	monitor := NewQuotaDropMonitor(rec, bus, 0.30, 0.05)
	if act := monitor.CheckAccount(lowQuotaAccount()); act == nil {
		t.Fatal("expected action without gate")
	}
	if len(rec.Payloads) != 1 || len(bus.GetHistory("", 10)) != 1 {
		t.Fatalf("expected 1 native + 1 bus announcement, got %d/%d", len(rec.Payloads), len(bus.GetHistory("", 10)))
	}
}

// When the new monitor claimed the condition first, the legacy loop stays
// silent on both channels (single dispatcher, no duplicates).
func TestDispatchGateSuppressesDuplicate(t *testing.T) {
	rec := &notify.Recorder{}
	bus := events.NewBus(20)
	monitor := NewQuotaDropMonitor(rec, bus, 0.30, 0.05)
	gate := capacitymonitor.NewCoordinator(nil)
	monitor.SetDispatchGate(gate)

	// The new monitor's claim for the same condition wins the cycle.
	if !gate.Claim(capacitymonitor.OwnerCapacityMonitor, capacitymonitor.DispatchKey{
		Provider: "claude", Profile: "work", Group: "", Window: "5h",
		Class: capacitymonitor.DispatchLow,
	}, time.Hour) {
		t.Fatal("setup claim must win")
	}
	monitor.CheckAccount(lowQuotaAccount())
	if len(rec.Payloads) != 0 || len(bus.GetHistory("", 10)) != 0 {
		t.Fatalf("expected silence on both channels, got %d/%d", len(rec.Payloads), len(bus.GetHistory("", 10)))
	}
}

// A different severity class is a different condition and still announces.
func TestDispatchGateClassSplit(t *testing.T) {
	rec := &notify.Recorder{}
	bus := events.NewBus(20)
	monitor := NewQuotaDropMonitor(rec, bus, 0.30, 0.05)
	gate := capacitymonitor.NewCoordinator(nil)
	monitor.SetDispatchGate(gate)

	gate.Claim(capacitymonitor.OwnerCapacityMonitor, capacitymonitor.DispatchKey{
		Provider: "claude", Profile: "work",
		Class: capacitymonitor.DispatchExhausted,
	}, time.Hour)
	if act := monitor.CheckAccount(lowQuotaAccount()); act == nil {
		t.Fatal("expected LOW action despite exhausted claim elsewhere")
	}
	if len(rec.Payloads) != 1 {
		t.Fatalf("expected 1 announcement, got %d", len(rec.Payloads))
	}
}

// The gate never silences this loop's own milestone sequence: successive
// step-downs keep announcing while the other loop stays quiet.
func TestDispatchGateOwnSequenceKept(t *testing.T) {
	rec := &notify.Recorder{}
	bus := events.NewBus(20)
	monitor := NewQuotaDropMonitor(rec, bus, 0.30, 0.05)
	monitor.SetDispatchGate(capacitymonitor.NewCoordinator(nil))

	acc := lowQuotaAccount() // 10%: crosses 20% milestone on first sight
	if act := monitor.CheckAccount(acc); act == nil {
		t.Fatal("expected first milestone action")
	}
	acc.QuotaRemaining = 0.04 // crosses 5% milestone next
	if qv := acc.QuotaView; qv != nil {
		for gi := range qv.ModelGroups {
			for wi := range qv.ModelGroups[gi].Windows {
				qv.ModelGroups[gi].Windows[wi].Remaining = 4
			}
		}
	}
	if act := monitor.CheckAccount(acc); act == nil {
		t.Fatal("expected second milestone action (own sequence, never self-suppressed)")
	}
	if len(rec.Payloads) != 2 || len(bus.GetHistory("", 10)) != 2 {
		t.Fatalf("expected 2 native + 2 bus announcements, got %d/%d", len(rec.Payloads), len(bus.GetHistory("", 10)))
	}
}
