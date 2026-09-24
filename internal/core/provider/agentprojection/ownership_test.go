package agentprojection

import "testing"

func TestResolveOwnershipFailsClosed(t *testing.T) {
	cases := []struct {
		name     string
		exists   bool
		managed  bool
		observed string
		receipt  string
		state    OwnershipState
		decision OwnershipDecision
	}{
		{"missing", false, false, "", "", OwnershipMissing, DecisionCreate},
		{"managed unchanged", true, true, "abc", "abc", OwnershipManagedUnmodified, DecisionUpdate},
		{"managed modified", true, true, "changed", "abc", OwnershipManagedModified, DecisionConflict},
		{"managed unknown hash", true, true, "", "abc", OwnershipManagedModified, DecisionConflict},
		{"unmanaged", true, false, "abc", "", OwnershipUnmanaged, DecisionConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state, decision := ResolveOwnership(tc.exists, tc.managed, tc.observed, tc.receipt)
			if state != tc.state || decision != tc.decision {
				t.Fatalf("got (%s,%s), want (%s,%s)", state, decision, tc.state, tc.decision)
			}
		})
	}
}

func TestContentHashIsDeterministic(t *testing.T) {
	a := ContentHash([]byte("nexus"))
	b := ContentHash([]byte("nexus"))
	c := ContentHash([]byte("changed"))
	if a == "" || a != b || a == c {
		t.Fatalf("unexpected hashes: a=%q b=%q c=%q", a, b, c)
	}
}
