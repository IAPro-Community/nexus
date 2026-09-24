//go:build !linux && !windows && !darwin

package desktop

import "fmt"

func autostartIsEnabled(m *AutoStartManager) (bool, error) {
	return false, nil
}

func autostartSetEnabled(m *AutoStartManager, enabled bool) error {
	if !enabled {
		return nil
	}
	return fmt.Errorf("autostart is unsupported on this platform")
}
