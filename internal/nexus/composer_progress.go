package nexus

// ComposerProgressSink reports bootstrap steps during CreateComposerSession.
// Handlers may stream these to the client; nil sinks are no-ops.
type ComposerProgressSink interface {
	Step(id, status string)
}

type noopComposerProgress struct{}

func (noopComposerProgress) Step(string, string) {}

func progressOrNoop(sink ComposerProgressSink) ComposerProgressSink {
	if sink == nil {
		return noopComposerProgress{}
	}
	return sink
}
