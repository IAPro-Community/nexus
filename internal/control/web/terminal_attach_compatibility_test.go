package web

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/gorilla/websocket"
)

func TestTerminalAttachBinaryMismatchReturnsActionableWebSocketError(t *testing.T) {
	runtimeID := "binary-mismatch-" + strings.TrimSpace(time.Now().Format("150405.000000000"))
	listener, err := protocol.Listen(runtimeID)
	if err != nil {
		t.Fatalf("listen for runtime control endpoint: %v", err)
	}
	defer listener.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			accepted <- conn
		}
	}()
	defer func() {
		select {
		case conn := <-accepted:
			_ = conn.Close()
		default:
		}
	}()

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve test executable: %v", err)
	}
	currentFingerprint, err := registry.BinaryFingerprint(executable)
	if err != nil {
		t.Fatalf("fingerprint test executable: %v", err)
	}
	reg := registry.NewRegistry("")
	if err := reg.Register(registry.RuntimeSession{
		RuntimeID:                 runtimeID,
		AgentID:                   "agent-binary-mismatch",
		ProviderID:                "test",
		ProfileID:                 "default",
		Workspace:                 t.TempDir(),
		PID:                       os.Getpid(),
		HostPID:                   os.Getpid(),
		HostBinaryPath:            executable,
		HostBinaryFingerprint:     strings.Repeat("0", 64),
		ProviderBinaryPath:        executable,
		ProviderBinaryFingerprint: currentFingerprint,
		State:                     registry.StateRunning,
		ControlLevel:              registry.ControlLevelTerminal,
		ControlEndpoint:           protocol.EndpointPath(runtimeID),
	}); err != nil {
		t.Fatalf("register runtime: %v", err)
	}

	hub := NewTerminalHub(nil, reg)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r, "agent-binary-mismatch", runtimeID)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	invalidOrigin := http.Header{"Origin": []string{"https://untrusted.example"}}
	_, response, err := websocket.DefaultDialer.Dial(wsURL, invalidOrigin)
	if err == nil {
		t.Fatal("expected untrusted Origin to be rejected")
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("expected 403 for untrusted Origin, got %d", status)
	}
	stored, ok := reg.Get(runtimeID)
	if !ok || stored.State != registry.StateRunning {
		t.Fatalf("untrusted Origin must not mutate runtime state, got %+v (found=%v)", stored, ok)
	}

	header := http.Header{"Origin": []string{server.URL}}
	conn, response, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("expected structured WebSocket error, got HTTP %d: %v", status, err)
	}
	defer conn.Close()

	var message struct {
		Type string `json:"type"`
		Code string `json:"code"`
	}
	if err := conn.ReadJSON(&message); err != nil {
		t.Fatalf("read structured attach error: %v", err)
	}
	if message.Type != "error" || message.Code != "RUNTIME_ATTACH_INCOMPATIBLE" {
		t.Fatalf("expected runtime incompatibility code, got %+v", message)
	}

	stored, ok = reg.Get(runtimeID)
	if !ok || stored.State != registry.StateStale {
		t.Fatalf("expected incompatible runtime to remain registered as STALE, got %+v (found=%v)", stored, ok)
	}
	if !registry.IsProcessAlive(os.Getpid()) {
		t.Fatal("attach rejection must not terminate the existing runtime host process")
	}
}
