package protocol

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestWaitForEndpointCancelsUnresponsivePeer(t *testing.T) {
	runtimeID := fmt.Sprintf("wait-cancel-%d", time.Now().UnixNano())
	listener, err := Listen(runtimeID)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer listener.Close()

	peerDone := make(chan struct{})
	go func() {
		defer close(peerDone)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		if _, err := reader.ReadString('\n'); err != nil {
			return
		}
		// Keep the connected peer silent. Ping must be canceled by the wait
		// deadline, including on Windows named pipes where deadlines alone
		// do not reliably interrupt a blocked read.
		_, _ = conn.Read(make([]byte, 1))
	}()

	started := time.Now()
	// The outer context only bounds the test and must never be the reason
	// WaitForEndpoint returns; keep it far above the 150ms wait deadline so
	// CPU contention on loaded runners cannot blur which deadline fired.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = WaitForEndpoint(ctx, runtimeID, 150*time.Millisecond)
	if err == nil {
		t.Fatal("WaitForEndpoint succeeded without a ping response")
	}
	// Contract: the wait deadline itself must end the wait. Wall-clock bounds
	// flake under CPU contention, so assert the deterministic error identity:
	// an outer-context failure surfaces context.DeadlineExceeded instead.
	if !strings.Contains(err.Error(), "waiting for endpoint") {
		t.Fatalf("WaitForEndpoint must fail on its own wait deadline after %s, got: %v", time.Since(started), err)
	}
	select {
	case <-peerDone:
	case <-time.After(10 * time.Second):
		t.Fatal("unresponsive peer was not disconnected after cancellation")
	}
}
