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

// TUILockAcquireTimeout bounds how long launch/Prepare wait for a competing
// Codex TUI. Leftover SessionHosts used to hold this lock forever.
const TUILockAcquireTimeout = 2 * time.Second

// ErrTUIBusy is returned when another Codex process already owns CODEX_HOME.
var ErrTUIBusy = errors.New("codex TUI already active for this profile")

// TUILock holds an exclusive flock on a Codex profile home so quota probes
// cannot spawn a competing app-server against the same CODEX_HOME.
type TUILock struct {
	file *os.File
	path string
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
	codexHome = strings.TrimSpace(codexHome)
	if codexHome == "" {
		return nil, false, fmt.Errorf("empty CODEX_HOME for TUI lock")
	}
	if err := os.MkdirAll(codexHome, 0700); err != nil {
		return nil, false, err
	}
	path := tuiLockPath(codexHome)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, false, fmt.Errorf("open TUI lock: %w", err)
	}
	if err := flockExclusive(f, false); err != nil {
		_ = f.Close()
		if isLockBusy(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("try TUI lock: %w", err)
	}
	return &TUILock{file: f, path: path}, true, nil
}

// AcquireTUILockTimeout retries a non-blocking exclusive lock until timeout.
// A held lock returns ErrTUIBusy instead of waiting indefinitely.
func AcquireTUILockTimeout(codexHome string, timeout time.Duration) (*TUILock, error) {
	deadline := time.Now().Add(timeout)
	for {
		lock, ok, err := TryAcquireTUILock(codexHome)
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

// IsTUILocked reports whether the interactive TUI currently owns codexHome.
// It never blocks: a held lock returns true; an available lock is acquired and
// immediately released so probes can detect occupancy without waiting.
func IsTUILocked(codexHome string) bool {
	lock, ok, err := TryAcquireTUILock(codexHome)
	if err != nil || !ok {
		return err == nil && !ok
	}
	_ = lock.Release()
	return false
}

// Release drops the exclusive lock. Safe to call on a nil receiver.
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
