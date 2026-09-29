//go:build windows

package protocol

import (
	"os"
	"testing"
	"time"
)

func TestNotifyWinSizeChangeCanBeStopped(t *testing.T) {
	ch := make(chan os.Signal, 1)
	stop := NotifyWinSizeChange(ch)
	stop()
	stop()

	// The cleanup must remain idempotent even when CI has no attached console.
	time.Sleep(10 * time.Millisecond)
}
