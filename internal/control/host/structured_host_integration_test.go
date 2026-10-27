package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func TestSessionHostStructuredACPLifecycle(t *testing.T) {
	runtimeID := "structured-acp-host-test"
	reg := registry.NewRegistry(filepath.Join(t.TempDir(), "test-registry.db"))

	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "opencode",
		ProfileID:    "default",
		Workspace:    t.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelEvents,
		Labels: map[string]string{
			"nexus.agent_protocol": "acp",
		},
	}

	sh, err := NewSessionHost(Config{
		Session:  sess,
		Registry: reg,
		Binary:   os.Args[0],
		Args:     []string{"-test.run=^TestMockWireAgentProcess$"},
		Env:      append(os.Environ(), "NEXUS_WIRE_TEST_CHILD=1"),
		Cwd:      t.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}

	if err := sh.Start(); err != nil {
		t.Fatalf("failed to start SessionHost: %v", err)
	}
	defer func() { _ = sh.Stop() }()

	// Assert PID is tracked
	sh.mu.Lock()
	pid := sh.session.PID
	state := sh.session.State
	sh.mu.Unlock()

	if pid <= 0 {
		t.Fatalf("expected positive PID, got %d", pid)
	}
	if state != registry.StateRunning {
		t.Fatalf("expected state running, got %s", state)
	}

	// Wait for handshake to bind provider session ID
	deadline := time.Now().Add(3 * time.Second)
	var providerSessID string
	for {
		sh.mu.Lock()
		providerSessID = sh.session.ProviderSessionID
		sh.mu.Unlock()
		if providerSessID != "" || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if providerSessID != "owned-sess-1" {
		t.Fatalf("expected provider session owned-sess-1, got %s", providerSessID)
	}

	// Verify lifecycle events published
	busEvents := events.DefaultBus().GetHistory(runtimeID, 10)
	if !containsEventType(busEvents, events.EventRuntimeStarted) {
		t.Fatalf("runtime_started event not published: %+v", busEvents)
	}

	// Connect client and send prompt
	waitForHostEndpoint(t, runtimeID)
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("failed to connect IPC client: %v", err)
	}
	defer client.Close()

	if err := client.Ping(); err != nil {
		t.Fatalf("ping failed: %v", err)
	}

	// Send prompt via client raw connection
	if _, err := client.RawConn().Write([]byte("run verification\r\n")); err != nil {
		t.Fatalf("failed writing prompt to client: %v", err)
	}

	// Wait for prompt to settle
	time.Sleep(150 * time.Millisecond)

	// Send /cancel
	if _, err := client.RawConn().Write([]byte("/cancel\r\n")); err != nil {
		t.Fatalf("failed writing /cancel: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	// Verify terminal output buffer never contains raw JSON-RPC protocol frames
	ringData := string(sh.ringBuffer.Bytes())
	if strings.Contains(ringData, "\"jsonrpc\":\"2.0\"") {
		t.Fatalf("raw JSON-RPC protocol leaked to user terminal: %s", ringData)
	}

	// Terminate SessionHost
	if err := sh.Terminate(); err != nil {
		t.Fatalf("failed to terminate SessionHost: %v", err)
	}
	sh.Wait()

	// Verify cleanup
	sh.mu.Lock()
	finalState := sh.session.State
	reason := sh.session.AttentionReason
	kind := sh.session.AttentionKind
	sh.mu.Unlock()

	if finalState != registry.StateStopped && finalState != registry.StateFailed {
		t.Fatalf("expected stopped or failed state, got %s", finalState)
	}
	if reason != "" || kind != AttentionIdle {
		t.Fatalf("stale attention remained after process exit: reason=%s kind=%s", reason, kind)
	}
}

func TestSessionHostStructuredPiRPCLifecycle(t *testing.T) {
	runtimeID := "structured-pi-host-test"
	reg := registry.NewRegistry(filepath.Join(t.TempDir(), "test-registry-pi.db"))

	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "omp",
		ProfileID:    "default",
		Workspace:    t.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelEvents,
		Labels: map[string]string{
			"nexus.agent_protocol": "pi_rpc",
		},
	}

	sh, err := NewSessionHost(Config{
		Session:  sess,
		Registry: reg,
		Binary:   os.Args[0],
		Args:     []string{"-test.run=^TestMockPiRPCAgentProcess$"},
		Env:      append(os.Environ(), "NEXUS_PI_TEST_CHILD=1"),
		Cwd:      t.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}

	if err := sh.Start(); err != nil {
		t.Fatalf("failed to start SessionHost: %v", err)
	}
	defer func() { _ = sh.Stop() }()

	sh.mu.Lock()
	pid := sh.session.PID
	state := sh.session.State
	sh.mu.Unlock()

	if pid <= 0 {
		t.Fatalf("expected positive PID, got %d", pid)
	}
	if state != registry.StateRunning {
		t.Fatalf("expected state running, got %s", state)
	}

	// Wait for get_state handshake to bind provider session ID
	deadline := time.Now().Add(3 * time.Second)
	var providerSessID string
	for {
		sh.mu.Lock()
		providerSessID = sh.session.ProviderSessionID
		sh.mu.Unlock()
		if providerSessID != "" || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if providerSessID != "pi-sess-100" {
		t.Fatalf("expected provider session pi-sess-100, got %s", providerSessID)
	}

	// Connect client and send prompt
	waitForHostEndpoint(t, runtimeID)
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("failed to connect IPC client: %v", err)
	}
	defer client.Close()

	if _, err := client.RawConn().Write([]byte("run verification\r\n")); err != nil {
		t.Fatalf("failed writing prompt to client: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	// Send /cancel
	if _, err := client.RawConn().Write([]byte("/cancel\r\n")); err != nil {
		t.Fatalf("failed writing /cancel: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	// Terminate SessionHost
	if err := sh.Terminate(); err != nil {
		t.Fatalf("failed to terminate SessionHost: %v", err)
	}
	sh.Wait()

	sh.mu.Lock()
	finalState := sh.session.State
	sh.mu.Unlock()

	if finalState != registry.StateStopped && finalState != registry.StateFailed {
		t.Fatalf("expected stopped or failed state, got %s", finalState)
	}
}
