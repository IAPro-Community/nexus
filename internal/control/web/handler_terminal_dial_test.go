package web

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

// A host that is not live must fail within a single dial attempt instead of
// burning the full attach budget: this is the fast-fail path behind the HTTP
// 404 that keeps a dead runtime cheap for the client's lease window.
func TestDialAttachControlFailsFastWhenHostNotLive(t *testing.T) {
	runtimeID := fmt.Sprintf("attach-dead-%d", time.Now().UnixNano())
	dead := registry.RuntimeSession{RuntimeID: runtimeID, State: registry.StateRunning}

	started := time.Now()
	client, err := dialAttachControlBudget(dead, runtimeID, attachDialBudget)
	if err == nil {
		_ = client.Close()
		t.Fatal("expected the dial to fail for a host that is not live")
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("dead host must fail within a single attempt, took %s", elapsed)
	}
}

// The attach dial must survive the same slow pipe acceptance the launch
// handshake tolerates: the endpoint appears after the first attempts fail and
// the retry loop still connects it. This is the regression test for the
// spurious 404 that broke the terminal reconnect when the pipe needed more
// than one short dial budget to start accepting.
func TestDialAttachControlRetriesUntilPipeAccepts(t *testing.T) {
	runtimeID := fmt.Sprintf("attach-slow-%d", time.Now().UnixNano())
	accepted := make(chan error, 1)

	go func() {
		time.Sleep(time.Second)
		listener, err := protocol.Listen(runtimeID)
		if err != nil {
			accepted <- err
			return
		}
		defer listener.Close()
		conn, err := listener.Accept()
		if err != nil {
			accepted <- err
			return
		}
		_ = conn.Close()
		accepted <- nil
	}()

	live := registry.RuntimeSession{
		RuntimeID: runtimeID,
		State:     registry.StateRunning,
		HostPID:   os.Getpid(),
	}
	client, err := dialAttachControlBudget(live, runtimeID, attachDialBudget)
	if err != nil {
		t.Fatalf("attach dial did not retry while the pipe was starting: %v", err)
	}
	_ = client.Close()

	select {
	case err := <-accepted:
		if err != nil {
			t.Fatalf("listener did not accept the retried connection: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("listener did not accept the retried connection")
	}
}

// A live host whose pipe never appears must stop once the budget elapses, with
// an error that records the attempt count — never an unbounded retry and never
// a return before the budget is spent.
func TestDialAttachControlStopsAtBudget(t *testing.T) {
	runtimeID := fmt.Sprintf("attach-missing-%d", time.Now().UnixNano())
	live := registry.RuntimeSession{
		RuntimeID: runtimeID,
		State:     registry.StateRunning,
		HostPID:   os.Getpid(),
	}

	started := time.Now()
	client, err := dialAttachControlBudget(live, runtimeID, 700*time.Millisecond)
	if err == nil {
		_ = client.Close()
		t.Fatal("expected the dial to exhaust its budget")
	}
	elapsed := time.Since(started)
	if elapsed < 700*time.Millisecond {
		t.Fatalf("dial returned before the budget elapsed: %s", elapsed)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("budget must bound the dial, took %s", elapsed)
	}
	if !strings.Contains(err.Error(), "attempts") {
		t.Fatalf("expected budget error to record attempts, got: %v", err)
	}
}
