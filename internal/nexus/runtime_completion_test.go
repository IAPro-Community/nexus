package nexus

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func TestRuntimeCompletionOutcomeRequiresTerminalState(t *testing.T) {
	if err := runtimeCompletionOutcome(registry.StateStopped, true); err != nil {
		t.Fatalf("stopped runtime must be accepted: %v", err)
	}
	if err := runtimeCompletionOutcome(registry.StateFailed, true); err == nil {
		t.Fatal("failed runtime must be rejected")
	}
	if err := runtimeCompletionOutcome(registry.StateRunning, true); err == nil {
		t.Fatal("running runtime after stream closure must not be reported as success")
	}
	if err := runtimeCompletionOutcome("", false); err == nil {
		t.Fatal("missing runtime registry state must not be reported as success")
	}
}
