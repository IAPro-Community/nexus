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
