package agentwire

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

// The terminal-shaped RPC bridge can be attached by multiple clients.
// It must never issue /answer instructions for fields the provider marks secret.
func TestSecretRPCDialogFailsClosedWithoutPromptDisclosure(t *testing.T) {
	for _, method := range []string{"input", "editor"} {
		t.Run(method, func(t *testing.T) {
			providerRead, clientWrite := io.Pipe()
			clientRead, providerWrite := io.Pipe()
			defer providerRead.Close()
			defer clientWrite.Close()
			defer clientRead.Close()
			defer providerWrite.Close()

			var printed strings.Builder
			var signals []WireSignal
			bridge, err := NewBridge("pi_rpc", clientRead, clientWrite,
				"/workspace/demo", func(s WireSignal) { signals = append(signals, s) },
				func(message string) { printed.WriteString(message) })
			if err != nil {
				t.Fatal(err)
			}
			reply := make(chan json.RawMessage, 1)
			go func() {
				frame, _ := NewJSONLines(providerRead, io.Discard).Read()
				reply <- frame
			}()

			const sensitive = "api-key-sensitive-title-should-not-render"
			payload, err := json.Marshal(map[string]any{
				"type":    "extension_ui_request",
				"id":      "private-dialog",
				"method":  method,
				"secret":  true,
				"title":   sensitive,
				"message": sensitive,
			})
			if err != nil {
				t.Fatal(err)
			}
			bridge.handlePiUI(payload)

			select {
			case frame := <-reply:
				var response struct {
					ID        string `json:"id"`
					Type      string `json:"type"`
					Cancelled bool   `json:"cancelled"`
					Value     string `json:"value"`
				}
				if err := json.Unmarshal(frame, &response); err != nil {
					t.Fatal(err)
				}
				if response.ID != "private-dialog" || response.Type != "extension_ui_response" ||
					!response.Cancelled || response.Value != "" {
					t.Fatalf("secret input was not cancelled safely: %s", frame)
				}
			case <-time.After(time.Second):
				t.Fatal("secret dialog cancellation was not sent to provider")
			}

			bridge.mu.Lock()
			pending := len(bridge.pending)
			bridge.mu.Unlock()
			if pending != 0 || len(signals) != 0 {
				t.Fatalf("secret dialog became remotely actionable: pending=%d signals=%v", pending, signals)
			}
			if strings.Contains(printed.String(), sensitive) ||
				strings.Contains(printed.String(), "/answer") {
				t.Fatal("secret prompt or remote answer instructions leaked to terminal")
			}
			if !strings.Contains(printed.String(), "cannot mask secrets") {
				t.Fatal("operator did not receive a safe explanation")
			}
		})
	}
}
