//go:build !windows

package protocol

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
)

// Listen creates the local server listener for a runtime.
func Listen(runtimeID string) (net.Listener, error) {
	path := EndpointPath(runtimeID)
	dir := filepath.Dir(path)
	if len(path) > maxUnixSocketPathLen {
		return nil, fmt.Errorf("unix socket path exceeds %d bytes", maxUnixSocketPathLen)
	}
	if !ensurePrivateSocketDir(dir, os.Getuid()) {
		return nil, fmt.Errorf("unix socket directory is not private or user-owned: %s", dir)
	}

	// Remove dead/stale socket file if present
	_ = os.Remove(path)

	oldMask := syscall.Umask(0177)
	l, err := net.Listen("unix", path)
	syscall.Umask(oldMask)
	if err != nil {
		return nil, err
	}

	// Enforce user-only permissions (0600) on the unix socket file
	if err := os.Chmod(path, 0600); err != nil {
		l.Close()
		return nil, fmt.Errorf("failed to chmod socket: %w", err)
	}

	return l, nil
}

func dialEndpoint(ctx context.Context, runtimeID string) (net.Conn, error) {
	path := EndpointPath(runtimeID)
	if len(path) > maxUnixSocketPathLen {
		return nil, fmt.Errorf("unix socket path exceeds %d bytes", maxUnixSocketPathLen)
	}
	if !ensurePrivateSocketDir(filepath.Dir(path), os.Getuid()) {
		return nil, fmt.Errorf("unix socket directory is not private or user-owned: %s", filepath.Dir(path))
	}
	var dialer net.Dialer
	return dialer.DialContext(ctx, "unix", path)
}

func isRetryableDialError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, os.ErrNotExist) ||
		errors.Is(err, syscall.ECONNREFUSED)
}

func isOwnedBy(st os.FileInfo, uid int) bool {
	if sys, ok := st.Sys().(*syscall.Stat_t); ok {
		return sys.Uid == uint32(uid)
	}
	return false
}
