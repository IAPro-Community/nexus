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

func TestValidateAttachCompatibilityRejectsLegacyIdentity(t *testing.T) {
	session := RuntimeSession{RuntimeID: "legacy", State: StateRunning, Transport: "ipc", PID: os.Getpid(), HostPID: os.Getpid(), HostGeneration: time.Now().UnixNano()}
	err := ValidateAttachCompatibility(session)
	if err == nil || !strings.Contains(err.Error(), "binary") {
		t.Fatalf("expected legacy runtime identity rejection, got %v", err)
	}
}
