package executil

import (
	"context"
	"os/exec"
)

// Command creates a non-interactive command without allowing it to flash a
// console window on Windows. Use exec.Command directly for interactive PTYs
// and HideWindow for GUI applications that should remain visible.
func Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	HideConsoleWindow(cmd)
	return cmd
}

// CommandContext is the context-aware form of Command.
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	HideConsoleWindow(cmd)
	return cmd
}
