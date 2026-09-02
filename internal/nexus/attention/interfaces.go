package attention

import "time"

type Writer interface {
	Ingest(event AttentionEvent) (stored AttentionEvent, inserted bool, err error)
	Acknowledge(eventID string, acknowledgedAt time.Time) (stored AttentionEvent, changed bool)
}

type Reader interface {
	Replay(filter ReplayFilter) []AttentionEvent
	ProjectSummary(projectID string) ProjectAttentionSummary
}

type Stream interface {
	Writer
	Reader
}
