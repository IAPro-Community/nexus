package runtime

import (
	"io"
	"os"
	"sync"
)

// stdinRelay owns at most one goroutine reading from src (typically os.Stdin)
// and forwards bytes to whichever terminal backend is currently attached.
// Keeping a single reader per source prevents leaked goroutines from racing on
// os.Stdin across sequential failovers or resumed interactive sessions.
type stdinRelay struct {
	src    io.Reader
	onExit func()

	mu           sync.Mutex
	cond         *sync.Cond
	started      bool
	closed       bool
	activeWriter io.Writer
	activeToken  uint64
}

var (
	stdinRelaysMu sync.Mutex
	stdinRelays   = map[io.Reader]*stdinRelay{}
)

func newStdinRelay(src io.Reader) *stdinRelay {
	r := &stdinRelay{src: src}
	r.cond = sync.NewCond(&r.mu)
	return r
}

func attachDefaultStdinRelay(w io.Writer) func() {
	return attachStdinRelay(os.Stdin, w)
}

func (r *stdinRelay) isClosed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closed
}

func attachStdinRelay(src io.Reader, w io.Writer) func() {
	if src == nil || w == nil {
		return func() {}
	}
	stdinRelaysMu.Lock()
	r := stdinRelays[src]
	if r == nil || r.isClosed() {
		var created *stdinRelay
		created = &stdinRelay{
			src: src,
			onExit: func() {
				stdinRelaysMu.Lock()
				if stdinRelays[src] == created {
					delete(stdinRelays, src)
				}
				stdinRelaysMu.Unlock()
			},
		}
		created.cond = sync.NewCond(&created.mu)
		stdinRelays[src] = created
		r = created
	}
	stdinRelaysMu.Unlock()
	return r.Attach(w)
}

// Attach routes stdin bytes to w until the returned detach function is called
// or a newer writer replaces w.
func (r *stdinRelay) Attach(w io.Writer) func() {
	if r == nil || w == nil || r.src == nil {
		return func() {}
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return func() {}
	}
	if r.cond == nil {
		r.cond = sync.NewCond(&r.mu)
	}
	r.activeToken++
	token := r.activeToken
	r.activeWriter = w
	if !r.started {
		r.started = true
		go r.pump()
	}
	r.cond.Broadcast()
	r.mu.Unlock()

	return func() {
		r.mu.Lock()
		if r.activeToken == token {
			r.activeWriter = nil
		}
		r.mu.Unlock()
	}
}

func (r *stdinRelay) Close() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.closed = true
	r.activeWriter = nil
	if r.cond != nil {
		r.cond.Broadcast()
	}
	r.mu.Unlock()
}

func (r *stdinRelay) pump() {
	defer func() {
		r.mu.Lock()
		r.closed = true
		r.activeWriter = nil
		r.mu.Unlock()
		if r.onExit != nil {
			r.onExit()
		}
	}()

	buf := make([]byte, 4096)
	for {
		r.mu.Lock()
		for !r.closed && r.activeWriter == nil {
			r.cond.Wait()
		}
		if r.closed {
			r.mu.Unlock()
			return
		}
		r.mu.Unlock()

		n, err := r.src.Read(buf)
		if n > 0 {
			r.mu.Lock()
			dst := r.activeWriter
			token := r.activeToken
			r.mu.Unlock()
			if dst != nil {
				if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
					r.mu.Lock()
					if r.activeToken == token {
						r.activeWriter = nil
					}
					r.mu.Unlock()
				}
			}
		}
		if err != nil {
			return
		}
	}
}
