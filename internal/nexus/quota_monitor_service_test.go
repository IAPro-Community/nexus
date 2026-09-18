package nexus

import (
	"path/filepath"
	"testing"
	"time"
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
