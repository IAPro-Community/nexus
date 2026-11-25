//go:build !windows

package runtime

import (
	"os"
	"syscall"
)

func interactiveResizeSignals() []os.Signal {
	return []os.Signal{syscall.SIGWINCH}
}

func isInteractiveResizeSignal(sig os.Signal) bool {
	return sig == syscall.SIGWINCH
}
