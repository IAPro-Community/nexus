//go:build !windows

package computer

import (
	"fmt"
	"io"
)

func nativeHelperSupported() bool { return false }

func runNativeHelper(io.Reader, io.Writer) error {
	return fmt.Errorf("%w: native computer-use helper is not implemented for this platform", ErrUnavailable)
}
