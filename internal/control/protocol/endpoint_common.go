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

	// Prefer XDG_RUNTIME_DIR if set and accessible
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		dir := filepath.Join(xdg, "nexus")
		_ = os.MkdirAll(dir, 0700)
		if path, ok := socketPathIn(dir, runtimeID); ok {
			return path
		}
	}

	// Fallback to /tmp/nexus-control-<uid>
	uid := os.Getuid()
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("nexus-control-%d", uid))
	_ = os.MkdirAll(dir, 0700)

	if st, err := os.Stat(dir); err == nil && !isOwnedBy(st, uid) {
		dir = filepath.Join(os.TempDir(), fmt.Sprintf("nexus-control-%d-%d", uid, time.Now().UnixNano()))
		_ = os.MkdirAll(dir, 0700)
	}

	if path, ok := socketPathIn(dir, runtimeID); ok {
		return path
	}

	// The per-user TMPDIR on macOS lives under /var/folders/... and can push
	// the socket path past the kernel limit once a ULID runtime ID is
	// appended. Fall back to a short, owner-checked private directory.
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

// shortSocketDir returns a stable, user-owned directory with a short path.
func shortSocketDir(uid int) string {
	dir := filepath.Join("/tmp", fmt.Sprintf("nexus-%d", uid))
	if err := os.MkdirAll(dir, 0700); err == nil {
		if st, statErr := os.Stat(dir); statErr == nil && isOwnedBy(st, uid) {
			return dir
		}
	}
	// Refuse to reuse a directory we do not own; nest one level deeper under a
	// per-process suffix instead of trusting the existing entry.
	dir = filepath.Join("/tmp", fmt.Sprintf("nexus-%d-%d", uid, os.Getpid()))
	_ = os.MkdirAll(dir, 0700)
	return dir
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
