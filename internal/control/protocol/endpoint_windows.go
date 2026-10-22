//go:build windows

package protocol

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

// Listen creates the local Windows Named Pipe listener for a runtime.
// Uses owner-restricted security descriptor to prevent cross-user access on local machine.
func Listen(runtimeID string) (net.Listener, error) {
	path := EndpointPath(runtimeID)

	cfg := &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;OW)",
		MessageMode:        false,
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	}

	l, err := winio.ListenPipe(path, cfg)
	if err != nil {
		return nil, fmt.Errorf("secure named pipe bind failed: %w", err)
	}
	return l, nil
}

func dialEndpoint(ctx context.Context, runtimeID string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, EndpointPath(runtimeID))
}

func isRetryableDialError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, os.ErrNotExist) ||
		errors.Is(err, windows.ERROR_FILE_NOT_FOUND) ||
		errors.Is(err, windows.ERROR_PATH_NOT_FOUND) ||
		errors.Is(err, winio.ErrTimeout)
}

func isOwnedBy(st os.FileInfo, uid int) bool {
	// Not applicable on Windows.
	return true
}
