package agentwire

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBridgeACPStartAndPrompt(t *testing.T) {
	// A fake ACP agent responds to the same JSONL messages as real clients.
	providerRead, clientWrite := io.Pipe()
	clientRead, providerWrite := io.Pipe()
	defer providerRead.Close()
	defer clientWrite.Close()
	defer clientRead.Close()
	defer providerWrite.Close()
	var outputMu sync.Mutex
	var output strings.Builder
	bridge, err := NewBridge("acp", clientRead, clientWrite, "/workspace/demo", nil, func(s string) { outputMu.Lock(); defer outputMu.Unlock(); output.WriteString(s) })
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		wire := NewJSONLines(providerRead, providerWrite)
		for {
			req, err := wire.Read()
			if err != nil {
				return
			}
			var v struct {
				ID     string `json:"id"`
				Method string `json:"method"`
			}
			if json.Unmarshal(req, &v) != nil {
				return
			}
			var result any
			switch v.Method {
			case "initialize":
				result = map[string]any{"protocolVersion": 1, "agentCapabilities": map[string]any{}}
			case "session/new":
				result = map[string]string{"sessionId": "sess-1"}
			case "session/prompt":
				_ = wire.Write(map[string]any{"jsonrpc": "2.0", "method": "session/update", "params": map[string]any{
					"sessionId": "sess-1",
					"update":    map[string]any{"sessionUpdate": "agent_message_chunk", "content": map[string]string{"text": "hello from provider"}},
				}})
				result = map[string]string{"stopReason": "end_turn"}
			default:
				result = map[string]any{}
			}
			_ = wire.Write(map[string]any{"jsonrpc": "2.0", "id": v.ID, "result": result})
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := bridge.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if bridge.SessionID() != "sess-1" {
		t.Fatal("provider session id not retained")
	}
	if err := bridge.Prompt(ctx, "test"); err != nil {
		t.Fatal(err)
	}
	outputMu.Lock()
	got := output.String()
	outputMu.Unlock()
	if !strings.Contains(got, "hello from provider") {
		t.Fatalf("missing provider update: %s", got)
	}
	bridge.Stop()
}

func TestBridgePermissionRequiresExactOption(t *testing.T) {
	b := &Bridge{pending: map[string]*permissionWait{"req-1": {
		options:  map[string]bool{"allow_once": true, "reject_once": true},
		selected: make(chan string, 1),
	}}}
	if err := b.Choose("req-1", "admin_override"); err == nil {
		t.Fatal("unadvertised grant accepted")
	}
	if err := b.Choose("not-real", "allow_once"); err == nil {
		t.Fatal("unknown request accepted")
	}
	if err := b.Choose("req-1", "reject_once"); err != nil {
		t.Fatal(err)
	}
	if err := b.Choose("req-1", "allow_once"); err == nil {
		t.Fatal("repeat permission accepted")
	}
	if got := <-b.pending["req-1"].selected; got != "reject_once" {
		t.Fatalf("response changed: %s", got)
	}
}

func TestBridgePiSettledNotAck(t *testing.T) {
	providerRead, clientWrite := io.Pipe()
	clientRead, providerWrite := io.Pipe()
	defer providerRead.Close()
	defer clientWrite.Close()
	defer clientRead.Close()
	defer providerWrite.Close()
	signals := make(chan WireSignal, 8)
	bridge, err := NewBridge("pi_rpc", clientRead, clientWrite, "/workspace/demo", func(s WireSignal) { signals <- s }, nil)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		wire := NewJSONLines(providerRead, providerWrite)
		for {
			msg, err := wire.Read()
			if err != nil {
				return
			}
			var p struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			}
			_ = json.Unmarshal(msg, &p)
			if p.Type == "get_state" {
				_ = wire.Write(map[string]any{"type": "response", "id": p.ID, "command": "get_state", "success": true,
					"data": map[string]string{"sessionId": "pi-session-1"}})
				continue
			}
			_ = wire.Write(map[string]any{"type": "response", "id": p.ID, "command": "prompt", "success": true})
			_ = wire.Write(map[string]any{"type": "agent_end"})
			_ = wire.Write(map[string]any{"type": "agent_settled"})
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := bridge.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := bridge.Prompt(ctx, "run tests"); err != nil {
		t.Fatal(err)
	}
	settled := false
	for i := 0; i < 5; i++ {
		select {
		case v := <-signals:
			if v.State == "done" {
				settled = true
			}
		case <-time.After(time.Second):
			t.Fatal("missing provider event")
		}
		if settled {
			break
		}
	}
	if !settled {
		t.Fatal("agent_settled not forwarded")
	}
	bridge.Stop()
}

func TestPiExtensionUIChoiceIsCorrelated(t *testing.T) {
	upstream, downstream := io.Pipe()
	clientInput, agentOutput := io.Pipe()
	defer upstream.Close()
	defer downstream.Close()
	defer clientInput.Close()
	defer agentOutput.Close()
	signal := make(chan WireSignal, 1)
	bridge, err := NewBridge("pi_rpc", clientInput, downstream, "/workspace/demo", func(s WireSignal) {
		if s.State == "blocked" {
			signal <- s
		}
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	bridge.ctx, bridge.cancel = context.WithCancel(context.Background())
	defer bridge.cancel()
	request := []byte(`{"type":"extension_ui_request","id":"provider-uuid","method":"select","title":"Pick policy","options":["Deny","Allow once"]}`)
	bridge.piEvent("extension_ui_request", request)
	var ask WireSignal
	select {
	case ask = <-signal:
	case <-time.After(time.Second):
		t.Fatal("dialog was not observed")
	}
	if ask.InteractionID == "" || len(ask.Options) != 2 {
		t.Fatalf("bad dialog %+v", ask)
	}
	if err := bridge.Choose(ask.InteractionID, "Arbitrary grant"); err == nil {
		t.Fatal("forged option accepted")
	}
	if err := bridge.Choose(ask.InteractionID, "Deny"); err != nil {
		t.Fatal(err)
	}
	done := make(chan json.RawMessage, 1)
	go func() { wire := NewJSONLines(upstream, io.Discard); response, _ := wire.Read(); done <- response }()
	select {
	case result := <-done:
		var response struct {
			ID    string `json:"id"`
			Type  string `json:"type"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal(result, &response); err != nil || response.ID != "provider-uuid" ||
			response.Type != "extension_ui_response" || response.Value != "Deny" {
			t.Fatalf("bad response %s %v", result, err)
		}
	case <-time.After(time.Second):
		t.Fatal("provider never received correlated response")
	}
}

func TestBridgeACPHandshakeTimeout(t *testing.T) {
	providerRead, clientWrite := io.Pipe()
	clientRead, providerWrite := io.Pipe()
	defer providerRead.Close()
	defer clientWrite.Close()
	defer clientRead.Close()
	defer providerWrite.Close()

	// Drain providerRead so clientWrite does not block inside pipe write
	go func() { _, _ = io.Copy(io.Discard, providerRead) }()

	bridge, err := NewBridge("acp", clientRead, clientWrite, "/workspace/demo", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = bridge.Start(ctx)
	if err == nil {
		t.Fatal("expected handshake timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestBridgeConcurrentPermissionRequests(t *testing.T) {
	upstream, downstream := io.Pipe()
	clientInput, agentOutput := io.Pipe()
	defer upstream.Close()
	defer downstream.Close()
	defer clientInput.Close()
	defer agentOutput.Close()

	signals := make(chan WireSignal, 4)
	bridge, err := NewBridge("pi_rpc", clientInput, downstream, "/workspace/demo", func(s WireSignal) {
		if s.State == "blocked" {
			signals <- s
		}
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	bridge.ctx, bridge.cancel = context.WithCancel(context.Background())
	defer bridge.cancel()

	// Emit two distinct UI requests
	req1 := []byte(`{"type":"extension_ui_request","id":"req-alpha","method":"confirm","title":"Format disk?"}`)
	req2 := []byte(`{"type":"extension_ui_request","id":"req-beta","method":"select","title":"Choose color","options":["Red","Blue"]}`)

	bridge.piEvent("extension_ui_request", req1)
	bridge.piEvent("extension_ui_request", req2)

	var sig1, sig2 WireSignal
	select {
	case sig1 = <-signals:
	case <-time.After(time.Second):
		t.Fatal("first dialog not observed")
	}
	select {
	case sig2 = <-signals:
	case <-time.After(time.Second):
		t.Fatal("second dialog not observed")
	}

	if sig1.InteractionID == sig2.InteractionID {
		t.Fatalf("interaction IDs collided: %s == %s", sig1.InteractionID, sig2.InteractionID)
	}

	// Submit response to sig1 only
	if err := bridge.Choose(sig1.InteractionID, "yes"); err != nil {
		t.Fatalf("failed choosing yes for sig1: %v", err)
	}

	// Verify sig1 cannot be chosen again
	if err := bridge.Choose(sig1.InteractionID, "no"); err == nil {
		t.Fatal("expected repeat choose on sig1 to fail")
	}

	// Verify sig2 is still pending and validates options
	if err := bridge.Choose(sig2.InteractionID, "Green"); err == nil {
		t.Fatal("expected invalid option on sig2 to fail")
	}
	if err := bridge.Choose(sig2.InteractionID, "Red"); err != nil {
		t.Fatalf("failed choosing Red for sig2: %v", err)
	}
}

func TestBridgeCancellationDuringPermission(t *testing.T) {
	upstream, downstream := io.Pipe()
	clientInput, agentOutput := io.Pipe()
	defer upstream.Close()
	defer downstream.Close()
	defer clientInput.Close()
	defer agentOutput.Close()

	signal := make(chan WireSignal, 1)
	bridge, err := NewBridge("pi_rpc", clientInput, downstream, "/workspace/demo", func(s WireSignal) {
		if s.State == "blocked" {
			signal <- s
		}
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	bridge.ctx, bridge.cancel = context.WithCancel(context.Background())

	req := []byte(`{"type":"extension_ui_request","id":"req-cancel-test","method":"confirm","title":"Proceed?"}`)
	bridge.piEvent("extension_ui_request", req)

	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("dialog not received")
	}

	// Reader waiting for cancel response
	respCh := make(chan json.RawMessage, 1)
	go func() {
		wire := NewJSONLines(upstream, io.Discard)
		raw, _ := wire.Read()
		respCh <- raw
	}()

	// Stop bridge (triggers context cancellation)
	bridge.Stop()

	select {
	case raw := <-respCh:
		var resp struct {
			ID        string `json:"id"`
			Type      string `json:"type"`
			Cancelled bool   `json:"cancelled"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil || resp.ID != "req-cancel-test" || !resp.Cancelled {
			t.Fatalf("expected cancelled response, got: %s", string(raw))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled response never received")
	}
}
