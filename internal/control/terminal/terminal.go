package terminal

import (
	"context"
	"io"
	"os"
	"os/exec"
)

// prepareInteractiveCommand keeps provider TUIs from treating Nexus's
// browser PTY as a non-interactive terminal when the parent exported
// TERM=dumb. A real PTY is allocated by the backend immediately afterward.
func prepareInteractiveCommand(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	env := cmd.Env
	if env == nil {
		env = os.Environ()
	}
	for i, entry := range env {
		if entry == "TERM=dumb" {
			cmd.Env = append([]string(nil), env...)
			cmd.Env[i] = "TERM=xterm-256color"
			return
		}
	}
}

// Backend abstracts platform-specific terminal emulation (Unix PTY, Windows ConPTY).
type Backend interface {
	io.Reader
	io.Writer
	io.Closer

	// Start launches the process wrapped in the platform terminal subsystem.
	Start(cmd *exec.Cmd, initialRows, initialCols int) error

	// Resize updates the terminal dimensions if supported by the backend.
	Resize(rows, cols int) error

	// PID returns the process ID of the supervised child process.
	PID() int

	// Wait blocks until the supervised child process terminates.
	Wait() error

	// WaitContext blocks until the child terminates or ctx is canceled.
	// On context cancellation the child is forcefully killed.
	WaitContext(ctx context.Context) error

	// Signal delivers an interrupt-style signal to the child (graceful stop).
	Signal(sig os.Signal) error

	// Kill forcefully terminates the child process tree.
	Kill() error

	// SupportsResize indicates whether dynamic window resizing is active.
	SupportsResize() bool

	// SupportsRawMode indicates whether VT/raw mode is supported.
	SupportsRawMode() bool

	// Mechanism returns a truthful description of the terminal backend in use.
	Mechanism() string

	// Supervise attaches the child process tree to the backend's lifecycle
	// supervisor. It is a no-op on platforms without a native tree primitive.
	Supervise() error
}

// NewBackend instantiates the optimal terminal backend for the current OS.
func NewBackend() Backend {
	return newPlatformBackend()
}
