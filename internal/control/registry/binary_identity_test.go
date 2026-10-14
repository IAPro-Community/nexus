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
