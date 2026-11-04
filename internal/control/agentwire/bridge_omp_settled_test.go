package agentwire

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"
)

// A yielded turn and a prompt ACK do not prove that background tasks or
// follow-up turns have settled. OMP v1 emits session_settled once truly idle.
func TestOMPSessionSettledAfterYieldAndDeduplicated(t *testing.T) {
	var signals []WireSignal
	b := &Bridge{
		busy:     true,
		promptID: "req-1",
		signals:  func(s WireSignal) { signals = append(signals, s) },
	}
	b.piEvent("agent_end", nil)
	if !b.busy || len(signals) != 0 {
		t.Fatalf("agent_end incorrectly completed the session: busy=%v signals=%v", b.busy, signals)
	}
	b.piEvent("prompt_result", []byte(`{"id":"req-1","agentInvoked":true,"status":"completed","sessionSettled":false}`))
	if !b.busy || len(signals) != 0 {
		t.Fatalf("yield was mistaken for session settlement: busy=%v signals=%v", b.busy, signals)
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
		busy:    true,
		signals: func(s WireSignal) { signals = append(signals, s) },
	}
	b.piEvent("agent_settled", nil)
	if b.busy || len(signals) != 1 || signals[0].State != "done" {
		t.Fatalf("legacy Pi settlement regressed: busy=%v signals=%v", b.busy, signals)
	}
}

func TestOMPLocalOnlyPromptDoesNotRequireSettlementEvent(t *testing.T) {
	providerRead, clientWrite := io.Pipe()
	clientRead, providerWrite := io.Pipe()
	defer providerRead.Close()
	defer clientWrite.Close()
	defer clientRead.Close()
	defer providerWrite.Close()

	b, err := NewBridge("pi_rpc", clientRead, clientWrite, "/workspace/demo", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		wire := NewJSONLines(providerRead, providerWrite)
		for {
			raw, err := wire.Read()
			if err != nil {
				return
			}
			var request struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			}
			if json.Unmarshal(raw, &request) != nil {
				return
			}
			response := map[string]any{
				"type": "response", "id": request.ID,
				"command": request.Type, "success": true,
			}
			if request.Type == "get_state" {
				response["data"] = map[string]string{"sessionId": "omp-local"}
			} else if request.Type == "prompt" {
				response["data"] = map[string]bool{"agentInvoked": false}
			}
			if wire.Write(response) != nil {
				return
			}
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := b.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer b.Stop()
	for i := 0; i < 2; i++ {
		if err := b.Prompt(ctx, "/help"); err != nil {
			t.Fatalf("local-only prompt %d rejected: %v", i, err)
		}
		b.mu.Lock()
		busy := b.busy
		b.mu.Unlock()
		if busy {
			t.Fatalf("local-only prompt %d left session busy without any run", i)
		}
	}
}

func TestOMPPromptResultErrorNeverBecomesDone(t *testing.T) {
	var events []WireSignal
	b := &Bridge{
		busy: true, promptID: "prompt-a",
		signals: func(s WireSignal) { events = append(events, s) },
	}
	b.piEvent("prompt_result", []byte(`{"id":"prompt-a","agentInvoked":true,"status":"error","sessionSettled":false}`))
	if !b.busy || len(events) != 0 {
		t.Fatalf("async work should remain active until settled: %+v", events)
	}
	b.piEvent("agent_end", nil)
	b.piEvent("session_settled", nil)
	if b.busy || len(events) != 1 || events[0].State != "error" ||
		events[0].Kind != "prompt_result" {
		t.Fatalf("failed prompt was reported as success: %+v", events)
	}
	b.piEvent("session_settled", nil)
	if len(events) != 1 {
		t.Fatalf("duplicate settlement emitted an extra result: %+v", events)
	}
}

func TestOMPPromptResultAbortedImmediateAndCorrelated(t *testing.T) {
	var events []WireSignal
	b := &Bridge{
		busy: true, promptID: "prompt-current",
		signals: func(s WireSignal) { events = append(events, s) },
	}
	for _, payload := range []string{
		`{"id":"prompt-old","agentInvoked":true,"status":"completed","sessionSettled":true}`,
		`{"id":"","agentInvoked":true,"status":"completed","sessionSettled":true}`,
		`{"id":"prompt-current","agentInvoked":true,"status":"unknown","sessionSettled":true}`,
		`{"id":"prompt-current","agentInvoked":true,"status":"completed"}`,
		`{"id":"prompt-current","agentInvoked":"true","status":"completed","sessionSettled":true}`,
	} {
		b.piEvent("prompt_result", []byte(payload))
	}
	if !b.busy || len(events) != 0 || b.promptResultSeen {
		t.Fatalf("untrusted prompt result changed active state: %+v", events)
	}
	b.piEvent("prompt_result", []byte(`{"id":"prompt-current","agentInvoked":true,"status":"aborted","sessionSettled":true}`))
	if b.busy || len(events) != 1 || events[0].State != "cancelled" {
		t.Fatalf("correlated aborted prompt not observed: %+v", events)
	}
	b.piEvent("session_settled", nil)
	if len(events) != 1 {
		t.Fatalf("late session_settled duplicated cancellation: %+v", events)
	}
}

func TestOMPLocalPromptResultAndUnknownSettlementFailClosed(t *testing.T) {
	var events []WireSignal
	b := &Bridge{
		busy: true, promptID: "local",
		signals: func(s WireSignal) { events = append(events, s) },
	}
	b.piEvent("prompt_result", []byte(`{"id":"local","agentInvoked":false,"status":"completed","sessionSettled":false}`))
	if b.busy || len(events) != 1 || events[0].State != "idle" {
		t.Fatalf("local prompt incorrectly required session_settled: %+v", events)
	}
	b.busy, b.promptID = true, "unverified"
	b.piEvent("session_settled", nil)
	if b.busy || len(events) != 2 || events[1].State != "idle" {
		t.Fatalf("missing correlated outcome fabricated DONE: %+v", events)
	}
}
