// Package oauthutil provides shared building blocks for OAuth token refreshers.
package oauthutil

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const revokedSentinelName = ".oauth-revoked"

var profileMutexes sync.Map

// MutexFor returns the process-local mutex associated with key.
func MutexFor(key string) *sync.Mutex {
	actual, _ := profileMutexes.LoadOrStore(key, &sync.Mutex{})
	return actual.(*sync.Mutex)
}

// AtomicWriteFile writes data to a temporary file next to path and renames it
// into place. If path exists, it is retained as path+".bak".
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	backupPath := path + ".bak"
	_ = os.Remove(backupPath)
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, backupPath); err != nil {
			return err
		}
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Rename(backupPath, path)
		return err
	}
	return nil
}

// WriteRevokedSentinel records why OAuth credentials in dir were revoked.
func WriteRevokedSentinel(dir, reason string) error {
	return AtomicWriteFile(
		filepath.Join(dir, revokedSentinelName),
		[]byte(strings.TrimSpace(reason)+"\n"),
		0600,
	)
}

// IsRevoked reports whether dir contains a revoked sentinel and its reason.
func IsRevoked(dir string) (bool, string) {
	data, err := os.ReadFile(filepath.Join(dir, revokedSentinelName))
	if err != nil {
		return false, ""
	}
	return true, strings.TrimSpace(string(data))
}

// HTTPClient returns an HTTP client configured with timeout.
func HTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

// KillSwitch reports whether envKey explicitly disables a capability.
func KillSwitch(envKey string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envKey))) {
	case "0", "false":
		return true
	default:
		return false
	}
}
