package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func TestMonitorModelIsReadOnlyAndDoesNotExposeActions(t *testing.T) {
	model := NewMonitorModel("runtime-monitor")
	model.status = protocol.StatusData{RuntimeID: "runtime-monitor", ProviderID: "codex", ProfileID: "work", State: "RUNNING", PID: 42}
	view := model.View()
	if !strings.Contains(view, "Read-only sidecar") {
		t.Fatalf("monitor view did not identify read-only mode: %s", view)
	}
	if strings.Contains(view, "Stop") || strings.Contains(view, "Attach") || strings.Contains(view, "lease") && !strings.Contains(view, "no PTY attach") {
		t.Fatalf("monitor exposed a destructive or interactive action: %s", view)
	}
	_, cmd := model.Update(monitorTick(time.Now()))
	if cmd == nil {
		t.Fatal("monitor did not schedule a controlled poll")
	}
}

func TestControlModelUsesInjectedControlRegistries(t *testing.T) {
	runtimeRegistry := registry.NewRegistry("")
	driverRegistry := driver.NewRegistry()
	model := NewControlModelWithDependencies(runtimeRegistry, driverRegistry)

	model.runtimes = []registry.RuntimeSession{{
		RuntimeID:  "runtime-injected",
		ProviderID: "codex",
		ProfileID:  "work",
	}}
	model.selectedIndex = 0

	if model.runtimeReg != runtimeRegistry || model.drivers != driverRegistry {
		t.Fatal("control model did not retain injected registries")
	}
	if strings.Contains(model.View(), "[h] Handoff") {
		t.Fatal("TUI exposed handoff capability from a global driver registry")
	}
}
