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

// EndpointPath returns the canonical local socket / pipe path for a runtime.
func EndpointPath(runtimeID string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf(`\\.\pipe\nexus-control-%s`, runtimeID)
	}

	// Prefer XDG_RUNTIME_DIR if set and accessible
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		dir := filepath.Join(xdg, "nexus")
		_ = os.MkdirAll(dir, 0700)
		return filepath.Join(dir, fmt.Sprintf("%s.sock", runtimeID))
	}

	// Fallback to /tmp/nexus-control-<uid>
	uid := os.Getuid()
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("nexus-control-%d", uid))
	_ = os.MkdirAll(dir, 0700)

	if st, err := os.Stat(dir); err == nil && !isOwnedBy(st, uid) {
		dir = filepath.Join(os.TempDir(), fmt.Sprintf("nexus-control-%d-%d", uid, time.Now().UnixNano()))
		_ = os.MkdirAll(dir, 0700)
	}

	return filepath.Join(dir, fmt.Sprintf("%s.sock", runtimeID))
}

// WaitForEndpoint polls until the control endpoint is ready or timeout expires.
func WaitForEndpoint(ctx context.Context, runtimeID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()
	for {
		client, err := NewClient(runtimeID)
		if err == nil {
			err = client.Ping()
			_ = client.Close()
			if err == nil {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for endpoint %s", runtimeID)
		}
	}
}
