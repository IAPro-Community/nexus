//go:build linux

package capacitymonitor

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type linuxPowerState struct {
	powerSupplyDir string
}

// NewPowerState returns a Linux PowerState reader inspecting /sys/class/power_supply.
func NewPowerState() PowerState {
	return &linuxPowerState{powerSupplyDir: "/sys/class/power_supply"}
}

func (p *linuxPowerState) LowPower(ctx context.Context) (bool, error) {
	entries, err := os.ReadDir(p.powerSupplyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		typePath := filepath.Join(p.powerSupplyDir, entry.Name(), "type")
		typeBytes, err := os.ReadFile(typePath)
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(typeBytes)) != "Battery" {
			continue
		}

		statusBytes, _ := os.ReadFile(filepath.Join(p.powerSupplyDir, entry.Name(), "status"))
		status := strings.TrimSpace(string(statusBytes))

		capacityBytes, _ := os.ReadFile(filepath.Join(p.powerSupplyDir, entry.Name(), "capacity"))
		capacity, _ := strconv.Atoi(strings.TrimSpace(string(capacityBytes)))

		// If battery is discharging and capacity is under 20%, consider it low power
		if status == "Discharging" && capacity <= 20 {
			return true, nil
		}
	}

	return false, nil
}
