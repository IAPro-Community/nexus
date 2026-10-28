package agentwire

import "testing"

func TestWireProjectionDoesNotResolveQuestionFromUncorrelatedProgress(t *testing.T) {
	r := NewReducer()
	ask := WireSignal{State: "blocked", Kind: "permission", Summary: "Proceed?", InteractionID: "req-1"}
	state, err := r.ObserveWireSignal("runtime-a", 10, 1, ask)
	if err != nil || len(state.Attention) != 1 {
		t.Fatalf("missing request: %+v %v", state, err)
	}
	state, err = r.ObserveWireSignal("runtime-a", 10, 2, WireSignal{State: "working"})
	if err != nil || len(state.Attention) != 1 {
		t.Fatalf("non-authoritative progress resolved approval: %+v %v", state, err)
	}
	state, err = r.ObserveWireSignal("runtime-a", 10, 3, WireSignal{State: "done"})
	if err != nil || len(state.Attention) != 0 || state.Turn != TurnSettled {
		t.Fatalf("authoritative settled did not resolve approval: %+v %v", state, err)
	}
}
func TestWireProjectionRuntimeAndEpochIsolation(t *testing.T) {
	r := NewReducer()
	for _, id := range []string{"alpha", "beta"} {
		_, err := r.ObserveWireSignal(id, 20, 1, WireSignal{State: "blocked", Kind: "question", Summary: "Same?", InteractionID: "ui-1"})
		if err != nil {
			t.Fatal(err)
		}
	}
	a, _ := r.ObserveWireSignal("alpha", 20, 2, WireSignal{State: "done"})
	b, ok := r.Get("beta")
	if !ok || len(a.Attention) != 0 || len(b.Attention) != 1 {
		t.Fatalf("runtime boundary lost: a=%+v b=%+v", a, b)
	}
	after, _ := r.ObserveWireSignal("beta", 21, 1, WireSignal{State: "idle"})
	if len(after.Attention) != 0 || after.Epoch != 21 {
		t.Fatalf("old generation alert survived: %+v", after)
	}
	stale, _ := r.ObserveWireSignal("beta", 20, 99, WireSignal{State: "blocked", Kind: "question", InteractionID: "stale"})
	if len(stale.Attention) != 0 {
		t.Fatal("stale generation request accepted")
	}
}

func TestWireProjectionCancelClosesPendingWithoutPretendingCompletion(t *testing.T) {
	r := NewReducer()
	_, err := r.ObserveWireSignal("runtime-cancel", 1, 1, WireSignal{
		State: "blocked", Kind: "question", Summary: "Text?", InteractionID: "ui-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	final, err := r.ObserveWireSignal("runtime-cancel", 1, 2, WireSignal{State: "cancelled"})
	if err != nil {
		t.Fatal(err)
	}
	if final.Turn != TurnCancelled || len(final.Attention) != 0 {
		t.Fatalf("cancellation was misrepresented: %+v", final)
	}
}
