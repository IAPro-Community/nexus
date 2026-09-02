package attention

import (
	"strings"
	"sync"
	"time"
)

type MemoryStream struct {
	mu       sync.RWMutex
	capacity int
	events   []AttentionEvent
	index    map[string]int
}

func NewMemoryStream(capacity int) *MemoryStream {
	if capacity <= 0 {
		capacity = DefaultStreamLimit
	}
	return &MemoryStream{
		capacity: capacity,
		events:   make([]AttentionEvent, 0, capacity),
		index:    make(map[string]int, capacity),
	}
}

func (s *MemoryStream) Ingest(event AttentionEvent) (AttentionEvent, bool, error) {
	normalized, err := NormalizeEvent(event)
	if err != nil {
		return AttentionEvent{}, false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if idx, ok := s.index[normalized.ID]; ok {
		return s.events[idx], false, nil
	}

	s.events = append(s.events, normalized)
	s.index[normalized.ID] = len(s.events) - 1
	s.trimLocked()

	return normalized, true, nil
}

func (s *MemoryStream) Acknowledge(eventID string, acknowledgedAt time.Time) (AttentionEvent, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx, ok := s.index[eventID]
	if !ok {
		return AttentionEvent{}, false
	}

	event := s.events[idx]
	if event.AcknowledgedAt != nil {
		return event, false
	}

	if acknowledgedAt.IsZero() {
		acknowledgedAt = time.Now().UTC()
	} else {
		acknowledgedAt = acknowledgedAt.UTC()
	}
	if acknowledgedAt.Before(event.OccurredAt) {
		acknowledgedAt = event.OccurredAt
	}

	event.AcknowledgedAt = &acknowledgedAt
	s.events[idx] = event

	return event, true
}

func (s *MemoryStream) Replay(filter ReplayFilter) []AttentionEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 {
		limit = DefaultReplayLimit
	}

	matches := make([]AttentionEvent, 0, limit)
	for i := len(s.events) - 1; i >= 0; i-- {
		event := s.events[i]
		if !matchesFilter(event, filter) {
			continue
		}
		matches = append(matches, event)
		if len(matches) == limit {
			break
		}
	}

	for left, right := 0, len(matches)-1; left < right; left, right = left+1, right-1 {
		matches[left], matches[right] = matches[right], matches[left]
	}

	return matches
}

func (s *MemoryStream) ProjectSummary(projectID string) ProjectAttentionSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := ProjectAttentionSummary{ProjectID: projectID}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return summary
	}
	summary.ProjectID = projectID

	latestBySubject := make(map[string]AttentionEvent)
	for _, event := range s.events {
		if event.ProjectID != projectID {
			continue
		}

		subjectKey := logicalSubjectKey(event)
		current, exists := latestBySubject[subjectKey]
		if !exists || current.OccurredAt.Before(event.OccurredAt) || (current.OccurredAt.Equal(event.OccurredAt) && current.ID < event.ID) {
			latestBySubject[subjectKey] = event
		}
	}

	priorityScore := 0
	var priorityOccurredAt time.Time
	for _, event := range latestBySubject {
		summary.Total++
		if summary.LatestOccurredAt.Before(event.OccurredAt) {
			summary.LatestOccurredAt = event.OccurredAt
		}

		switch event.State {
		case StateWorking:
			summary.Working++
		case StateWaitingInput:
			summary.WaitingInput++
		case StateWaitingApproval:
			summary.WaitingApproval++
		case StateCompleted:
			summary.Completed++
			if event.AcknowledgedAt == nil {
				summary.CompletedUnread++
			}
		case StateFailed:
			summary.Failed++
		case StateRateLimited:
			summary.RateLimited++
		case StateRecoverable:
			summary.Recoverable++
		}

		score := priorityForEvent(event)
		if score > priorityScore || (score == priorityScore && shouldReplacePriority(priorityOccurredAt, summary.PriorityEventID, event)) {
			priorityScore = score
			priorityOccurredAt = event.OccurredAt
			summary.PriorityState = event.State
			summary.PriorityReason = event.Reason
			summary.PriorityEventID = event.ID
		}
	}

	return summary
}

func (s *MemoryStream) trimLocked() {
	if len(s.events) <= s.capacity {
		return
	}
	excess := len(s.events) - s.capacity
	for i := 0; i < excess; i++ {
		delete(s.index, s.events[i].ID)
	}
	s.events = append([]AttentionEvent(nil), s.events[excess:]...)
	s.reindexLocked()
}

func (s *MemoryStream) reindexLocked() {
	s.index = make(map[string]int, len(s.events))
	for idx, event := range s.events {
		s.index[event.ID] = idx
	}
}

func matchesFilter(event AttentionEvent, filter ReplayFilter) bool {
	if filter.ProjectID != "" && event.ProjectID != filter.ProjectID {
		return false
	}
	if filter.AgentID != "" && event.AgentID != filter.AgentID {
		return false
	}
	if filter.RuntimeID != "" && event.RuntimeID != filter.RuntimeID {
		return false
	}
	if filter.FlowRunID != "" && event.FlowRunID != filter.FlowRunID {
		return false
	}
	if filter.ViewLogicalKey != "" && event.ViewLogicalKey != filter.ViewLogicalKey {
		return false
	}
	return true
}

func logicalSubjectKey(event AttentionEvent) string {
	switch {
	case event.ViewLogicalKey != "":
		return "view:" + event.ViewLogicalKey
	case event.FlowRunID != "":
		return "flow:" + event.FlowRunID
	case event.AgentID != "":
		return "agent:" + event.AgentID
	case event.RuntimeID != "":
		return "runtime:" + event.RuntimeID
	default:
		return "event:" + event.ID
	}
}

func priorityForEvent(event AttentionEvent) int {
	switch event.State {
	case StateFailed:
		return 700
	case StateWaitingApproval:
		return 600
	case StateWaitingInput:
		return 590
	case StateCompleted:
		if event.AcknowledgedAt == nil {
			return 500
		}
		return 150
	case StateRecoverable:
		return 400
	case StateRateLimited:
		return 350
	case StateWorking:
		return 300
	default:
		return 0
	}
}

func shouldReplacePriority(currentOccurredAt time.Time, currentEventID string, event AttentionEvent) bool {
	if currentEventID == "" {
		return true
	}
	if event.OccurredAt.After(currentOccurredAt) {
		return true
	}
	if event.OccurredAt.Equal(currentOccurredAt) && event.ID > currentEventID {
		return true
	}
	return false
}
