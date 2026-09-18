//go:build windows

package update

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	modKernel32              = syscall.NewLazyDLL("kernel32.dll")
	procMoveFileExW          = modKernel32.NewProc("MoveFileExW")
	moveFileReplaceExisting  = uintptr(0x1)
	moveFileDelayUntilReboot = uintptr(0x4)
)

func replaceExecutable(binaryPath, tempFile, backupPath string) error {
	if _, err := os.Stat(binaryPath); err == nil {
		if err := os.Remove(backupPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to remove stale backup: %w", err)
		}
		if err := os.Rename(binaryPath, backupPath); err != nil {
			// Binary is likely locked (in use). Stage beside and schedule reboot replace.
			pending := binaryPath + ".new"
			_ = os.Remove(pending)
			if errCopy := os.Rename(tempFile, pending); errCopy != nil {
				return fmt.Errorf("failed to stage Windows in-use update: %w", errCopy)
			}
			if err := moveFileEx(pending, binaryPath, moveFileReplaceExisting|moveFileDelayUntilReboot); err != nil {
				return fmt.Errorf("failed to schedule Windows in-use replace (reboot required): %w", err)
			}
			return fmt.Errorf("update staged for reboot: binary in use (%w)", err)
		}
	}
	if err := os.Rename(tempFile, binaryPath); err != nil {
		_ = os.Rename(backupPath, binaryPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}
	return nil
}

func moveFileEx(from, to string, flags uintptr) error {
	fromPtr, err := syscall.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	toPtr, err := syscall.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	r1, _, e1 := procMoveFileExW.Call(uintptr(unsafe.Pointer(fromPtr)), uintptr(unsafe.Pointer(toPtr)), flags)
	if r1 == 0 {
		if e1 != nil {
			return e1
		}
		return syscall.EINVAL
	}
	return nil
}
