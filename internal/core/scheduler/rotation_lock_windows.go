//go:build windows

package scheduler

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	rotationKernel32     = syscall.NewLazyDLL("kernel32.dll")
	rotationLockFileEx   = rotationKernel32.NewProc("LockFileEx")
	rotationUnlockFileEx = rotationKernel32.NewProc("UnlockFileEx")
)

const rotationLockExclusive = 0x00000002

func acquireRotationLock(path string) (func() error, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open account rotation lock: %w", err)
	}
	var overlapped syscall.Overlapped
	result, _, callErr := rotationLockFileEx.Call(file.Fd(), uintptr(rotationLockExclusive), 0, 0xFFFFFFFF, 0xFFFFFFFF, uintptr(unsafe.Pointer(&overlapped)))
	if result == 0 {
		_ = file.Close()
		return nil, fmt.Errorf("lock account rotation state: %w", callErr)
	}
	return func() error {
		var unlockOverlapped syscall.Overlapped
		result, _, callErr := rotationUnlockFileEx.Call(file.Fd(), 0, 0xFFFFFFFF, 0xFFFFFFFF, uintptr(unsafe.Pointer(&unlockOverlapped)))
		closeErr := file.Close()
		if result == 0 {
			return fmt.Errorf("unlock account rotation state: %w", callErr)
		}
		return closeErr
	}, nil
}
