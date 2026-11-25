//go:build windows

package launcher

func processOwnedByCurrentUID(int) bool { return true }

func killProcessGroup(int) {}
