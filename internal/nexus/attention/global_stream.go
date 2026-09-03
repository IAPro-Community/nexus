package attention

import (
	"encoding/json"
	"sync"
	"time"
)

// StreamEvent is the wire format for attention events sent over the global stream.
type StreamEvent struct {
	Type      string           `json:"type"` // "attention", "summary", "replay"
	Event     *AttentionEvent  `json:"event,omitempty"`
	Summary   *ProjectAttentionSummary `json:"summary,omitempty"`
	Events    []AttentionEvent `json:"events,omitempty"`
	Timestamp time.Time        `json:"timestamp"`
}

// GlobalStream manages a single WebSocket connection that delivers attention
// events from all projects to the browser.
type GlobalStream struct {
	mu          sync.RWMutex
	manager     *Manager
	connections map[*StreamConnection]struct{}
}

// StreamConnection represents a single browser WebSocket connection.
type StreamConnection struct {
	ID       string
	send     chan []byte
	done     chan struct{}
	lastSeen map[string]time.Time // projectID -> last event time for dedup
}

// NewGlobalStream creates a new global attention stream.
func NewGlobalStream(manager *Manager) *GlobalStream {
	return &GlobalStream{
		manager:     manager,
		connections: make(map[*StreamConnection]struct{}),
	}
}

// Connect registers a new WebSocket connection and starts streaming events.
func (gs *GlobalStream) Connect(connID string) *StreamConnection {
	conn := &StreamConnection{
		ID:       connID,
		send:     make(chan []byte, 64),
		done:     make(chan struct{}),
		lastSeen: make(map[string]time.Time),
	}

	gs.mu.Lock()
	gs.connections[conn] = struct{}{}
	gs.mu.Unlock()

	// Subscribe to all events
	eventCh := gs.manager.SubscribeAll()

	go func() {
		defer func() {
			gs.manager.UnsubscribeAll(eventCh)
			gs.mu.Lock()
			delete(gs.connections, conn)
			gs.mu.Unlock()
		}()

		for {
			select {
			case event, ok := <-eventCh:
				if !ok {
					return
				}
				gs.sendEvent(conn, StreamEvent{
					Type:      "attention",
					Event:     &event,
					Timestamp: time.Now(),
				})
			case <-conn.done:
				return
			}
		}
	}()

	return conn
}

// Disconnect removes a WebSocket connection.
func (gs *GlobalStream) Disconnect(conn *StreamConnection) {
	select {
	case <-conn.done:
		// Already closed
	default:
		close(conn.done)
	}

	gs.mu.Lock()
	delete(gs.connections, conn)
	gs.mu.Unlock()
}

// Send sends a raw message to a specific connection.
func (gs *GlobalStream) Send(conn *StreamConnection, data []byte) {
	select {
	case conn.send <- data:
	default:
		// Drop if full
	}
}

// Broadcast sends a message to all connected clients.
func (gs *GlobalStream) Broadcast(data []byte) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	for conn := range gs.connections {
		select {
		case conn.send <- data:
		default:
			// Drop if full
		}
	}
}

// ReplayAndSubscribe sends recent events to a new connection and then subscribes
// to live events. This implements the bounded replay + live stream pattern.
func (gs *GlobalStream) ReplayAndSubscribe(connID string, lastEventID string) *StreamConnection {
	conn := gs.Connect(connID)

	// Determine which events to replay
	var events []AttentionEvent
	if lastEventID != "" {
		// Replay all events since the last seen
		events = gs.manager.Replay(ReplayFilter{Limit: DefaultReplayLimit})
		// Filter to events after lastEventID
		filtered := make([]AttentionEvent, 0)
		found := false
		for _, e := range events {
			if found {
				filtered = append(filtered, e)
			}
			if e.ID == lastEventID {
				found = true
			}
		}
		events = filtered
	} else {
		// Replay recent events
		events = gs.manager.Replay(ReplayFilter{Limit: DefaultReplayLimit})
	}

	// Send replay
	if len(events) > 0 {
		replayData, err := json.Marshal(StreamEvent{
			Type:      "replay",
			Events:    events,
			Timestamp: time.Now(),
		})
		if err == nil {
			gs.Send(conn, replayData)
		}
	}

	return conn
}

func (gs *GlobalStream) sendEvent(conn *StreamConnection, event StreamEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	// Dedup: skip if we've sent the same event ID recently
	gs.mu.Lock()
	if event.Event != nil {
		if lastTime, ok := conn.lastSeen[event.Event.ProjectID]; ok {
			if time.Since(lastTime) < 100*time.Millisecond {
				gs.mu.Unlock()
				return
			}
		}
		conn.lastSeen[event.Event.ProjectID] = time.Now()
	}
	gs.mu.Unlock()

	select {
	case conn.send <- data:
	default:
		// Drop if full
	}
}

// ConnectionCount returns the number of active connections.
func (gs *GlobalStream) ConnectionCount() int {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return len(gs.connections)
}
