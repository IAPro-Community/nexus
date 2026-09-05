//go:build windows

package capacitymonitor

import (
	"context"
)

type windowsActivityDetector struct{}

// NewActivityDetector returns an unsupported/degraded ActivityDetector for Windows until native tests exist.
func NewActivityDetector() ActivityDetector {
	return &windowsActivityDetector{}
}

func (d *windowsActivityDetector) Snapshot(ctx context.Context) (map[string]bool, error) {
	return make(map[string]bool), nil
}
