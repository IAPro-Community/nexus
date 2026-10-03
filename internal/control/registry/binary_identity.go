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
)

// BinaryFingerprint is a content identity, not a timestamp. It detects an
// in-place upgrade at the same path as well as a removed executable.
func BinaryFingerprint(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("binary path is empty")
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
	return hex.EncodeToString(hash.Sum(nil)), nil
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

// ValidateAttachCompatibility is the fail-closed attach gate shared by CLI
// and Web. It never kills the runtime; callers mark it stale and ask the user
// to recreate the session when it fails.
func ValidateAttachCompatibility(session RuntimeSession) error {
	if session.Transport == TransportMock {
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
	if !sameExecutablePath(currentHost, session.HostBinaryPath) || currentFingerprint != session.HostBinaryFingerprint {
		return errors.New("runtime host uses an older or different Nexus binary")
	}
	if processPath, err := ProcessExecutablePath(session.HostPID); err == nil && processPath != "" && !sameExecutablePath(processPath, session.HostBinaryPath) {
		return fmt.Errorf("runtime host PID %d belongs to a different binary", session.HostPID)
	}
	if IsRemoteRuntimeTransport(session.Transport) {
		if strings.TrimSpace(session.MachineID) == "" || strings.TrimSpace(session.Location) == "" {
			return errors.New("remote runtime identity is incomplete")
		}
		return nil
	}
	if session.PID <= 0 || !IsProcessAlive(session.PID) {
		return errors.New("runtime provider PID is not alive")
	}
	if session.ProviderBinaryPath == "" || session.ProviderBinaryFingerprint == "" {
		return errors.New("provider binary identity is missing")
	}
	if err := ValidateBinaryIdentity(session.ProviderBinaryPath, session.ProviderBinaryFingerprint); err != nil {
		return fmt.Errorf("provider binary is incompatible: %w", err)
	}
	return nil
}
