//go:build !linux

package capacitymonitor

import (
	"context"
)

// OtherAutostartManager provides a stub for non-Linux platforms.
type OtherAutostartManager struct{}

// NewOtherAutostartManager creates a fallback manager for non-Linux OSes.
func NewOtherAutostartManager() *OtherAutostartManager {
	return &OtherAutostartManager{}
}

// NewAutostartManager returns the platform-specific autostart manager.
func NewAutostartManager() AutostartManager {
	return NewOtherAutostartManager()
}

func (m *OtherAutostartManager) Status(ctx context.Context) (AutostartStatus, error) {
	return AutostartStatus{
		Supported: false,
		Installed: false,
		Enabled:   false,
		Active:    false,
	}, nil
}

func (m *OtherAutostartManager) InstallAndEnable(ctx context.Context, execPath string) error {
	return ErrAutostartUnsupported
}

func (m *OtherAutostartManager) DisableAndRemove(ctx context.Context) error {
	return nil
}
