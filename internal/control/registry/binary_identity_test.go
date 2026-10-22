package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBinaryFingerprintDetectsInPlaceReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "provider")
	if err := os.WriteFile(path, []byte("old"), 0700); err != nil {
		t.Fatal(err)
	}
	fingerprint, err := BinaryFingerprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateBinaryIdentity(path, fingerprint); err != nil {
		t.Fatalf("fresh identity rejected: %v", err)
	}
	if err := os.WriteFile(path, []byte("new"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := ValidateBinaryIdentity(path, fingerprint); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("expected in-place replacement to be rejected, got %v", err)
	}
}

func TestValidateHostBinaryIdentitySamePathUsesCurrentFingerprint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nexus")
	if err := os.WriteFile(path, []byte("nexus-binary"), 0700); err != nil {
		t.Fatal(err)
	}
	fingerprint, err := BinaryFingerprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateHostBinaryIdentity(path, fingerprint, path, fingerprint); err != nil {
		t.Fatalf("matching same-path host identity rejected: %v", err)
	}
	if err := validateHostBinaryIdentity(path, fingerprint, path, "different"); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("expected changed same-path host identity to be rejected, got %v", err)
	}
	otherPath := filepath.Join(t.TempDir(), "nexus")
	if err := os.WriteFile(otherPath, []byte("nexus-binary"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := validateHostBinaryIdentity(path, fingerprint, otherPath, fingerprint); err == nil || !strings.Contains(err.Error(), "older or different") {
		t.Fatalf("expected different host path to be rejected, got %v", err)
	}
}

func TestValidateAttachCompatibilityRejectsLegacyIdentity(t *testing.T) {
	session := RuntimeSession{RuntimeID: "legacy", State: StateRunning, Transport: "ipc", PID: os.Getpid(), HostPID: os.Getpid(), HostGeneration: time.Now().UnixNano()}
	err := ValidateAttachCompatibility(session)
	if err == nil || !strings.Contains(err.Error(), "binary") {
		t.Fatalf("expected legacy runtime identity rejection, got %v", err)
	}
}

func TestValidateAttachCompatibilityManagedSSHProxySkipsRemoteProviderPID(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := BinaryFingerprint(exe)
	if err != nil {
		t.Fatal(err)
	}
	session := RuntimeSession{
		RuntimeID:             "remote-runtime",
		State:                 StateRunning,
		Transport:             TransportManagedSSHProxy,
		PID:                   1 << 30, // remote PID: deliberately nonexistent locally
		HostPID:               os.Getpid(),
		HostGeneration:        time.Now().UnixNano(),
		HostBinaryPath:        exe,
		HostBinaryFingerprint: fingerprint,
		MachineID:             "remote-machine",
		Location:              "devbox",
	}
	if err := ValidateAttachCompatibility(session); err != nil {
		t.Fatalf("managed SSH proxy rejected because remote provider PID is not local: %v", err)
	}

	session.MachineID = ""
	if err := ValidateAttachCompatibility(session); err == nil || !strings.Contains(err.Error(), "remote runtime identity") {
		t.Fatalf("incomplete remote identity must fail closed, got %v", err)
	}
}

// TestBinaryFingerprintMemoizesPerOnDiskVersion pins the attach-path
// optimization: repeated hashing of the same on-disk version (same path, size
// and mtime) must not re-read the binary, while a genuinely different version
// re-hashes and a removed binary still fails closed.
func TestBinaryFingerprintMemoizesPerOnDiskVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nexus")
	if err := os.WriteFile(path, []byte("nexus-binary-v1"), 0700); err != nil {
		t.Fatal(err)
	}
	first, err := BinaryFingerprint(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	// Same on-disk version (identical size and mtime): the memo returns the
	// hash it already computed instead of reading the file again.
	if err := os.WriteFile(path, []byte("nexus-binary-vX"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	cached, err := BinaryFingerprint(path)
	if err != nil {
		t.Fatalf("memoized fingerprint must resolve, got %v", err)
	}
	if cached != first {
		t.Fatalf("memoized fingerprint changed: %q != %q", cached, first)
	}

	// A different on-disk version must invalidate the memo.
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
	second, err := BinaryFingerprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if second == first {
		t.Fatal("fingerprint of a different on-disk version must differ")
	}

	// A removed binary still fails closed.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := BinaryFingerprint(path); err == nil {
		t.Fatal("expected missing binary to fail the fingerprint read")
	}
}
