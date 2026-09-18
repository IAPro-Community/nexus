//go:build !windows

package registry

import "os"

// ProcessExecutablePath returns the executable currently backing pid. Linux
// and procfs-compatible Unix systems expose this without attaching or
// inspecting process memory.
func ProcessExecutablePath(pid int) (string, error) {
	if pid <= 0 {
		return "", os.ErrInvalid
	}
	return os.Readlink("/proc/" + itoa(pid) + "/exe")
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var digits [20]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		i--
		digits[i] = '-'
	}
	return string(digits[i:])
}
