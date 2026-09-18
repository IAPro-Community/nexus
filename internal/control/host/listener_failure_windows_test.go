//go:build windows

package host

import (
	"testing"
)

func prepareListenerFailureFixture(t *testing.T) (string, func()) {
	t.Helper()
	// Named Pipes do not have a filesystem path that can be occupied by a
	// directory. A NUL in the endpoint name is rejected by go-winio while
	// converting the path to UTF-16, exercising the real listener-creation
	// failure path before the provider process can start.
	return "rt-listener-failure\x00", func() {}
}
