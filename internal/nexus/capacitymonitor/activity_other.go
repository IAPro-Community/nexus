//go:build !linux && !windows && !darwin

package capacitymonitor

import (
	"context"
)

type otherActivityDetector struct{}

// NewActivityDetector returns an unsupported/degraded ActivityDetector for other platforms.
func NewActivityDetector() ActivityDetector {
	return &otherActivityDetector{}
}

func (d *otherActivityDetector) Snapshot(ctx context.Context) (map[string]bool, error) {
	return make(map[string]bool), nil
}
