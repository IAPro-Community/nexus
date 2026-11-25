//go:build !windows

package launcher

import (
	"os"
	"strconv"
	"syscall"
)

func processOwnedByCurrentUID(pid int) bool {
	info, err := os.Stat("/proc/" + strconv.Itoa(pid))
	if err != nil {
		return false
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	return int(st.Uid) == os.Getuid()
}

func killProcessGroup(pid int) {
	if pid <= 0 {
		return
	}
	if pgid, err := syscall.Getpgid(pid); err == nil && pgid == pid {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}
}
