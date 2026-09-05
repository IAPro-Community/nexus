//go:build darwin

package capacitymonitor

import (
	"context"
)

type darwinActivityDetector struct{}

// NewActivityDetector returns an unsupported/degraded ActivityDetector for Darwin until native tests exist.
func NewActivityDetector() ActivityDetector {
	return &darwinActivityDetector{}
}

func (d *darwinActivityDetector) Snapshot(ctx context.Context) (map[string]bool, error) {
	return make(map[string]bool), nil
}
