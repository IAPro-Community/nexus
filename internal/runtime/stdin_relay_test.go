package runtime

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func waitForBuffer(t *testing.T, buf *syncBuffer, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if buf.String() == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("buffer timed out: got %q, want %q", buf.String(), want)
}

func TestStdinRelaySequentialHandoffWithoutLeak(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() {
		_ = pw.Close()
		_ = pr.Close()
	})

	relay := newStdinRelay(pr)
	t.Cleanup(relay.Close)

	first := &syncBuffer{}
	detachFirst := relay.Attach(first)
	if _, err := io.WriteString(pw, "alpha"); err != nil {
		t.Fatal(err)
	}
	waitForBuffer(t, first, "alpha")

	// Simulate failover: second session attaches before or after first detaches.
	second := &syncBuffer{}
	detachSecond := relay.Attach(second)
	detachFirst() // stale detach must not clear second writer
	if _, err := io.WriteString(pw, "beta"); err != nil {
		t.Fatal(err)
	}
	waitForBuffer(t, second, "beta")
	if got := first.String(); got != "alpha" {
		t.Fatalf("stale writer received failover bytes: %q", got)
	}
	detachSecond()
}
