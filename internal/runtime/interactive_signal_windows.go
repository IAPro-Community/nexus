//go:build windows

package runtime

import "os"

func interactiveResizeSignals() []os.Signal { return nil }

func isInteractiveResizeSignal(os.Signal) bool { return false }
