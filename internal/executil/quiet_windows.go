//go:build windows

package executil

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// HideWindow prevents a child window from flashing while preserving the
// process's console and standard handles. It is suitable for interactive
// commands whose input/output are already attached to a terminal.
func HideWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	attr := cmd.SysProcAttr
	if attr == nil {
		attr = &syscall.SysProcAttr{}
	}
	attr.HideWindow = true
	cmd.SysProcAttr = attr
}

// HideConsoleWindow prevents non-interactive subprocesses from opening a
// visible console window. Do not use for commands intended to own a terminal.
func HideConsoleWindow(cmd *exec.Cmd) {
	HideWindow(cmd)
	if cmd != nil {
		cmd.SysProcAttr.CreationFlags |= createNoWindow
	}
}
