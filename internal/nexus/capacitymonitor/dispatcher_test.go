package capacitymonitor

import (
	"testing"
	"time"
)

type stepClock struct{ now time.Time }

func (s *stepClock) Now() time.Time { return s.now }
func (s *stepClock) Advance(d time.Duration) {
	s.now = s.now.Add(d)
}

func TestCoordinatorCrossLoopSuppression(t *testing.T) {
	clock := &stepClock{now: time.Now()}
	c := NewCoordinator(clock)
	key := DispatchKey{Provider: "codex", Profile: "work", Group: "g", Window: "5h", Class: DispatchLow}
	if !c.Claim(OwnerCapacityMonitor, key, time.Hour) {
		t.Fatal("first claim must win")
	}
	if c.Claim(OwnerLegacyQuota, key, time.Hour) {
		t.Fatal("other loop within TTL must lose (single dispatcher)")
	}
}

func TestCoordinatorOwnRepeatsNeverSuppressed(t *testing.T) {
	clock := &stepClock{now: time.Now()}
	c := NewCoordinator(clock)
	key := DispatchKey{Provider: "codex", Profile: "work", Class: DispatchLow}
	for i := 0; i < 3; i++ {
		if !c.Claim(OwnerLegacyQuota, key, time.Hour) {
			t.Fatalf("own milestone sequence must never self-suppress (step %d)", i)
		}
	}
}

func TestCoordinatorExpiryReleases(t *testing.T) {
	clock := &stepClock{now: time.Now()}
	c := NewCoordinator(clock)
	key := DispatchKey{Provider: "codex", Profile: "work", Class: DispatchExhausted}
	if !c.Claim(OwnerCapacityMonitor, key, time.Minute) {
		t.Fatal("first claim must win")
	}
	clock.Advance(2 * time.Minute)
	if !c.Claim(OwnerLegacyQuota, key, time.Minute) {
		t.Fatal("claim after TTL expiry must win again")
	}
}

func TestCoordinatorClassSplit(t *testing.T) {
	clock := &stepClock{now: time.Now()}
	c := NewCoordinator(clock)
	low := DispatchKey{Provider: "p", Profile: "q", Class: DispatchLow}
	exh := DispatchKey{Provider: "p", Profile: "q", Class: DispatchExhausted}
	if !c.Claim(OwnerCapacityMonitor, low, time.Hour) || !c.Claim(OwnerLegacyQuota, exh, time.Hour) {
		t.Fatal("different classes must not suppress each other (state change re-alerts)")
	}
}

func TestCoordinatorOwnershipTakeover(t *testing.T) {
	clock := &stepClock{now: time.Now()}
	c := NewCoordinator(clock)
	key := DispatchKey{Provider: "p", Profile: "q", Class: DispatchLow}
	if !c.Claim(OwnerLegacyQuota, key, time.Hour) {
		t.Fatal("first claim must win")
	}
	// Same owner re-claims (its own repeat policy) without expiry.
	if !c.Claim(OwnerLegacyQuota, key, time.Hour) {
		t.Fatal("same owner repeat must win")
	}
	// The other loop takes over only after expiry.
	clock.Advance(2 * time.Hour)
	if !c.Claim(OwnerCapacityMonitor, key, time.Hour) {
		t.Fatal("other owner after expiry must win")
	}
}

func TestCoordinatorNilAllows(t *testing.T) {
	var c *Coordinator
	if !c.Claim(OwnerLegacyQuota, DispatchKey{Provider: "p", Class: DispatchLow}, time.Hour) {
		t.Fatal("nil coordinator must allow (legacy behavior preserved)")
	}
	c.Forget(DispatchKey{})
}

func TestClassForState(t *testing.T) {
	if ClassForState(StateWarning) != DispatchLow || ClassForState(StateCritical) != DispatchLow {
		t.Fatal("warning/critical must collapse to low")
	}
	if ClassForState(StateExhausted) != DispatchExhausted {
		t.Fatal("exhausted mapping")
	}
	if ClassForState(StateRecovered) != DispatchRecovered {
		t.Fatal("recovered mapping")
	}
	if ClassForState(StateHealthy) != "" || ClassForState(StateUnknown) != "" {
		t.Fatal("healthy/unknown must never claim")
	}
}
