package capacitymonitor

import (
	"time"
)

// DetermineCollectionInterval calculates the appropriate collection interval based on
// current capacity state, external CLI activity, power state, and backoff history.
func DetermineCollectionInterval(
	state State,
	activeActivity bool,
	lowPower bool,
	consecutiveFailures int,
	probeMinInterval time.Duration,
) time.Duration {
	var interval time.Duration

	if lowPower {
		interval = 5 * time.Minute
	} else if consecutiveFailures > 0 {
		switch consecutiveFailures {
		case 1:
			interval = 1 * time.Minute
		case 2:
			interval = 2 * time.Minute
		case 3:
			interval = 5 * time.Minute
		default:
			interval = 15 * time.Minute
		}
	} else if state == StateWarning || state == StateCritical {
		interval = 45 * time.Second
	} else if activeActivity {
		interval = 60 * time.Second
	} else {
		// Healthy or idle
		interval = 10 * time.Minute
	}

	if probeMinInterval > 0 && interval < probeMinInterval {
		interval = probeMinInterval
	}

	return interval
}
