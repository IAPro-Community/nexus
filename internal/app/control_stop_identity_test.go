package app

import (
	"os"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func TestValidatedFallbackHostPIDUsesSessionHostIdentity(t *testing.T) {
	session := registry.RuntimeSession{
		PID:            99999999,
		HostPID:        os.Getpid(),
		HostGeneration: time.Now().UnixNano(),
	}
	pid, err := validatedFallbackHostPID(session)
	if err != nil {
		t.Fatalf("expected current SessionHost identity to validate: %v", err)
	}
	if pid != os.Getpid() {
		t.Fatalf("validated host pid = %d, want %d", pid, os.Getpid())
	}
}

func TestValidatedFallbackHostPIDDoesNotUseSurvivingProviderPID(t *testing.T) {
	session := registry.RuntimeSession{
		PID:            os.Getpid(),
		HostPID:        99999999,
		HostGeneration: time.Now().UnixNano(),
	}
	pid, err := validatedFallbackHostPID(session)
	if err != nil {
		t.Fatalf("dead SessionHost should not make provider PID a kill target: %v", err)
	}
	if pid != 0 {
		t.Fatalf("provider PID must not be returned as fallback kill target: %d", pid)
	}
}

func TestValidatedFallbackHostPIDFailsClosedForLegacyLiveProvider(t *testing.T) {
	session := registry.RuntimeSession{PID: os.Getpid()}
	pid, err := validatedFallbackHostPID(session)
	if err == nil {
		t.Fatal("expected legacy live provider without HostPID to fail closed")
	}
	if pid != 0 {
		t.Fatalf("legacy provider PID must not become a kill target: %d", pid)
	}
}
