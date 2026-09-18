package doctor

import (
	"os"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func TestBuildTerminalReportDescribesRecoveryContract(t *testing.T) {
	t.Setenv("CODEX_TUI_DISABLE_KEYBOARD_ENHANCEMENT", "1")
	t.Setenv("NEXUS_RUNTIME_ID", "rt-current")

	report := BuildTerminalReport(-1, []registry.RuntimeSession{
		{RuntimeID: "rt-current", State: registry.StateStale, HostPID: 1234},
		{RuntimeID: "rt-stopped", State: registry.StateStopped},
	})

	if report.Schema != "nexus.doctor.terminal/v1" || report.AttachedRuntimeID != "rt-current" {
		t.Fatalf("unexpected report identity: %#v", report)
	}
	if report.CodexKeyboardEnhancement != "1" {
		t.Fatalf("expected enhancement guard to be visible, got %q", report.CodexKeyboardEnhancement)
	}
	for _, mode := range []string{"application_cursor", "kitty_csi_u", "modify_other_keys", "focus_reporting", "bracketed_paste"} {
		if report.KeyboardModes[mode] == "" {
			t.Fatalf("missing keyboard mode %q", mode)
		}
	}
	if len(report.Runtimes) != 1 || report.Runtimes[0].RuntimeID != "rt-current" {
		t.Fatalf("stale runtime should remain diagnosable, got %#v", report.Runtimes)
	}
	if report.Runtimes[0].Attachable || report.Runtimes[0].RecoveryAction == "" {
		t.Fatalf("incompatible runtime must have an actionable recovery: %#v", report.Runtimes[0])
	}
}

func TestRecoverTerminalIsSafeWithoutTTY(t *testing.T) {
	if err := RecoverTerminal(os.Stdout); err != nil {
		t.Fatalf("non-TTY recovery should be a no-op: %v", err)
	}
}
