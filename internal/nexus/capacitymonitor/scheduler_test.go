package capacitymonitor

import (
	"math"
	"testing"
	"time"
)

func TestDetermineCollectionInterval(t *testing.T) {
	cases := []struct {
		name     string
		state    State
		active   bool
		failures int
		lowPower bool
		floor    time.Duration
		want     time.Duration
	}{
		{"healthy idle", StateHealthy, false, 0, false, 0, 10 * time.Minute},
		{"unknown idle", StateUnknown, false, 0, false, 0, 10 * time.Minute},
		{"exhausted stays slow", StateExhausted, false, 0, false, 0, 10 * time.Minute},
		{"active host polls faster", StateHealthy, true, 0, false, 0, 60 * time.Second},
		{"warning polls fast", StateWarning, false, 0, false, 0, 45 * time.Second},
		{"critical polls fast", StateCritical, false, 0, false, 0, 45 * time.Second},
		{"one failure", StateHealthy, false, 1, false, 0, time.Minute},
		{"two failures", StateHealthy, false, 2, false, 0, 2 * time.Minute},
		{"three failures", StateHealthy, false, 3, false, 0, 5 * time.Minute},
		{"many failures cap", StateHealthy, false, 9, false, 0, 15 * time.Minute},
		{"low power wins over warning", StateWarning, false, 0, true, 0, 5 * time.Minute},
		{"probe floor respected", StateHealthy, false, 0, false, 30 * time.Minute, 30 * time.Minute},
		{"failures beat floor only upward", StateWarning, false, 5, false, 30 * time.Minute, 30 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetermineCollectionInterval(tc.state, tc.active, tc.lowPower, tc.failures, tc.floor); got != tc.want {
				t.Fatalf("interval = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRealJitterBounded(t *testing.T) {
	j := RealJitter{}
	base := 10 * time.Minute
	for i := 0; i < 25; i++ {
		got := j.Duration(base)
		if got < base || got > base+base/10+time.Second {
			t.Fatalf("jittered = %v, want within [base, base+10%%+slack]", got)
		}
	}
	// Sub-millisecond bases keep the 0-10% bound instead of exploding.
	// The bound is exact (no slack): n < base/10 always.
	small := 5 * time.Millisecond
	for i := 0; i < 25; i++ {
		got := j.Duration(small)
		if got < small || got > small+small/10 {
			t.Fatalf("small jittered = %v, want within [base, base+10%%] exactly", got)
		}
	}
	if got := j.Duration(0); got != 0 {
		t.Fatalf("zero base = %v, want 0", got)
	}
}

func TestRealJitterSaturatesNearMaxInt64(t *testing.T) {
	got := RealJitter{}.Duration(time.Duration(math.MaxInt64))
	if got != time.Duration(math.MaxInt64) {
		t.Fatalf("saturated = %v, want MaxInt64 (must never wrap negative)", got)
	}
}
