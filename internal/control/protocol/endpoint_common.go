package protocol

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	defaultDialTimeout = 5 * time.Second
	dialAttemptTimeout = 500 * time.Millisecond
	dialRetryInterval  = 50 * time.Millisecond
)

// Dial connects to a runtime endpoint, retrying while the local endpoint is starting.
func Dial(runtimeID string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultDialTimeout)
	defer cancel()
	return DialContext(ctx, runtimeID)
}

// DialContext connects to a runtime endpoint until it succeeds or ctx expires.
func DialContext(ctx context.Context, runtimeID string) (net.Conn, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		attemptCtx, cancel := context.WithTimeout(ctx, dialAttemptTimeout)
		conn, err := dialEndpoint(attemptCtx, runtimeID)
		cancel()
		if err == nil {
			return conn, nil
		}
		if !isRetryableDialError(err) {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		timer := time.NewTimer(dialRetryInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// maxUnixSocketPathLen is the longest usable unix socket path. macOS
// sockaddr_un.sun_path holds 104 bytes including the trailing NUL, and bind
// fails with EINVAL beyond that. Linux allows 107, so the stricter value is
// used everywhere.
const maxUnixSocketPathLen = 103

// EndpointPath returns the canonical local socket / pipe path for a runtime.
func EndpointPath(runtimeID string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf(`\\.\pipe\nexus-control-%s`, runtimeID)
	}

	uid := os.Getuid()

	// Prefer XDG_RUNTIME_DIR only when its Nexus child is an owned, private
	// directory. An attacker-controlled directory must not host our socket.
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		dir := filepath.Join(xdg, "nexus")
		if path, ok := socketPathIn(dir, runtimeID); ok && ensurePrivateSocketDir(dir, uid) {
			return path
		}
	}

	// The ordinary per-user temporary directory can be deeply nested on
	// macOS. Use it only if the path fits and it is private.
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("nexus-control-%d", uid))
	if path, ok := socketPathIn(dir, runtimeID); ok && ensurePrivateSocketDir(dir, uid) {
		return path
	}

	// The fallback is deliberately deterministic across the control host and
	// its clients. Listen and dialEndpoint reject this path if its directory
	// cannot be established with safe ownership and permissions.
	return filepath.Join(shortSocketDir(uid), fmt.Sprintf("%s.sock", runtimeID))
}

// socketPathIn joins dir and runtimeID when the result fits the kernel limit.
func socketPathIn(dir, runtimeID string) (string, bool) {
	path := filepath.Join(dir, fmt.Sprintf("%s.sock", runtimeID))
	if len(path) > maxUnixSocketPathLen {
		return "", false
	}
	return path, true
}

// shortSocketDir is stable across processes. A hostile pre-existing directory
// cannot make it fall back to a process-specific name and break host/client IPC.
// Directory validation is enforced at both the listener and the dialer.
func shortSocketDir(uid int) string {
	return filepath.Join("/tmp", fmt.Sprintf("nexus-%d", uid))
}

// ensurePrivateSocketDir never follows a symlink at the socket directory
// itself. Existing directories must belong to the current user and have 0700
// permissions; fail closed rather than silently loosening access or using a
// different path in the client and server.
func ensurePrivateSocketDir(dir string, uid int) bool {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return false
	}
	st, err := os.Lstat(dir)
	return err == nil && st.IsDir() && st.Mode()&os.ModeSymlink == 0 &&
		st.Mode().Perm() == 0700 && isOwnedBy(st, uid)
}

// WaitForEndpoint polls until the control endpoint is ready or timeout expires.
func WaitForEndpoint(ctx context.Context, runtimeID string, timeout time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()
	for {
		attemptCtx, cancelAttempt := context.WithTimeout(waitCtx, dialAttemptTimeout)
		client, err := NewClientContext(attemptCtx, runtimeID)
		if err == nil {
			err = client.PingContext(attemptCtx)
			_ = client.Close()
		}
		cancelAttempt()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if waitCtx.Err() != nil {
			return fmt.Errorf("timed out waiting for endpoint %s", runtimeID)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-waitCtx.Done():
			return fmt.Errorf("timed out waiting for endpoint %s", runtimeID)
		case <-ticker.C:
		}
	}
}
