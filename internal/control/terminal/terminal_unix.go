//go:build !windows

package terminal

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

type unixPTYBackend struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	ptmx   *os.File
	inPipe io.WriteCloser
	rPipe  io.ReadCloser
	isPTY  bool
}

func newPlatformBackend() Backend {
	return &unixPTYBackend{}
}

// BackendMechanism reports the platform's primary terminal backend.
func BackendMechanism() string { return "PTY (creack/pty)" }

func (b *unixPTYBackend) Start(cmd *exec.Cmd, initialRows, initialCols int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	prepareInteractiveCommand(cmd)

	b.cmd = cmd
	if initialRows <= 0 {
		initialRows = 24
	}
	if initialCols <= 0 {
		initialCols = 80
	}

	size := &pty.Winsize{
		Rows: uint16(initialRows),
		Cols: uint16(initialCols),
	}

	ptmx, err := pty.StartWithSize(cmd, size)
	if err != nil {
		// Reset Stdin, Stdout, Stderr in case pty partially set them
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil
		if cmd.SysProcAttr == nil {
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		} else {
			cmd.SysProcAttr.Setsid = false
			cmd.SysProcAttr.Setctty = false
			cmd.SysProcAttr.Setpgid = true
		}

		// Fallback to standard pipes if PTY cannot be created (e.g. strict sandbox)
		inPipe, pipeErr := cmd.StdinPipe()
		if pipeErr != nil {
			return fmt.Errorf("failed to create stdin pipe: %w", pipeErr)
		}
		outPipe, pipeErr := cmd.StdoutPipe()
		if pipeErr != nil {
			return fmt.Errorf("failed to create stdout pipe: %w", pipeErr)
		}
		cmd.Stderr = cmd.Stdout

		if startErr := cmd.Start(); startErr != nil {
			return fmt.Errorf("failed to start process: %w", startErr)
		}

		b.inPipe = inPipe
		b.rPipe = outPipe
		b.isPTY = false
		return nil
	}

	b.ptmx = ptmx
	b.isPTY = true
	return nil
}

func (b *unixPTYBackend) Read(p []byte) (int, error) {
	b.mu.Lock()
	ptmx := b.ptmx
	rPipe := b.rPipe
	b.mu.Unlock()

	if ptmx != nil {
		return ptmx.Read(p)
	}
	if rPipe != nil {
		return rPipe.Read(p)
	}
	return 0, io.EOF
}

func (b *unixPTYBackend) Write(p []byte) (int, error) {
	b.mu.Lock()
	ptmx := b.ptmx
	inPipe := b.inPipe
	b.mu.Unlock()

	if ptmx != nil {
		return ptmx.Write(p)
	}
	if inPipe != nil {
		return inPipe.Write(p)
	}
	return 0, io.ErrClosedPipe
}

func (b *unixPTYBackend) Resize(rows, cols int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.isPTY || b.ptmx == nil || rows <= 0 || cols <= 0 {
		return nil
	}

	return pty.Setsize(b.ptmx, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
}

func (b *unixPTYBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.ptmx != nil {
		_ = b.ptmx.Close()
		b.ptmx = nil
	}
	if b.inPipe != nil {
		_ = b.inPipe.Close()
		b.inPipe = nil
	}
	if b.rPipe != nil {
		_ = b.rPipe.Close()
		b.rPipe = nil
	}
	return nil
}

func (b *unixPTYBackend) PID() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cmd != nil && b.cmd.Process != nil {
		return b.cmd.Process.Pid
	}
	return 0
}

func (b *unixPTYBackend) SupportsResize() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.isPTY
}

func (b *unixPTYBackend) SupportsRawMode() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.isPTY
}

func (b *unixPTYBackend) Mechanism() string {
	return "PTY (creack/pty)"
}

func (b *unixPTYBackend) Supervise() error { return nil }

func (b *unixPTYBackend) Wait() error {
	b.mu.Lock()
	cmd := b.cmd
	ptmx := b.ptmx
	b.mu.Unlock()
	if cmd == nil {
		return nil
	}
	err := cmd.Wait()
	if ptmx != nil {
		buf := make([]byte, 4096)
		for {
			_, readErr := ptmx.Read(buf)
			if readErr != nil {
				break
			}
		}
	}
	return err
}

func (b *unixPTYBackend) WaitContext(ctx context.Context) error {
	done := make(chan error, 1)
	go func() { done <- b.Wait() }()
	select {
	case <-ctx.Done():
		_ = b.Kill()
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// Signal delivers sig to the child process group.
func (b *unixPTYBackend) Signal(sig os.Signal) error {
	b.mu.Lock()
	cmd := b.cmd
	b.mu.Unlock()
	if cmd == nil || cmd.Process == nil || cmd.Process.Pid <= 0 {
		return nil
	}
	sysSig, ok := sig.(syscall.Signal)
	if !ok {
		sysSig = syscall.SIGTERM
	}
	if err := syscall.Kill(-cmd.Process.Pid, sysSig); err != nil {
		return cmd.Process.Signal(sig)
	}
	return nil
}

// Kill forcefully terminates the child process group.
func (b *unixPTYBackend) Kill() error {
	b.mu.Lock()
	cmd := b.cmd
	b.mu.Unlock()
	if cmd == nil || cmd.Process == nil || cmd.Process.Pid <= 0 {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		return cmd.Process.Kill()
	}
	return nil
}
