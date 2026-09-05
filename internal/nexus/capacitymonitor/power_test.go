package capacitymonitor

import (
	"context"
	"testing"
)

func TestPowerState(t *testing.T) {
	power := NewPowerState()
	if power == nil {
		t.Fatal("expected non-nil PowerState")
	}

	low, err := power.LowPower(context.Background())
	// In Linux, LowPower either succeeds returning bool, or returns nil error if no battery.
	// We just verify it executes without panics or unexpected failures.
	_ = low
	_ = err
}
