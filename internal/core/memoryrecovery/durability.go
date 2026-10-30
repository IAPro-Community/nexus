package memoryrecovery

import (
	"fmt"
	"os"
	"runtime"
)

// syncRecoveryDirectory makes newly created locks and operation directories
// durable on local filesystems where directory fsync is supported.
// Windows cannot safely use the same directory fsync contract here: journal
// recovery must be separately verified during platform homologation.
func syncRecoveryDirectory(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	f, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open recovery directory: %w", err)
	}
	syncErr := f.Sync()
	closeErr := f.Close()
	if syncErr != nil {
		return fmt.Errorf("sync recovery directory: %w", syncErr)
	}
	return closeErr
}
