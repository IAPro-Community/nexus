package host

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/agentwire"
)

// Test helper is launched as an actual child with stdin/stdout JSONL.
// It never uses network credentials or a real provider account.
func TestMockWireAgentProcess(t *testing.T) {
	if os.Getenv("NEXUS_WIRE_TEST_CHILD") != "1" {
		return
	}
	input := bufio.NewScanner(os.Stdin)
	w := bufio.NewWriter(os.Stdout)
	for input.Scan() {
		var req struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if json.Unmarshal(input.Bytes(), &req) != nil {
			os.Exit(3)
		}
		switch req.Method {
		case "initialize":
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{"protocolVersion": 1}})
		case "session/new":
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{"sessionId": "owned-sess-1"}})
		case "session/prompt":
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{"stopReason": "end_turn"}})
		}
		if err := w.Flush(); err != nil {
			os.Exit(4)
		}
	}
	os.Exit(0)
}
func TestStructuredBackendSupervisesRealPipeProcess(t *testing.T) {
	status := make(chan agentwire.WireSignal, 10)
	sessions := make(chan string, 1)
	backend := newStructuredBackend("acp", t.TempDir(), func(e agentwire.WireSignal) { status <- e }, func(id string) { sessions <- id })
	cmd := exec.Command(os.Args[0], "-test.run=^TestMockWireAgentProcess$")
	cmd.Env = append(os.Environ(), "NEXUS_WIRE_TEST_CHILD=1")
	cmd.Dir = t.TempDir()
	if err := backend.Start(cmd, 24, 80); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = backend.Kill(); _ = backend.Close(); _ = backend.Wait() }()
	select {
	case id := <-sessions:
		if id != "owned-sess-1" {
			t.Fatalf("session id=%s", id)
		}
	case <-time.After(time.Second * 2):
		t.Fatal("provider session not bound")
	}
	if _, err := backend.Write([]byte("review the code\r")); err != nil {
		t.Fatal(err)
	}
	// An ACP prompt must complete without echoing raw JSON-RPC to UI.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() {
		for {
			select {
			case e := <-status:
				if e.State == "done" {
					close(done)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("provider turn never completed")
	}
	if !strings.Contains(backend.Mechanism(), "acp") {
		t.Fatal("transport incorrectly identified")
	}
}
