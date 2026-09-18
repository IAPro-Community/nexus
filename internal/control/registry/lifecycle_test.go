package registry

import "testing"

func TestRuntimeStateTransitions(t *testing.T) {
	valid := []struct {
		from RuntimeState
		to   RuntimeState
	}{
		{StateStarting, StateRunning},
		{StateStarting, StateFailed},
		{StateRunning, StateWaiting},
		{StateRunning, StateStopping},
		{StateRunning, StateFailed},
		{StateStopping, StateStopped},
		{StateStopping, StateFailed},
		{StateStopped, StateStarting},
		{StateFailed, StateStarting},
	}
	for _, test := range valid {
		if !CanTransition(test.from, test.to) {
			t.Errorf("CanTransition(%q, %q) = false, want true", test.from, test.to)
		}
	}

	invalid := []struct {
		from RuntimeState
		to   RuntimeState
	}{
		{StateStarting, StateStopped},
		{StateStopped, StateRunning},
		{StateFailed, StateRunning},
		{StateStopping, StateWaiting},
	}
	for _, test := range invalid {
		if CanTransition(test.from, test.to) {
			t.Errorf("CanTransition(%q, %q) = true, want false", test.from, test.to)
		}
	}
}

func TestCanTransitionAllowsIdempotentStateUpdate(t *testing.T) {
	if !CanTransition(StateRunning, StateRunning) {
		t.Fatal("same-state update must be idempotent")
	}
}
