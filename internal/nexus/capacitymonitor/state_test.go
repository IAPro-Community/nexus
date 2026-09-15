package capacitymonitor

import (
	"strings"
	"testing"
	"time"
)

func TestEvaluateTransitionTable(t *testing.T) {
	now := time.Now().UTC()
	obs := now.Add(-time.Minute)
	fresh := now.Add(time.Hour)
	f64 := func(v float64) *float64 { return &v }
	cases := []struct {
		name      string
		prev      State
		remaining *float64
		limited   bool
		observed  time.Time
		expires   time.Time
		want      State
	}{
		{"nil observation", StateHealthy, nil, false, obs, fresh, StateUnknown},
		{"zero observedAt", StateHealthy, f64(80), false, time.Time{}, fresh, StateUnknown},
		{"expired", StateHealthy, f64(80), false, obs, now.Add(-time.Second), StateUnknown},
		{"exhausted at zero", StateHealthy, f64(0), false, obs, fresh, StateExhausted},
		{"rate limited at 80", StateHealthy, f64(80), true, obs, fresh, StateExhausted},
		{"critical", StateHealthy, f64(5), false, obs, fresh, StateCritical},
		{"warning", StateHealthy, f64(15), false, obs, fresh, StateWarning},
		{"healthy", StateHealthy, f64(80), false, obs, fresh, StateHealthy},
		{"recovered past threshold", StateWarning, f64(30), false, obs, fresh, StateRecovered},
		{"recovered from critical", StateCritical, f64(90), false, obs, fresh, StateRecovered},
		{"between warning and recovery is healthy", StateWarning, f64(22), false, obs, fresh, StateHealthy},
		{"healthy stays healthy", StateHealthy, f64(30), false, obs, fresh, StateHealthy},
		{"unknown previous still evaluates", StateUnknown, f64(15), false, obs, fresh, StateWarning},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason := EvaluateTransition(tc.prev, tc.remaining, tc.limited, tc.observed, tc.expires, now, DefaultThresholds())
			if got != tc.want {
				t.Fatalf("state = %s, want %s (reason: %s)", got, tc.want, reason)
			}
			if strings.TrimSpace(reason) == "" {
				t.Fatal("reason must never be empty")
			}
		})
	}
}

func TestThresholdsValidate(t *testing.T) {
	if err := DefaultThresholds().Validate(); err != nil {
		t.Fatalf("defaults invalid: %v", err)
	}
	bad := []Thresholds{
		{Warning: 20, Critical: 10, Exhausted: 10, Recovery: 25}, // exhausted !< critical
		{Warning: 10, Critical: 10, Exhausted: 0, Recovery: 25},  // critical !< warning
		{Warning: 30, Critical: 10, Exhausted: 0, Recovery: 30},  // warning !< recovery
		{Warning: 20, Critical: 10, Exhausted: 0, Recovery: 101}, // recovery > 100
	}
	for i, th := range bad {
		if err := th.Validate(); err == nil {
			t.Fatalf("case %d: expected validation error, got nil", i)
		}
	}
}

func TestDeriveFingerprintStableAndScoped(t *testing.T) {
	a := DeriveFingerprint("codex", "kiver", "claude_gpt", "5h", "CRITICAL", "")
	b := DeriveFingerprint("codex", "kiver", "claude_gpt", "5h", "CRITICAL", "")
	if a != b {
		t.Fatal("same inputs must produce the same fingerprint")
	}
	if !strings.HasPrefix(a, "alert-") {
		t.Fatalf("fingerprint = %q, want alert- prefix", a)
	}
	if c := DeriveFingerprint("codex", "kiver", "claude_gpt", "5h", "WARNING", ""); a == c {
		t.Fatal("different severity must produce a different fingerprint")
	}
}
