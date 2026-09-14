//go:build windows

package codex

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	lockfileExclusiveLock   = 0x00000002
	lockfileFailImmediately = 0x00000001
	errorLockViolation      = syscall.Errno(33)
)

func flockExclusive(f *os.File, blocking bool) error {
	var overlapped syscall.Overlapped
	flags := uintptr(lockfileExclusiveLock)
	if !blocking {
		flags |= lockfileFailImmediately
	}
	ret, _, errCode := procLockFileEx.Call(
		f.Fd(),
		flags,
		0,
		1,
		0,
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if ret == 0 {
		if errno, ok := errCode.(syscall.Errno); ok {
			return errno
		}
		return fmt.Errorf("LockFileEx: %v", errCode)
	}
	return nil
}

func flockShared(f *os.File, blocking bool) error {
	var overlapped syscall.Overlapped
	flags := uintptr(0)
	if !blocking {
		flags |= lockfileFailImmediately
	}
	ret, _, errCode := procLockFileEx.Call(
		f.Fd(), flags, 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)),
	)
	if ret == 0 {
		if errno, ok := errCode.(syscall.Errno); ok {
			return errno
		}
		return fmt.Errorf("LockFileEx: %v", errCode)
	}
	return nil
}

func flockUnlock(f *os.File) error {
	var overlapped syscall.Overlapped
	ret, _, errCode := procUnlockFileEx.Call(
		f.Fd(),
		0,
		1,
		0,
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if ret == 0 {
		if errno, ok := errCode.(syscall.Errno); ok {
			return errno
		}
		return fmt.Errorf("UnlockFileEx: %v", errCode)
	}
	return nil
}

func isLockBusy(err error) bool {
	return errors.Is(err, errorLockViolation)
}
