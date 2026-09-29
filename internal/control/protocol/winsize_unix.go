//go:build !windows

package protocol

import (
	"os"
	"os/signal"
	"syscall"
)

// NotifyWinSizeChange registers a channel for window resize signals and
// returns a cleanup function for the registration.
func NotifyWinSizeChange(ch chan os.Signal) func() {
	signal.Notify(ch, syscall.SIGWINCH)
	return func() { signal.Stop(ch) }
}
