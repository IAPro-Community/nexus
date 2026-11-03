package agentwire

import "testing"

// A yielded turn and a prompt ACK do not prove that background tasks or
// follow-up turns have settled. OMP v1 emits session_settled once truly idle.
func TestOMPSessionSettledAfterYieldAndDeduplicated(t *testing.T) {
	var signals []WireSignal
	b := &Bridge{
		busy: true,
		signals: func(s WireSignal) { signals = append(signals, s) },
	}
	b.piEvent("agent_end", nil)
	if !b.busy || len(signals) != 0 {
		t.Fatalf("agent_end incorrectly completed the session: busy=%v signals=%v", b.busy, signals)
	}
	b.piEvent("session_settled", nil)
	if b.busy || len(signals) != 1 || signals[0].State != "done" {
		t.Fatalf("OMP settlement not observed: busy=%v signals=%v", b.busy, signals)
	}
	b.piEvent("session_settled", nil)
	b.piEvent("agent_settled", nil)
	if len(signals) != 1 {
		t.Fatalf("duplicate terminal event was emitted: %v", signals)
	}
}

func TestLegacyPiAgentSettledStillRecognized(t *testing.T) {
	var signals []WireSignal
	b := &Bridge{
		busy: true,
		signals: func(s WireSignal) { signals = append(signals, s) },
	}
	b.piEvent("agent_settled", nil)
	if b.busy || len(signals) != 1 || signals[0].State != "done" {
		t.Fatalf("legacy Pi settlement regressed: busy=%v signals=%v", b.busy, signals)
	}
}
