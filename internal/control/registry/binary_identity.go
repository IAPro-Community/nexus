package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	stdruntime "runtime"
	"strings"
	"sync"
	"time"
)

// fingerprintCacheKey identifies one immutable on-disk version of a binary. A
// rewrite changes size or mtime (NTFS mtime resolution is 100ns), so a stale
// entry is never returned for replaced content.
type fingerprintCacheKey struct {
	path    string
	size    int64
	modTime time.Time
}

var (
	fingerprintCacheMu sync.Mutex
	fingerprintCache   = map[fingerprintCacheKey]string{}
)

// cachedBinaryFingerprint resolves a cached fingerprint for the exact on-disk
// version being read, when present.
func cachedBinaryFingerprint(path string, info os.FileInfo) (string, bool) {
	key := fingerprintCacheKey{
		path:    normalizedExecutablePath(path),
		size:    info.Size(),
		modTime: info.ModTime(),
	}
	fingerprintCacheMu.Lock()
	defer fingerprintCacheMu.Unlock()
	fingerprint, ok := fingerprintCache[key]
	return fingerprint, ok
}

func storeBinaryFingerprint(path string, info os.FileInfo, fingerprint string) {
	key := fingerprintCacheKey{
		path:    normalizedExecutablePath(path),
		size:    info.Size(),
		modTime: info.ModTime(),
	}
	fingerprintCacheMu.Lock()
	defer fingerprintCacheMu.Unlock()
	fingerprintCache[key] = fingerprint
}

// BinaryFingerprint is a content identity, not a timestamp. It detects an
// in-place upgrade at the same path as well as a removed executable. The
// content hash is memoized per on-disk version (path+size+mtime) so repeated
// attach validations of the same executable do not re-read a multi-megabyte
// binary on every WebSocket connection.
func BinaryFingerprint(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("binary path is empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if cached, ok := cachedBinaryFingerprint(path, info); ok {
		return cached, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	fingerprint := hex.EncodeToString(hash.Sum(nil))
	storeBinaryFingerprint(path, info, fingerprint)
	return fingerprint, nil
}

func normalizedExecutablePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	if absolute, err := filepath.Abs(path); err == nil {
		path = absolute
	}
	return filepath.Clean(path)
}

func sameExecutablePath(left, right string) bool {
	left = normalizedExecutablePath(left)
	right = normalizedExecutablePath(right)
	if left == "" || right == "" {
		return false
	}
	if stdruntime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

// ValidateBinaryIdentity verifies both existence and recorded content. A
// missing fingerprint is intentionally incompatible: sessions created before
// binary identity was persisted must be recreated rather than attached
// silently.
func ValidateBinaryIdentity(path, fingerprint string) error {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(fingerprint) == "" {
		return errors.New("binary identity is missing")
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("binary %q is unavailable: %w", path, err)
	}
	actual, err := BinaryFingerprint(path)
	if err != nil {
		return fmt.Errorf("read binary %q: %w", path, err)
	}
	if !strings.EqualFold(actual, strings.TrimSpace(fingerprint)) {
		return fmt.Errorf("binary %q changed since the runtime was created", path)
	}
	return nil
}

// validateHostBinaryIdentity compares the recorded host identity to the running
// Nexus binary. When both paths are the same, currentFingerprint already proves
// the on-disk identity, so avoid reading a large executable twice per attach.
func validateHostBinaryIdentity(currentPath, currentFingerprint, hostPath, hostFingerprint string) error {
	if !sameExecutablePath(currentPath, hostPath) {
		if err := ValidateBinaryIdentity(hostPath, hostFingerprint); err != nil {
			return fmt.Errorf("runtime host binary is incompatible: %w", err)
		}
		return errors.New("runtime host uses an older or different Nexus binary")
	}
	if strings.TrimSpace(hostFingerprint) == "" {
		return errors.New("binary identity is missing")
	}
	if !strings.EqualFold(currentFingerprint, strings.TrimSpace(hostFingerprint)) {
		return fmt.Errorf("binary %q changed since the runtime was created", hostPath)
	}
	return nil
}

// ValidateAttachCompatibility is the fail-closed attach gate shared by CLI
// and Web. It never kills the runtime; callers mark it stale and ask the user
// to recreate the session when it fails.
func ValidateAttachCompatibility(session RuntimeSession) error {
	if session.Transport == "mock" {
		return nil
	}
	if !session.IsActive() {
		return fmt.Errorf("runtime state is %s", session.State)
	}
	if !session.HostLive() {
		return errors.New("runtime host is not live or its generation no longer matches")
	}
	if session.HostPID <= 0 {
		return errors.New("runtime host PID is missing")
	}
	if session.PID <= 0 || !IsProcessAlive(session.PID) {
		return errors.New("runtime provider PID is not alive")
	}
	if err := ValidateBinaryIdentity(session.HostBinaryPath, session.HostBinaryFingerprint); err != nil {
		return fmt.Errorf("runtime host binary is incompatible: %w", err)
	}
	currentHost, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current Nexus binary: %w", err)
	}
	currentFingerprint, err := BinaryFingerprint(currentHost)
	if err != nil {
		return fmt.Errorf("read current Nexus binary: %w", err)
	}
	if err := validateHostBinaryIdentity(
		currentHost,
		currentFingerprint,
		session.HostBinaryPath,
		session.HostBinaryFingerprint,
	); err != nil {
		return err
	}
	if processPath, err := ProcessExecutablePath(session.HostPID); err == nil && processPath != "" && !sameExecutablePath(processPath, session.HostBinaryPath) {
		return fmt.Errorf("runtime host PID %d belongs to a different binary", session.HostPID)
	}
	if session.ProviderBinaryPath == "" || session.ProviderBinaryFingerprint == "" {
		return errors.New("provider binary identity is missing")
	}
	if err := ValidateBinaryIdentity(session.ProviderBinaryPath, session.ProviderBinaryFingerprint); err != nil {
		return fmt.Errorf("provider binary is incompatible: %w", err)
	}
	return nil
}
