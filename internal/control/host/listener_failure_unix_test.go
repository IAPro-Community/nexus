//go:build !windows

package host

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
)

func prepareListenerFailureFixture(t *testing.T) (string, func()) {
	t.Helper()
	runtimeID := "rt-listen-fail-test"
	sockPath := protocol.EndpointPath(runtimeID)
	// A non-empty directory at the Unix socket path makes protocol.Listen fail
	// when it attempts to remove/replace the occupied endpoint.
	if err := os.MkdirAll(filepath.Join(sockPath, "blocking-child"), 0700); err != nil {
		t.Fatalf("create listener failure fixture: %v", err)
	}
	return runtimeID, func() { _ = os.RemoveAll(sockPath) }
}
