package protocol

import (
	"bufio"
	"context"
	"fmt"
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = WaitForEndpoint(ctx, runtimeID, 150*time.Millisecond)
	if err == nil {
		t.Fatal("WaitForEndpoint succeeded without a ping response")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("WaitForEndpoint exceeded its deadline by too much: %s", elapsed)
	}
	select {
	case <-peerDone:
	case <-time.After(time.Second):
		t.Fatal("unresponsive peer was not disconnected after cancellation")
	}
}
