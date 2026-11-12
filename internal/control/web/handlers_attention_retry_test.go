package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus"
)

func TestAttentionHTTPRetryReplaysPersistedOutcome(t *testing.T) {
	for _, acknowledge := range []bool{true, false} {
		t.Run(fmt.Sprintf("acknowledge=%t", acknowledge), func(t *testing.T) {
			runtimeID := fmt.Sprintf("http-attention-retry-%d", time.Now().UnixNano())
			listener, err := protocol.Listen(runtimeID)
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			var dispatches atomic.Int32
			go func() {
				for {
					conn, err := listener.Accept()
					if err != nil {
						return
					}
					var request protocol.Request
					if err := json.NewDecoder(conn).Decode(&request); err == nil {
						dispatches.Add(1)
						if acknowledge {
							_ = json.NewEncoder(conn).Encode(protocol.Response{Version: protocol.ProtocolVersion, ID: request.ID, OK: true})
						}
					}
					_ = conn.Close()
				}
			}()
			path := filepath.Join(t.TempDir(), "runtimes.json")
			reg := registry.NewRegistry(path)
			if err := reg.Register(registry.RuntimeSession{
				RuntimeID: runtimeID, State: registry.StateWaiting, HostGeneration: 11, ProviderGeneration: 12,
				AttentionReason: "QUESTION", AttentionKind: "needs_user", AttentionContext: "Continue?",
				PromptKind: "yn", AttentionSource: "INFERRED", AttentionFingerprint: "retry-fp", AttentionRevision: 1,
			}); err != nil {
				t.Fatal(err)
			}
			handler := &APIHandler{reg: reg, runtimes: nexus.NewRuntimeApplicationService(reg, nil, nil)}
			request := func(input string) *httptest.ResponseRecorder {
				payload, err := json.Marshal(map[string]any{
					"input": input, "attention_fingerprint": "retry-fp", "attention_revision": 1, "attention_generation": "11:12",
				})
				if err != nil {
					t.Fatal(err)
				}
				recorder := httptest.NewRecorder()
				handler.handleRuntimeDetail(recorder, httptest.NewRequest(http.MethodPost,
					"/api/v1/runtimes/"+runtimeID+"/respond-attention", bytes.NewReader(payload)))
				return recorder
			}
			initial := request("y")
			wantStatus := http.StatusAccepted
			if !acknowledge {
				wantStatus = http.StatusBadGateway
			}
			if initial.Code != wantStatus {
				t.Fatalf("initial status=%d body=%s", initial.Code, initial.Body.String())
			}
			reopened := registry.NewRegistry(path)
			handler = &APIHandler{reg: reopened, runtimes: nexus.NewRuntimeApplicationService(reopened, nil, nil)}
			retry := request("y")
			if retry.Code != initial.Code || retry.Body.String() != initial.Body.String() {
				t.Fatalf("retry must replay outcome: initial=%d %s retry=%d %s", initial.Code, initial.Body.String(), retry.Code, retry.Body.String())
			}
			if changed := request("n"); changed.Code != http.StatusConflict {
				t.Fatalf("different payload status=%d body=%s", changed.Code, changed.Body.String())
			}
			if got := dispatches.Load(); got != 1 {
				t.Fatalf("HTTP retry dispatched %d operations, want 1", got)
			}
		})
	}
}
