package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// The same process-host IPC connection must receive lease_acquire *before*
// the remaining browser gets the new CONTROL role. A WS-only handoff causes
// input to be dropped by SessionHost even though the UI says CONTROL.
func TestBrokerPromotesHostLeaseBeforeBrowserControl(t *testing.T) {
	for _, operation := range []string{"disconnect", "release"} {
		t.Run(operation, func(t *testing.T) {
			connected := make(chan *websocket.Conn, 2)
			upgrader := websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool { return true },
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := upgrader.Upgrade(w, r, nil)
				if err == nil {
					connected <- conn
				}
			}))
			defer server.Close()
			url := "ws" + strings.TrimPrefix(server.URL, "http")
			connect := func() (*websocket.Conn, *websocket.Conn) {
				t.Helper()
				client, _, err := websocket.DefaultDialer.Dial(url, nil)
				if err != nil {
					t.Fatal(err)
				}
				host := <-connected
				t.Cleanup(func() {
					client.Close()
					host.Close()
				})
				return client, host
			}
			clientA, hostA := connect()
			_, hostB := connect()
			b := &AgentTerminalBroker{agents: make(map[string]*agentTerminalState)}
			if role := b.Attach("agent", hostA, true, &sync.Mutex{}); role != "CONTROL" {
				t.Fatalf("initial role=%s", role)
			}
			if role := b.Attach("agent", hostB, true, &sync.Mutex{}); role != "VIEW_ONLY" {
				t.Fatalf("second role=%s", role)
			}
			var promoted atomic.Int32
			if !b.SetPromotionHook("agent", hostA, func() error {
				promoted.Add(1)
				return nil
			}) {
				t.Fatal("could not register host-lease transfer hook")
			}
			if !b.TakeControl("agent", hostB) {
				t.Fatal("mobile control request not granted")
			}
			_ = clientA.SetReadDeadline(time.Now().Add(3 * time.Second))
			var role map[string]string
			if err := clientA.ReadJSON(&role); err != nil || role["role"] != "VIEW_ONLY" {
				t.Fatalf("expected demotion: %+v, %v", role, err)
			}
			if operation == "disconnect" {
				b.Detach("agent", hostB)
			} else {
				b.ReleaseControl("agent", hostB)
			}
			if err := clientA.ReadJSON(&role); err != nil || role["role"] != "CONTROL" {
				t.Fatalf("expected promotion: %+v, %v", role, err)
			}
			if promoted.Load() != 1 {
				t.Fatalf("browser received CONTROL without exactly one host acquire: %d", promoted.Load())
			}
			if b.Writer("agent") != hostA {
				t.Fatal("broker writer does not match promoted stream")
			}
			b.Detach("agent", hostA)
			b.Detach("agent", hostB)
		})
	}
}

func TestBrokerPromotionHookRejectsUnknownConnection(t *testing.T) {
	b := &AgentTerminalBroker{agents: make(map[string]*agentTerminalState)}
	if b.SetPromotionHook("missing", nil, func() error { return nil }) {
		t.Fatal("unattached connection must not register a host lease callback")
	}
}

func TestBrokerHostLeaseFailureNeverGrantsBrowserControl(t *testing.T) {
	for _, operation := range []string{"disconnect", "release"} {
		t.Run(operation, func(t *testing.T) {
			connected := make(chan *websocket.Conn, 2)
			upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := upgrader.Upgrade(w, r, nil)
				if err == nil {
					connected <- conn
				}
			}))
			defer server.Close()
			url := "ws" + strings.TrimPrefix(server.URL, "http")
			connect := func() (*websocket.Conn, *websocket.Conn) {
				t.Helper()
				client, _, err := websocket.DefaultDialer.Dial(url, nil)
				if err != nil {
					t.Fatal(err)
				}
				host := <-connected
				t.Cleanup(func() {
					_ = client.Close()
					_ = host.Close()
				})
				return client, host
			}
			clientA, hostA := connect()
			_, hostB := connect()
			broker := &AgentTerminalBroker{agents: make(map[string]*agentTerminalState)}
			broker.Attach("agent", hostA, true, &sync.Mutex{})
			broker.Attach("agent", hostB, true, &sync.Mutex{})
			if !broker.SetPromotionHook("agent", hostA, func() error {
				return errors.New("simulated host lease write error")
			}) {
				t.Fatal("missing promotion hook")
			}
			if !broker.TakeControl("agent", hostB) {
				t.Fatal("second client could not take control")
			}
			_ = clientA.SetReadDeadline(time.Now().Add(3 * time.Second))
			var msg map[string]string
			if err := clientA.ReadJSON(&msg); err != nil || msg["role"] != "VIEW_ONLY" {
				t.Fatalf("initial demotion = %+v, %v", msg, err)
			}
			if operation == "disconnect" {
				broker.Detach("agent", hostB)
			} else {
				broker.ReleaseControl("agent", hostB)
			}
			if err := clientA.ReadJSON(&msg); err != nil || msg["role"] != "VIEW_ONLY" {
				t.Fatalf("host acquisition failure must not announce CONTROL: %+v, %v", msg, err)
			}
			if err := clientA.ReadJSON(&msg); err != nil || msg["type"] != "error" {
				t.Fatalf("host acquisition failure must be reported: %+v, %v", msg, err)
			}
			if got := broker.Writer("agent"); got != nil {
				t.Fatalf("broker granted a writer without SessionHost authority: %v", got)
			}
			broker.Detach("agent", hostA)
			broker.Detach("agent", hostB)
		})
	}
}
