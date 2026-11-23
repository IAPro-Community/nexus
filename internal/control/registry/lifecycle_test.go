package registry

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRuntimeStateTransitions(t *testing.T) {
	valid := []struct {
		from RuntimeState
		to   RuntimeState
	}{
		{StateStarting, StateRunning},
		{StateStarting, StateFailed},
		{StateRunning, StateWaiting},
		{StateRunning, StateStopping},
		{StateRunning, StateFailed},
		{StateStopping, StateStopped},
		{StateStopping, StateFailed},
		{StateStopped, StateStarting},
		{StateFailed, StateStarting},
	}
	for _, test := range valid {
		if !CanTransition(test.from, test.to) {
			t.Errorf("CanTransition(%q, %q) = false, want true", test.from, test.to)
		}
	}

	invalid := []struct {
		from RuntimeState
		to   RuntimeState
	}{
		{StateStarting, StateStopped},
		{StateStopped, StateRunning},
		{StateFailed, StateRunning},
		{StateStopping, StateWaiting},
	}
	for _, test := range invalid {
		if CanTransition(test.from, test.to) {
			t.Errorf("CanTransition(%q, %q) = true, want false", test.from, test.to)
		}
	}
}

func TestCanTransitionAllowsIdempotentStateUpdate(t *testing.T) {
	if !CanTransition(StateRunning, StateRunning) {
		t.Fatal("same-state update must be idempotent")
	}
}

func TestRuntimeHealthReportsUnknownForUnprovableSignals(t *testing.T) {
	now := time.Now().UTC()
	report := DiagnoseRuntimeHealth(RuntimeSession{RuntimeID: "rt-health-unknown"}, now)

	if report.RuntimeID != "rt-health-unknown" {
		t.Fatalf("runtime id = %q, want rt-health-unknown", report.RuntimeID)
	}
	if report.Lease.Status != HealthUnknown {
		t.Fatalf("lease status = %s, want UNKNOWN", report.Lease.Status)
	}
	if report.PTY.Status != HealthUnknown {
		t.Fatalf("pty status = %s, want UNKNOWN", report.PTY.Status)
	}
	if report.Heartbeat.Status != HealthUnknown {
		t.Fatalf("heartbeat status = %s, want UNKNOWN", report.Heartbeat.Status)
	}
	if report.PID.Status != HealthUnknown {
		t.Fatalf("pid status = %s, want UNKNOWN", report.PID.Status)
	}
	if report.ProviderCapability.Status != HealthError {
		t.Fatalf("provider capability status = %s, want ERROR", report.ProviderCapability.Status)
	}
}

func TestRuntimeHealthReportsErrorsWithoutClaimingRecovery(t *testing.T) {
	dead := exec.Command("sh", "-c", "exit 0")
	if err := dead.Start(); err != nil {
		t.Fatalf("start helper process: %v", err)
	}
	pid := dead.Process.Pid
	if err := dead.Wait(); err != nil {
		t.Fatalf("wait helper process: %v", err)
	}

	now := time.Now().UTC()
	report := DiagnoseRuntimeHealth(RuntimeSession{
		RuntimeID:          "rt-health-error",
		ProviderID:         "codex",
		State:              StateRunning,
		ControlLevel:       ControlLevelTerminal,
		StartupStage:       StartupTerminalStarting,
		LastFault:          StartupFaultConPTYStartFailed,
		PID:                pid,
		UpdatedAt:          now.Add(-10 * time.Minute),
		ProviderBinaryPath: "/nonexistent/codex",
	}, now)

	if report.PTY.Status != HealthError {
		t.Fatalf("pty status = %s, want ERROR", report.PTY.Status)
	}
	if report.PID.Status != HealthError {
		t.Fatalf("pid status = %s, want ERROR", report.PID.Status)
	}
	if report.Heartbeat.Status != HealthError {
		t.Fatalf("heartbeat status = %s, want ERROR", report.Heartbeat.Status)
	}
	for name, signal := range map[string]HealthSignal{
		"lease":               report.Lease,
		"pty":                 report.PTY,
		"heartbeat":           report.Heartbeat,
		"pid":                 report.PID,
		"provider_capability": report.ProviderCapability,
	} {
		if strings.Contains(strings.ToLower(signal.Detail), "recover") {
			t.Fatalf("%s detail claims recovery: %q", name, signal.Detail)
		}
	}
}

func TestRuntimeHealthReportsFreshContinuityLease(t *testing.T) {
	reg := NewRegistry(filepath.Join(t.TempDir(), "runtimes.json"))
	if err := reg.Register(RuntimeSession{
		RuntimeID:    "rt-health-lease",
		ProviderID:   "codex",
		State:        StateRunning,
		ControlLevel: ControlLevelTerminal,
		UpdatedAt:    time.Now().UTC(),
	}); err != nil {
		t.Fatalf("register runtime: %v", err)
	}
	if _, claimed, err := reg.ClaimContinuity("rt-health-lease"); err != nil || !claimed {
		t.Fatalf("claim continuity: claimed=%t err=%v", claimed, err)
	}
	session, ok := reg.Get("rt-health-lease")
	if !ok {
		t.Fatal("runtime not found after claim")
	}

	report := DiagnoseRuntimeHealth(session, time.Now().UTC())
	if report.Lease.Status != HealthOK {
		t.Fatalf("lease status = %s, want OK", report.Lease.Status)
	}
	if report.Heartbeat.Status != HealthOK {
		t.Fatalf("heartbeat status = %s, want OK", report.Heartbeat.Status)
	}
}

func TestRuntimeHealthReportsStaleContinuityLeaseHeartbeat(t *testing.T) {
	reg := NewRegistry(filepath.Join(t.TempDir(), "runtimes.json"))
	if err := reg.Register(RuntimeSession{
		RuntimeID:  "rt-health-stale-lease",
		ProviderID: "codex",
		State:      StateRunning,
		UpdatedAt:  time.Now().UTC(),
	}); err != nil {
		t.Fatalf("register runtime: %v", err)
	}
	if _, claimed, err := reg.ClaimContinuity("rt-health-stale-lease"); err != nil || !claimed {
		t.Fatalf("claim continuity: claimed=%t err=%v", claimed, err)
	}
	if err := reg.UpdateLabels("rt-health-stale-lease", map[string]string{
		ContinuityClaimedAtLabel: time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("age continuity lease: %v", err)
	}
	session, ok := reg.Get("rt-health-stale-lease")
	if !ok {
		t.Fatal("runtime not found after claim")
	}

	report := DiagnoseRuntimeHealth(session, time.Now().UTC())
	if report.Lease.Status != HealthError {
		t.Fatalf("lease status = %s, want ERROR", report.Lease.Status)
	}
	if report.Heartbeat.Status != HealthError {
		t.Fatalf("heartbeat status = %s, want ERROR", report.Heartbeat.Status)
	}
}
