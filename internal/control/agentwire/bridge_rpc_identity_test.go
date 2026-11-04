package agentwire

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

// A successful RPC command without a valid provider session cannot establish
// identity or authorize the structured backend to advertise READY.
func TestPiRPCHandshakeRejectsUnverifiedSessionIdentity(t *testing.T) {
	cases := []struct {
		name    string
		state   any
		wantID  string
		wantErr bool
	}{
		{name: "valid legacy Pi identity", state: map[string]any{"sessionId": "pi-session-1"}, wantID: "pi-session-1"},
		{name: "valid OMP identity", state: map[string]any{"sessionId": "omp-18-session"}, wantID: "omp-18-session"},
		{name: "missing session identity", state: map[string]any{}, wantErr: true},
		{name: "empty session identity", state: map[string]any{"sessionId": ""}, wantErr: true},
		{name: "null session identity", state: map[string]any{"sessionId": nil}, wantErr: true},
		{name: "invalid session type", state: map[string]any{"sessionId": 42}, wantErr: true},
		{name: "unexpected state envelope", state: []any{"session"}, wantErr: true},
		{name: "leading whitespace", state: map[string]any{"sessionId": " session"}, wantErr: true},
		{name: "line break in identity", state: map[string]any{"sessionId": "session\nunsafe"}, wantErr: true},
		{name: "oversized identity", state: map[string]any{"sessionId": strings.Repeat("x", 257)}, wantErr: true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			providerRead, clientWrite := io.Pipe()
			clientRead, providerWrite := io.Pipe()
			t.Cleanup(func() {
				_ = providerRead.Close()
				_ = clientWrite.Close()
				_ = clientRead.Close()
				_ = providerWrite.Close()
			})

			events := make(chan WireSignal, 4)
			bridge, err := NewBridge("pi_rpc", clientRead, clientWrite,
				"/workspace/demo", func(signal WireSignal) { events <- signal }, nil)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(bridge.Stop)
			go func() {
				wire := NewJSONLines(providerRead, providerWrite)
				raw, err := wire.Read()
				if err != nil {
					return
				}
				var request struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				}
				if json.Unmarshal(raw, &request) != nil || request.Type != "get_state" {
					return
				}
				_ = wire.Write(map[string]any{
					"type": "response", "id": request.ID,
					"command": "get_state", "success": true, "data": tt.state,
				})
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			err = bridge.Start(ctx)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "session") {
					t.Fatalf("unsafe handshake accepted or incorrectly described: %v", err)
				}
				if bridge.SessionID() != "" {
					t.Fatalf("invalid provider identity retained: %q", bridge.SessionID())
				}
				select {
				case event := <-events:
					t.Fatalf("invalid handshake emitted READY: %+v", event)
				default:
				}
				return
			}
			if err != nil || bridge.SessionID() != tt.wantID {
				t.Fatalf("valid provider identity rejected: id=%q error=%v", bridge.SessionID(), err)
			}
			select {
			case signal := <-events:
				if signal.State != "idle" {
					t.Fatalf("unexpected state after handshake: %+v", signal)
				}
			case <-time.After(time.Second):
				t.Fatal("valid handshake did not advertise READY")
			}
		})
	}
}
