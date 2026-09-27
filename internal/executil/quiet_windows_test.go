//go:build windows

package executil

import (
	"os/exec"
	"strings"
	"syscall"
	"testing"
)

func TestHideConsoleWindowPreservesExistingProcessFlags(t *testing.T) {
	cmd := exec.Command("cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200}
	HideConsoleWindow(cmd)

	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("expected hidden window")
	}
	if cmd.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatal("expected CREATE_NO_WINDOW")
	}
	if cmd.SysProcAttr.CreationFlags&0x00000200 == 0 {
		t.Fatal("existing creation flags were not preserved")
	}
}

func TestCommandSuppressesConsoleWindow(t *testing.T) {
	cmd := Command("cmd.exe", "/d", "/c", "exit", "0")
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("expected command to hide its window")
	}
	if cmd.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatal("expected command to use CREATE_NO_WINDOW")
	}
}

func TestCommandContextSuppressesConsoleWindow(t *testing.T) {
	cmd := CommandContext(t.Context(), "cmd.exe", "/d", "/c", "exit", "0")
	if !cmd.SysProcAttr.HideWindow || cmd.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatal("expected context command to suppress its console window")
	}
}

func TestCommandCapturesOutputWithoutChangingResult(t *testing.T) {
	cmd := Command("cmd.exe", "/d", "/q", "/c", "echo", "quiet-output")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run hidden command: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "quiet-output" {
		t.Fatalf("captured output = %q, want %q", got, "quiet-output")
	}
}

func TestHideWindowPreservesInteractiveConsoleFlags(t *testing.T) {
	cmd := exec.Command("cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200}
	HideWindow(cmd)
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("expected hidden window")
	}
	if cmd.SysProcAttr.CreationFlags&createNoWindow != 0 {
		t.Fatal("interactive process must not receive CREATE_NO_WINDOW")
	}
	if cmd.SysProcAttr.CreationFlags&0x00000200 == 0 {
		t.Fatal("existing creation flags were not preserved")
	}
}
