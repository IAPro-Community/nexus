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
