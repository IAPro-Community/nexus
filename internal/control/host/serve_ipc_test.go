package host

import (
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

// fakePipeAddr is a minimal net.Addr for test connections.
type fakePipeAddr struct{}

func (fakePipeAddr) Network() string { return "pipe" }
func (fakePipeAddr) String() string  { return "fake" }

// eofPipeConn closes immediately from the peer side so handleClient returns
// after the accept, mirroring a client that connects and disconnects.
type eofPipeConn struct {
	closeOnce sync.Once
	closed    chan struct{}
}

func newEOFPipeConn() *eofPipeConn {
	return &eofPipeConn{closed: make(chan struct{})}
}

func (c *eofPipeConn) Read([]byte) (int, error)  { return 0, io.EOF }
func (c *eofPipeConn) Write(b []byte) (int, error) { return len(b), nil }
func (c *eofPipeConn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return nil
}
func (c *eofPipeConn) LocalAddr() net.Addr                { return fakePipeAddr{} }
func (c *eofPipeConn) RemoteAddr() net.Addr               { return fakePipeAddr{} }
func (c *eofPipeConn) SetDeadline(time.Time) error        { return nil }
func (c *eofPipeConn) SetReadDeadline(time.Time) error    { return nil }
func (c *eofPipeConn) SetWriteDeadline(time.Time) error   { return nil }

// fakeAcceptListener fails a configurable number of Accept calls before
// serving queued connections, reproducing a transient accept error while the
// pipe object still exists in the Windows namespace.
type fakeAcceptListener struct {
	mu        sync.Mutex
	failures  int
	conns     chan net.Conn
	closeOnce sync.Once
	closed    chan struct{}
}

func newFakeAcceptListener() *fakeAcceptListener {
	return &fakeAcceptListener{conns: make(chan net.Conn, 4), closed: make(chan struct{})}
}

func (l *fakeAcceptListener) failOnce() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failures = 1
}

func (l *fakeAcceptListener) Accept() (net.Conn, error) {
	l.mu.Lock()
	fail := l.failures > 0
	if fail {
		l.failures--
	}
	l.mu.Unlock()
	if fail {
		return nil, errors.New("transient accept failure")
	}
	select {
	case conn := <-l.conns:
		return conn, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *fakeAcceptListener) Close() error {
	l.closeOnce.Do(func() { close(l.closed) })
	return nil
}

func (l *fakeAcceptListener) Addr() net.Addr { return fakePipeAddr{} }

// TestServeIPCSurvivesTransientAcceptError pins the regression where a single
// transient Accept error retired the accept loop: the pipe kept existing in
// the object namespace while no instance ever accepted a connection, so every
// client dial timed out even though the host process was alive. The loop must
// keep serving after the error and exit only when the listener closes.
func TestServeIPCSurvivesTransientAcceptError(t *testing.T) {
	sess := registry.RuntimeSession{
		RuntimeID:    "rt-serve-ipc-transient",
		ProviderID:   "test",
		ProfileID:    "default",
		Workspace:    os.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}
	sh, err := NewSessionHost(Config{Session: sess})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}

	listener := newFakeAcceptListener()
	done := make(chan struct{})
	go func() {
		defer close(done)
		sh.serveIPC(listener)
	}()

	listener.failOnce()
	conn := newEOFPipeConn()
	listener.conns <- conn

	select {
	case <-conn.closed:
		// handleClient ran after the transient failure: the loop survived.
	case <-time.After(3 * time.Second):
		t.Fatal("serveIPC stopped accepting after a transient accept error")
	}

	sh.stopOnce.Do(func() { close(sh.stopChan) })
	_ = listener.Close()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("serveIPC did not exit after the listener closed")
	}
}
