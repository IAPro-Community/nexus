package events

import (
	"sort"
	"sync"
)

// Bus provides thread-safe in-memory pub/sub routing and historical buffer for events.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan Event
	history     map[string][]Event
	maxHistory  int
	recorder    func(Event)
}

var (
	defaultBus *Bus
	busOnce    sync.Once
)

// DefaultBus returns the singleton event bus instance.
func DefaultBus() *Bus {
	busOnce.Do(func() {
		defaultBus = NewBus(200)
	})
	return defaultBus
}

// NewBus creates a new Event Bus with a max history size per runtime.
func NewBus(maxHistory int) *Bus {
	if maxHistory <= 0 {
		maxHistory = 200
	}
	return &Bus{
		subscribers: make(map[string][]chan Event),
		history:     make(map[string][]Event),
		maxHistory:  maxHistory,
	}
}

// SetRecorder configures a durable recorder hook called whenever an event is published.
func (b *Bus) SetRecorder(recorder func(Event)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.recorder = recorder
}

// Publish distributes an event to all interested subscribers and appends to history.
func (b *Bus) Publish(e Event) {
	b.mu.Lock()
	recorder := b.recorder

	// Append to history
	hist := b.history[e.RuntimeID]
	hist = append(hist, e)
	if len(hist) > b.maxHistory {
		hist = hist[len(hist)-b.maxHistory:]
	}
	b.history[e.RuntimeID] = hist

	// Broadcast to runtime subscribers and wildcard subscribers ("*")
	targets := append([]chan Event{}, b.subscribers[e.RuntimeID]...)
	targets = append(targets, b.subscribers["*"]...)
	b.mu.Unlock()

	if recorder != nil {
		recorder(e)
	}

	for _, ch := range targets {
		// Protect against send-on-closed-channel when Unsubscribe races with Publish.
		func(c chan Event) {
			defer func() { recover() }() //nolint:errcheck // closed channel is expected race
			select {
			case c <- e:
			default:
				// Non-blocking drop if channel buffer is full
			}
		}(ch)
	}
}

// Subscribe opens a stream of events for a runtime ID (or "*" for all runtimes).
// Returns the event channel and an unsubscribe function.
func (b *Bus) Subscribe(runtimeID string) (<-chan Event, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, 64)
	b.subscribers[runtimeID] = append(b.subscribers[runtimeID], ch)

	unsub := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		subs := b.subscribers[runtimeID]
		for i, sub := range subs {
			if sub == ch {
				b.subscribers[runtimeID] = append(subs[:i], subs[i+1:]...)
				close(ch)
				break
			}
		}
	}

	return ch, unsub
}

// GetHistory returns recent events for a given runtime ID.
func (b *Bus) GetHistory(runtimeID string, limit int) []Event {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if runtimeID == "" {
		var all []Event
		for _, hist := range b.history {
			all = append(all, hist...)
		}
		sort.SliceStable(all, func(i, j int) bool { return all[i].Timestamp.Before(all[j].Timestamp) })
		if limit <= 0 || limit > len(all) {
			limit = len(all)
		}
		if limit == 0 {
			return nil
		}
		out := make([]Event, limit)
		copy(out, all[len(all)-limit:])
		return out
	}

	hist, ok := b.history[runtimeID]
	if !ok {
		return nil
	}

	if limit <= 0 || limit > len(hist) {
		limit = len(hist)
	}

	out := make([]Event, limit)
	copy(out, hist[len(hist)-limit:])
	return out
}
