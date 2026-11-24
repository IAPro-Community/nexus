package host

import (
	"net"
	"sync"
	"sync/atomic"
)

// FanoutStats describes output that could not be delivered to a slow client.
// Dropped chunks are intentional backpressure protection; callers can use the
// counters to detect when a reconnect/history recovery may be necessary.
type FanoutStats struct {
	DroppedChunks uint64
	DroppedBytes  uint64
}

type clientWorker struct {
	conn     net.Conn
	queue    chan []byte
	doneChan chan struct{}
}

// BoundedFanout manages isolated per-client delivery queues so slow observers
// never block the supervised child process terminal stream.
type BoundedFanout struct {
	mu            sync.RWMutex
	clients       map[net.Conn]*clientWorker
	bufCap        int
	droppedChunks atomic.Uint64
	droppedBytes  atomic.Uint64
}

func NewBoundedFanout(bufCap int) *BoundedFanout {
	if bufCap <= 0 {
		bufCap = 256
	}
	return &BoundedFanout{
		clients: make(map[net.Conn]*clientWorker),
		bufCap:  bufCap,
	}
}

func (f *BoundedFanout) AddClient(conn net.Conn, initialFrame ...[]byte) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.clients[conn]; exists {
		return
	}

	worker := &clientWorker{
		conn:     conn,
		queue:    make(chan []byte, f.bufCap),
		doneChan: make(chan struct{}),
	}
	if len(initialFrame) > 0 && len(initialFrame[0]) > 0 {
		worker.queue <- initialFrame[0]
	}
	f.clients[conn] = worker

	go f.pumpClient(worker)
}

func (f *BoundedFanout) RemoveClient(conn net.Conn) {
	f.mu.Lock()
	worker, exists := f.clients[conn]
	if exists {
		delete(f.clients, conn)
	}
	f.mu.Unlock()

	if exists && worker != nil {
		select {
		case <-worker.doneChan:
		default:
			close(worker.doneChan)
		}
	}
}

func (f *BoundedFanout) Broadcast(data []byte) {
	if len(data) == 0 {
		return
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	// Clone chunk so goroutines don't race on underlying buffer
	chunk := make([]byte, len(data))
	copy(chunk, data)

	for _, w := range f.clients {
		select {
		case w.queue <- chunk:
		default:
			// Slow consumer queue is full: drop chunk rather than blocking child process.
			f.droppedChunks.Add(1)
			f.droppedBytes.Add(uint64(len(chunk)))
		}
	}
}

// Stats returns cumulative delivery loss for all clients since creation.
func (f *BoundedFanout) Stats() FanoutStats {
	return FanoutStats{
		DroppedChunks: f.droppedChunks.Load(),
		DroppedBytes:  f.droppedBytes.Load(),
	}
}

func (f *BoundedFanout) pumpClient(w *clientWorker) {
	for {
		select {
		case <-w.doneChan:
			return
		case data, ok := <-w.queue:
			if !ok {
				return
			}
			if _, err := w.conn.Write(data); err != nil {
				return
			}
		}
	}
}

func (f *BoundedFanout) Close() {
	f.mu.Lock()
	workers := make([]*clientWorker, 0, len(f.clients))
	for _, w := range f.clients {
		workers = append(workers, w)
	}
	f.clients = make(map[net.Conn]*clientWorker)
	f.mu.Unlock()

	for _, w := range workers {
		select {
		case <-w.doneChan:
		default:
			close(w.doneChan)
		}
	}
}
