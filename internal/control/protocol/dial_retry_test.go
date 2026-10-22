package protocol

import (
	"fmt"
	"testing"
	"time"
)

func TestDialRetriesWhenEndpointAppears(t *testing.T) {
	runtimeID := fmt.Sprintf("dial-retry-%d", time.Now().UnixNano())
	accepted := make(chan error, 1)

	go func() {
		time.Sleep(time.Second)
		listener, err := Listen(runtimeID)
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

	conn, err := Dial(runtimeID)
	if err != nil {
		t.Fatalf("Dial did not retry while the endpoint was starting: %v", err)
	}
	_ = conn.Close()

	select {
	case err := <-accepted:
		if err != nil {
			t.Fatalf("listener did not accept the retried connection: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("listener did not accept the retried connection")
	}
}
