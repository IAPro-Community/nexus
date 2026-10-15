package protocol

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestWaitForEndpointCancelsUnresponsivePeer verifies two Windows-critical
// contracts of the ping-based readiness probe:
//
//  1. When the peer accepts the connection but never answers, WaitForEndpoint
//     must fail on its own wait deadline (an outer-context failure surfaces
//     context.DeadlineExceeded instead and would mask the real cause).
//  2. The canceled ping must close the pipe so the unresponsive peer is
//     disconnected (go-winio deadlines alone do not interrupt a blocked read,
//     so a canceled RPC would otherwise leak its connection).
//
// On Windows named pipes the client can only connect while the listener has an
// Accept pending: if the peer goroutine has not reached Accept yet, dialing
// fails retryably. The test therefore completes one probe connection before
// starting the deadline-bearing wait, which proves the accept loop is live and
// keeps both assertions independent of scheduler latency on loaded runners.
func TestWaitForEndpointCancelsUnresponsivePeer(t *testing.T) {
	runtimeID := fmt.Sprintf("wait-cancel-%d", time.Now().UnixNano())
	listener, err := Listen(runtimeID)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer listener.Close()

	peerDisconnected := make(chan struct{}, 8)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			reader := bufio.NewReader(conn)
			if _, err := reader.ReadString('\n'); err != nil {
				// Client vanished before sending a request (probe connection).
				_ = conn.Close()
				continue
			}
			// Keep the connected peer silent. Ping cancellation must close the
			// pipe and disconnect this blocked read.
			_, _ = conn.Read(make([]byte, 1))
			_ = conn.Close()
			select {
			case peerDisconnected <- struct{}{}:
			default:
			}
		}
	}()

	// Finalize the first accept before the wait starts; see the doc comment.
	probeCtx, cancelProbe := context.WithTimeout(context.Background(), 15*time.Second)
	probe, err := DialContext(probeCtx, runtimeID)
	cancelProbe()
	if err != nil {
		t.Fatalf("probe dial: %v", err)
	}
	_ = probe.Close()

	// The outer context only bounds the test and must never be the reason
	// WaitForEndpoint returns; keep it far above the wait deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	started := time.Now()
	err = WaitForEndpoint(ctx, runtimeID, 1500*time.Millisecond)
	if err == nil {
		t.Fatal("WaitForEndpoint succeeded without a ping response")
	}
	// Contract: the wait deadline itself must end the wait. Wall-clock bounds
	// flake under CPU contention, so assert the deterministic error identity:
	// an outer-context failure surfaces context.DeadlineExceeded instead.
	if !strings.Contains(err.Error(), "timed out waiting for endpoint") {
		t.Fatalf("WaitForEndpoint must fail on its own wait deadline after %s, got: %v", time.Since(started), err)
	}
	select {
	case <-peerDisconnected:
	case <-time.After(10 * time.Second):
		t.Fatal("unresponsive peer was not disconnected after cancellation")
	}
}
