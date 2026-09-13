package host

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/codex"
)

type failingStartBackend struct{}

func (failingStartBackend) Start(*exec.Cmd, int, int) error {
	return errors.New("injected start failure")
}
func (failingStartBackend) Read([]byte) (int, error)          { return 0, io.EOF }
func (failingStartBackend) Write([]byte) (int, error)         { return 0, io.ErrClosedPipe }
func (failingStartBackend) Close() error                      { return nil }
func (failingStartBackend) Resize(int, int) error             { return nil }
func (failingStartBackend) PID() int                          { return 0 }
func (failingStartBackend) Wait() error                       { return nil }
func (failingStartBackend) WaitContext(context.Context) error { return nil }
func (failingStartBackend) Signal(os.Signal) error            { return nil }
func (failingStartBackend) Kill() error                       { return nil }
func (failingStartBackend) SupportsResize() bool              { return false }
func (failingStartBackend) SupportsRawMode() bool             { return false }
func (failingStartBackend) Mechanism() string                 { return "injected failure" }
func (failingStartBackend) Supervise() error                  { return nil }

func TestSessionHostStartFailureCompletesLifecycle(t *testing.T) {
	runtimeID := "rt-start-failure-completes"
	owned := registry.NewRegistry(t.TempDir() + "/runtimes.json")
	sh, err := NewSessionHost(Config{
		Session: registry.RuntimeSession{
			RuntimeID:  runtimeID,
			ProviderID: "test",
			State:      registry.StateStarting,
		},
		Registry:        owned,
		Binary:          testTerminalBinary,
		Args:            testTerminalArgs(),
		Env:             os.Environ(),
		Cwd:             os.TempDir(),
		terminalBackend: failingStartBackend{},
	})
	if err != nil {
		t.Fatalf("NewSessionHost: %v", err)
	}
	if err := sh.Start(); err == nil {
		t.Fatal("Start() error = nil, want injected terminal startup failure")
	}

	waitDone := make(chan struct{})
	go func() {
		sh.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
	case <-time.After(time.Second):
		t.Fatal("Wait blocked after Start failed")
	}

	if sh.session.State != registry.StateFailed {
		t.Fatalf("session state = %s, want %s", sh.session.State, registry.StateFailed)
	}
	if err := sh.Stop(); err != nil {
		t.Fatalf("Stop after failed Start: %v", err)
	}
}

func TestSessionHostCodexTUILockBusyFailsFast(t *testing.T) {
	home := t.TempDir()
	held, err := codex.AcquireTUILock(home)
	if err != nil {
		t.Fatalf("AcquireTUILock: %v", err)
	}
	defer func() { _ = held.Release() }()

	runtimeID := "rt-codex-tui-busy"
	owned := registry.NewRegistry(t.TempDir() + "/runtimes.json")
	env := append([]string{}, os.Environ()...)
	env = append(env, "CODEX_HOME="+home)
	sh, err := NewSessionHost(Config{
		Session: registry.RuntimeSession{
			RuntimeID:  runtimeID,
			ProviderID: "codex",
			ProfileID:  "busy",
			State:      registry.StateStarting,
		},
		Registry:        owned,
		Binary:          testTerminalBinary,
		Args:            testTerminalArgs(),
		Env:             env,
		Cwd:             os.TempDir(),
		terminalBackend: failingStartBackend{},
	})
	if err != nil {
		t.Fatalf("NewSessionHost: %v", err)
	}

	start := time.Now()
	err = sh.Start()
	if !errors.Is(err, codex.ErrTUIBusy) {
		t.Fatalf("Start err=%v want ErrTUIBusy", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("Start blocked for %s; busy Codex lock must fail immediately", time.Since(start))
	}

	waitDone := make(chan struct{})
	go func() {
		sh.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
	case <-time.After(time.Second):
		t.Fatal("Wait blocked after Codex TUI lock failure")
	}
	if sh.session.State != registry.StateFailed {
		t.Fatalf("session state = %s, want %s", sh.session.State, registry.StateFailed)
	}
}
