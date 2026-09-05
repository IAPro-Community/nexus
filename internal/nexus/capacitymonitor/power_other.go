//go:build !linux

package capacitymonitor

import (
	"context"
)

type unsupportedPowerState struct{}

// NewPowerState returns a truthful unsupported PowerState for non-Linux platforms.
func NewPowerState() PowerState {
	return &unsupportedPowerState{}
}

func (p *unsupportedPowerState) LowPower(ctx context.Context) (bool, error) {
	return false, ErrPowerStateUnsupported
}
