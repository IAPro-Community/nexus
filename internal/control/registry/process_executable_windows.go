//go:build windows

package registry

import "os"

// Windows process image lookup requires a native handle with query rights.
// Binary identity and HostGeneration remain authoritative on Windows; this
// optional probe is kept as a no-op when the handle cannot be queried.
func ProcessExecutablePath(pid int) (string, error) {
	if pid <= 0 {
		return "", os.ErrInvalid
	}
	return "", nil
}
