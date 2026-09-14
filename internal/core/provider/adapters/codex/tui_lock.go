package codex

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const tuiLockFileName = "nexus-tui.lock"

// TUILockAcquireTimeout bounds compatibility callers waiting for profile
// maintenance to finish. Interactive launches use non-blocking shared leases.
const TUILockAcquireTimeout = 2 * time.Second

// ErrTUIBusy is returned when a maintenance operation cannot obtain CODEX_HOME.
var ErrTUIBusy = errors.New("codex profile lock is busy")

// TUILock holds an OS lock on a Codex profile home. TUI leases are shared;
// maintenance locks are exclusive.
type TUILock struct {
	file *os.File
	path string
}

func acquireLock(codexHome string, shared, blocking bool) (*TUILock, bool, error) {
	codexHome = strings.TrimSpace(codexHome)
	if codexHome == "" {
		return nil, false, fmt.Errorf("empty CODEX_HOME for lock")
	}
	if err := os.MkdirAll(codexHome, 0700); err != nil {
		return nil, false, err
	}
	f, err := os.OpenFile(tuiLockPath(codexHome), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, false, fmt.Errorf("open Codex profile lock: %w", err)
	}
	var lockErr error
	if shared {
		lockErr = flockShared(f, blocking)
	} else {
		lockErr = flockExclusive(f, blocking)
	}
	if lockErr != nil {
		_ = f.Close()
		if isLockBusy(lockErr) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("acquire Codex profile lock: %w", lockErr)
	}
	return &TUILock{file: f, path: tuiLockPath(codexHome)}, true, nil
}

// AcquireTUILease acquires a shared lease for one interactive runtime. Any
// number of TUIs may hold leases for the same profile concurrently.
func AcquireTUILease(codexHome string) (*TUILock, error) {
	return AcquireTUILeaseTimeout(codexHome, 0)
}

// TryAcquireTUILease attempts a non-blocking shared TUI lease.
func TryAcquireTUILease(codexHome string) (*TUILock, bool, error) {
	return acquireLock(codexHome, true, false)
}

// AcquireTUILeaseTimeout waits up to timeout for maintenance to finish.
func AcquireTUILeaseTimeout(codexHome string, timeout time.Duration) (*TUILock, error) {
	return acquireWithTimeout(codexHome, timeout, true)
}

// AcquireMaintenanceLock acquires the exclusive lock used by profile
// preparation, migration and app-server quota probes.
func AcquireMaintenanceLock(codexHome string) (*TUILock, error) {
	return acquireWithTimeout(codexHome, 0, false)
}

// TryAcquireMaintenanceLock attempts a non-blocking exclusive maintenance lock.
func TryAcquireMaintenanceLock(codexHome string) (*TUILock, bool, error) {
	return acquireLock(codexHome, false, false)
}

func acquireWithTimeout(codexHome string, timeout time.Duration, shared bool) (*TUILock, error) {
	deadline := time.Now().Add(timeout)
	for {
		lock, ok, err := acquireLock(codexHome, shared, false)
		if err != nil {
			return nil, err
		}
		if ok {
			return lock, nil
		}
		if timeout <= 0 || !time.Now().Before(deadline) {
			return nil, fmt.Errorf("%w", ErrTUIBusy)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func tuiLockPath(codexHome string) string {
	return filepath.Join(strings.TrimSpace(codexHome), tuiLockFileName)
}

// AcquireTUILock acquires the exclusive lock for codexHome without blocking
// forever. A competing TUI returns ErrTUIBusy immediately.
// The caller must Release when the interactive Codex process exits.
func AcquireTUILock(codexHome string) (*TUILock, error) {
	return AcquireTUILockTimeout(codexHome, 0)
}

// TryAcquireTUILock attempts a non-blocking exclusive lock. ok is false when
// another process (typically the interactive TUI) already holds it.
func TryAcquireTUILock(codexHome string) (lock *TUILock, ok bool, err error) {
	return TryAcquireMaintenanceLock(codexHome)
}

// AcquireTUILockTimeout retries a non-blocking exclusive lock until timeout.
// A held lock returns ErrTUIBusy instead of waiting indefinitely.
func AcquireTUILockTimeout(codexHome string, timeout time.Duration) (*TUILock, error) {
	return acquireWithTimeout(codexHome, timeout, false)
}

// IsTUILocked reports whether any TUI lease prevents an exclusive maintenance
// operation on codexHome.
// It never blocks: a held lock returns true; an available lock is acquired and
// immediately released so probes can detect occupancy without waiting.
func IsTUILocked(codexHome string) bool {
	lock, ok, err := TryAcquireMaintenanceLock(codexHome)
	if err != nil || !ok {
		return err == nil && !ok
	}
	_ = lock.Release()
	return false
}

// Release drops the held lease or maintenance lock. Safe on nil receivers.
func (l *TUILock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	err := flockUnlock(l.file)
	closeErr := l.file.Close()
	l.file = nil
	if err != nil {
		return err
	}
	return closeErr
}

// EnvCODEXHome extracts CODEX_HOME from an environment slice.
func EnvCODEXHome(env []string) string {
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if ok && key == "CODEX_HOME" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
