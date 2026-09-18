//go:build !windows

package update

import (
	"errors"
	"fmt"
	"os"
)

func replaceExecutable(binaryPath, tempFile, backupPath string) error {
	if _, err := os.Stat(binaryPath); err == nil {
		if err := os.Remove(backupPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to remove stale backup: %w", err)
		}
		if err := os.Rename(binaryPath, backupPath); err != nil {
			return fmt.Errorf("failed to backup current binary: %w", err)
		}
	}
	if err := os.Rename(tempFile, binaryPath); err != nil {
		_ = os.Rename(backupPath, binaryPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}
	return nil
}
