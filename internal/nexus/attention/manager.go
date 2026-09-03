package attention

import (
	"sync"
	"time"
)

// Manager is the global attention manager that coordinates attention events
// across all projects and provides aggregated views.
type Manager struct {
	mu      sync.RWMutex
	stream  Stream
 subscribers map[string]map[chan AttentionEvent]struct{} // projectID -> set of channels
}

// NewManager creates a new global attention manager.
func NewManager(stream Stream) *Manager {
	if stream == nil {
		stream = NewMemoryStream(DefaultStreamLimit)
	}
	return &Manager{
		stream:      stream,
		subscribers: make(map[string]map[chan AttentionEvent]struct{}),
	}
}

// Ingest adds an attention event to the global stream and notifies subscribers.
func (m *Manager) Ingest(event AttentionEvent) (AttentionEvent, bool, error) {
	stored, inserted, err := m.stream.Ingest(event)
	if err != nil {
		return AttentionEvent{}, false, err
	}

	if inserted {
		m.notifySubscribers(event.ProjectID, stored)
	}

	return stored, inserted, nil
}

// Acknowledge marks an attention event as acknowledged.
func (m *Manager) Acknowledge(projectID, eventID string, acknowledgedAt time.Time) (AttentionEvent, bool) {
	return m.stream.Acknowledge(projectID, eventID, acknowledgedAt)
}

// Replay returns attention events matching the filter.
func (m *Manager) Replay(filter ReplayFilter) []AttentionEvent {
	return m.stream.Replay(filter)
}

// ProjectSummary returns the aggregated attention summary for a project.
func (m *Manager) ProjectSummary(projectID string) ProjectAttentionSummary {
	return m.stream.ProjectSummary(projectID)
}

// AllProjectSummaries returns summaries for all projects that have active attention events.
func (m *Manager) AllProjectSummaries() []ProjectAttentionSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect all unique project IDs from the stream
	allEvents := m.stream.Replay(ReplayFilter{Limit: DefaultStreamLimit})
	projectIDs := make(map[string]struct{})
	for _, event := range allEvents {
		projectIDs[event.ProjectID] = struct{}{}
	}

	summaries := make([]ProjectAttentionSummary, 0, len(projectIDs))
	for projectID := range projectIDs {
		summaries = append(summaries, m.stream.ProjectSummary(projectID))
	}

	return summaries
}

// Subscribe returns a channel that receives attention events for a specific project.
// The channel is buffered and drops events if full (non-blocking).
func (m *Manager) Subscribe(projectID string) <-chan AttentionEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan AttentionEvent, 64)
	if m.subscribers[projectID] == nil {
		m.subscribers[projectID] = make(map[chan AttentionEvent]struct{})
	}
	m.subscribers[projectID][ch] = struct{}{}

	return ch
}

// Unsubscribe removes a subscription channel.
func (m *Manager) Unsubscribe(projectID string, ch <-chan AttentionEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	subs := m.subscribers[projectID]
	if subs == nil {
		return
	}

	// Find the writable channel behind the read-only channel
	for sub := range subs {
		// Compare by converting writable to read-only
		var readOnlyCh <-chan AttentionEvent = sub
		if readOnlyCh == ch {
			delete(subs, sub)
			close(sub)
			break
		}
	}

	if len(subs) == 0 {
		delete(m.subscribers, projectID)
	}
}

// SubscribeAll returns a channel that receives all attention events across all projects.
func (m *Manager) SubscribeAll() <-chan AttentionEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan AttentionEvent, 128)
	// Use empty string as wildcard project ID
	if m.subscribers[""] == nil {
		m.subscribers[""] = make(map[chan AttentionEvent]struct{})
	}
	m.subscribers[""][ch] = struct{}{}

	return ch
}

// UnsubscribeAll removes a global subscription channel.
func (m *Manager) UnsubscribeAll(ch <-chan AttentionEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	subs := m.subscribers[""]
	if subs == nil {
		return
	}

	for sub := range subs {
		var readOnlyCh <-chan AttentionEvent = sub
		if readOnlyCh == ch {
			delete(subs, sub)
			close(sub)
			break
		}
	}

	if len(subs) == 0 {
		delete(m.subscribers, "")
	}
}

func (m *Manager) notifySubscribers(projectID string, event AttentionEvent) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Notify project-specific subscribers
	if projectSubs, ok := m.subscribers[projectID]; ok {
		for ch := range projectSubs {
			select {
			case ch <- event:
			default:
				// Drop if channel is full (non-blocking)
			}
		}
	}

	// Notify global (wildcard) subscribers
	if globalSubs, ok := m.subscribers[""]; ok {
		for ch := range globalSubs {
			select {
			case ch <- event:
			default:
				// Drop if channel is full (non-blocking)
			}
		}
	}
}

// DefaultManager is the global singleton attention manager.
var (
	defaultManager *Manager
	managerOnce    sync.Once
)

// DefaultManager returns the global attention manager singleton.
func DefaultManager() *Manager {
	managerOnce.Do(func() {
		defaultManager = NewManager(nil)
	})
	return defaultManager
}
