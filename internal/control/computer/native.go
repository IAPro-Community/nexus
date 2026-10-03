package computer

import (
	"io"
)

// NativeHelperSupported reports whether this binary contains a native
// computer-use implementation for the current OS.
func NativeHelperSupported() bool {
	return nativeHelperSupported()
}

// RunNativeHelper serves exactly one helper request from stdin and writes one
// protocol envelope to stdout. The helper is intentionally process-isolated
// from the main Nexus Core.
func RunNativeHelper(in io.Reader, out io.Writer) error {
	return runNativeHelper(in, out)
}
