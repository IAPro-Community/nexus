package capacitymonitor

import (
	"context"
	"errors"
)

// ErrPowerStateUnsupported is returned when the platform does not expose power state telemetry.
var ErrPowerStateUnsupported = errors.New("power state is unsupported on this platform")

// PowerState detects whether the host machine is in low-power / battery-saver mode.
type PowerState interface {
	LowPower(ctx context.Context) (bool, error)
}
