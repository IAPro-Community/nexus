package nexus

import "testing"

func TestEvaluateContextReadinessStates(t *testing.T) {
	fp := &ContextFingerprint{ProjectID: "p1", Branch: "main", Head: "abc", MaestroVersion: "1"}
	cases := []struct {
		name, want, failure string
		previous, current   *ContextFingerprint
		hydrating           bool
	}{
		{name: "missing", want: ContextMissing, current: fp},
		{name: "hydrating", want: ContextHydrating, current: fp, hydrating: true},
		{name: "ready", want: ContextReady, previous: fp, current: fp},
		{name: "stale branch", want: ContextStale, previous: fp, current: &ContextFingerprint{ProjectID: "p1", Branch: "feature", Head: "abc", MaestroVersion: "1"}},
		{name: "failed", want: ContextFailed, current: fp, failure: "maestro unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateContextReadiness(tc.previous, tc.current, tc.hydrating, tc.failure)
			if got.State != tc.want {
				t.Fatalf("state=%q, want %q", got.State, tc.want)
			}
		})
	}
}

func TestEvaluateContextReadinessDetectsDirtyAndMaestroDrift(t *testing.T) {
	previous := &ContextFingerprint{ProjectID: "p1", Branch: "main", Head: "abc", DirtyFingerprint: "clean", MaestroVersion: "1"}
	current := *previous
	current.DirtyFingerprint = "modified"
	if got := EvaluateContextReadiness(previous, &current, false, ""); got.State != ContextStale {
		t.Fatalf("dirty drift state=%q, want %q", got.State, ContextStale)
	}
	current = *previous
	current.MaestroVersion = "2"
	if got := EvaluateContextReadiness(previous, &current, false, ""); got.State != ContextStale {
		t.Fatalf("Maestro drift state=%q, want %q", got.State, ContextStale)
	}
}
