//go:build linux

package capacitymonitor

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type linuxActivityDetector struct{}

// NewActivityDetector returns a Linux-specific ActivityDetector using /proc.
func NewActivityDetector() ActivityDetector {
	return &linuxActivityDetector{}
}

func (d *linuxActivityDetector) Snapshot(ctx context.Context) (map[string]bool, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	result := make(map[string]bool)
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if !entry.IsDir() {
			continue
		}
		// Check if name is numeric PID
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}

		pid := entry.Name()
		// Inspect only executable symlink and comm name
		exePath, _ := os.Readlink(fmt.Sprintf("/proc/%s/exe", pid))
		commBytes, _ := os.ReadFile(fmt.Sprintf("/proc/%s/comm", pid))
		comm := strings.TrimSpace(string(commBytes))

		if provider, active := NormalizeProviderActivity(comm, exePath); active {
			result[provider] = true
		}
	}

	return result, nil
}
