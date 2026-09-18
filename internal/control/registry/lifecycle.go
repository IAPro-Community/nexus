package registry

// CanTransition reports whether a runtime may move between operational
// states. Equal states are idempotent so repeated observations are safe.
// Administrative recovery still uses the explicit STOPPED/FAILED -> STARTING
// paths rather than jumping directly into a live state.
func CanTransition(from, to RuntimeState) bool {
	if from == to {
		return true
	}
	switch from {
	case StateStarting:
		return to == StateRunning || to == StateFailed
	case StateRunning:
		return to == StateWaiting || to == StateApproval || to == StateDetached ||
			to == StateHandoff || to == StateStopping || to == StateStopped || to == StateFailed
	case StateWaiting:
		return to == StateRunning || to == StateApproval || to == StateHandoff ||
			to == StateStopping || to == StateStopped || to == StateFailed
	case StateApproval:
		return to == StateRunning || to == StateWaiting || to == StateHandoff ||
			to == StateStopping || to == StateStopped || to == StateFailed
	case StateDetached:
		return to == StateRunning || to == StateHandoff || to == StateStopping ||
			to == StateStopped || to == StateFailed
	case StateHandoff:
		return to == StateRunning || to == StateStarting || to == StateStopping ||
			to == StateStopped || to == StateFailed
	case StateStopping:
		return to == StateStopped || to == StateFailed
	case StateStopped, StateFailed, StateStale:
		return to == StateStarting
	default:
		return false
	}
}
