//go:build !windows

package executil

import "os/exec"

// HideConsoleWindow is a no-op outside Windows. Do not use for commands
// intended to own a terminal.
func HideConsoleWindow(*exec.Cmd) {}

// HideWindow is a no-op outside Windows.
func HideWindow(*exec.Cmd) {}
