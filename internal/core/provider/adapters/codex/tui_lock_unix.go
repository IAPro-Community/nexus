//go:build !windows

package codex

import (
	"errors"
	"os"
	"syscall"
)

func flockExclusive(f *os.File, blocking bool) error {
	how := syscall.LOCK_EX
	if !blocking {
		how |= syscall.LOCK_NB
	}
	return syscall.Flock(int(f.Fd()), how)
}

func flockShared(f *os.File, blocking bool) error {
	how := syscall.LOCK_SH
	if !blocking {
		how |= syscall.LOCK_NB
	}
	return syscall.Flock(int(f.Fd()), how)
}

func flockUnlock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}

func isLockBusy(err error) bool {
	return errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN)
}
