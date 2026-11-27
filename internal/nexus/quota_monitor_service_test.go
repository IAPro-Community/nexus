package nexus

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/profile"
)

func TestQuotaMonitorLeaseAllowsOneCollectorAndRecovers(t *testing.T) {
	leasePath := filepath.Join(t.TempDir(), "quota-monitor.lease")
	first := NewQuotaMonitorService(nil, nil)
	second := NewQuotaMonitorService(nil, nil)
	first.leasePath = leasePath
	second.leasePath = leasePath
	first.leaseTTL = time.Minute
	second.leaseTTL = time.Minute
	if first.leaseToken == second.leaseToken {
		t.Fatal("lease tokens must be unique within one process")
	}

	if !first.acquireOrRenewLease() {
		t.Fatal("first collector should acquire the lease")
	}
	if second.acquireOrRenewLease() {
		t.Fatal("second collector must not collect while the lease is fresh")
	}

	first.releaseLease()
	if !second.acquireOrRenewLease() {
		t.Fatal("second collector should recover after leader release")
	}
	second.releaseLease()
}

func TestQuotaMonitorLeaseExpiresAfterLeaderCrash(t *testing.T) {
	leasePath := filepath.Join(t.TempDir(), "quota-monitor.lease")
	leader := NewQuotaMonitorService(nil, nil)
	follower := NewQuotaMonitorService(nil, nil)
	leader.leasePath = leasePath
	follower.leasePath = leasePath
	leader.leaseTTL = time.Millisecond
	follower.leaseTTL = time.Millisecond

	if !leader.acquireOrRenewLease() {
		t.Fatal("leader should acquire the lease")
	}
	time.Sleep(10 * time.Millisecond)
	if !follower.acquireOrRenewLease() {
		t.Fatal("follower should recover an expired lease")
	}
	follower.releaseLease()
}

func TestIsFreshIdleMonitorSnapshot(t *testing.T) {
	now := time.Now()
	rem := 70.0
	freshLive := model.UsageSnapshot{
		Status:    model.UsageLive,
		FetchedAt: now.Add(-2 * time.Minute),
		Windows:   []model.UsageWindow{{Kind: "session", Remaining: &rem}},
	}
	if !isFreshIdleMonitorSnapshot(freshLive, now) {
		t.Fatal("expected fresh live snapshot with windows to be reused when idle")
	}

	staleLive := freshLive
	staleLive.FetchedAt = now.Add(-10 * time.Minute)
	if isFreshIdleMonitorSnapshot(staleLive, now) {
		t.Fatal("expected stale snapshot beyond DefaultTTL to require refresh")
	}

	unknown := freshLive
	unknown.Status = model.UsageUnknown
	if isFreshIdleMonitorSnapshot(unknown, now) {
		t.Fatal("expected UNKNOWN snapshot not to skip refresh")
	}

	rateLimited := model.UsageSnapshot{
		Status:    model.UsageRateLimited,
		FetchedAt: now.Add(-time.Minute),
	}
	if !isFreshIdleMonitorSnapshot(rateLimited, now) {
		t.Fatal("expected fresh RATE_LIMITED snapshot to be reused when idle")
	}
}

func TestQuotaMonitorServiceSkipsUnsupportedProviders(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("NEXUS_REAL_HOME", "")

	bus := events.NewBus(16)
	svc := NewQuotaMonitorService(NewQuotaDropMonitor(nil, bus, 0.30, 0.05), bus)
	svc.mu.Lock()
	svc.degraded["cursor:default"] = true
	svc.mu.Unlock()

	if _, err := profile.Create("cursor", "default"); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	svc.check()

	svc.mu.Lock()
	stillDegraded := svc.degraded["cursor:default"]
	svc.mu.Unlock()
	if stillDegraded {
		t.Fatal("expected QuotaUnsupported provider degraded flag to clear without emitting degraded error")
	}
	for _, ev := range bus.GetHistory("", 10) {
		if ev.Type == events.EventQuotaMonitorDegraded && strings.Contains(fmt.Sprint(ev.Data["profile_key"]), "cursor:") {
			t.Fatalf("unexpected degraded event for QuotaUnsupported provider: %+v", ev)
		}
	}
}
