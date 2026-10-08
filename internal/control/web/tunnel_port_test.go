package web

import (
	"context"
	"net"
	"testing"
)

func TestTunnelTargetsActualServerListenerPort(t *testing.T) {
	srv, err := NewServer(ServerOptions{
		Host: "127.0.0.1", Port: 0, NoOpen: true, DisableHostsFile: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })
	tcp, ok := srv.listener.Addr().(*net.TCPAddr)
	if !ok || tcp.Port == 0 {
		t.Fatalf("expected bound TCP listener, got %v", srv.listener.Addr())
	}
	if got := srv.port(); got != tcp.Port {
		t.Fatalf("tunnel must forward to actual listener %d, got %d", tcp.Port, got)
	}
}
