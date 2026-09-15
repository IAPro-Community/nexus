package capacitymonitor

import (
	"testing"
	"time"
)

func TestDetermineCollectionInterval(t *testing.T) {
	cases := []struct {
		name     string
		state    State
		failures int
		lowPower bool
		floor    time.Duration
		want     time.Duration
	}{
		{"healthy idle", StateHealthy, 0, false, 0, 10 * time.Minute},
		{"unknown idle", StateUnknown, 0, false, 0, 10 * time.Minute},
		{"warning polls fast", StateWarning, 0, false, 0, 45 * time.Second},
		{"critical polls fast", StateCritical, 0, false, 0, 45 * time.Second},
		{"one failure", StateHealthy, 1, false, 0, time.Minute},
		{"two failures", StateHealthy, 2, false, 0, 2 * time.Minute},
		{"three failures", StateHealthy, 3, false, 0, 5 * time.Minute},
		{"many failures cap", StateHealthy, 9, false, 0, 15 * time.Minute},
		{"low power wins over warning", StateWarning, 0, true, 0, 5 * time.Minute},
		{"probe floor respected", StateHealthy, 0, false, 30 * time.Minute, 30 * time.Minute},
		{"failures beat floor only upward", StateWarning, 5, false, 30 * time.Minute, 30 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetermineCollectionInterval(tc.state, false, tc.lowPower, tc.failures, tc.floor); got != tc.want {
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
	if got := j.Duration(0); got != 0 {
		t.Fatalf("zero base = %v, want 0", got)
	}
}
