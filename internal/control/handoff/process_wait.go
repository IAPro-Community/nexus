package handoff

import (
	"context"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

const processStopPollInterval = 50 * time.Millisecond

// waitForProcessStop waits for the source process to stop without hiding
// cancellation from the caller. A false result means the deadline or context
// ended while the process was still alive.
func waitForProcessStop(ctx context.Context, pid int, timeout time.Duration) bool {
	if !registry.IsProcessAlive(pid) {
		return true
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(processStopPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-timer.C:
			return !registry.IsProcessAlive(pid)
		case <-ticker.C:
			if !registry.IsProcessAlive(pid) {
				return true
			}
		}
	}
}
