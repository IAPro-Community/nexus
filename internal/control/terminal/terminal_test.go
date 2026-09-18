//go:build !windows

package terminal

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestTerminalBackendExecution(t *testing.T) {
	backend := NewBackend()
	cmd := testEchoCommand("hello world")

	if err := backend.Start(cmd, 24, 80); err != nil {
		t.Fatalf("failed to start terminal backend: %v", err)
	}
	defer backend.Close()

	if backend.PID() <= 0 {
		t.Errorf("expected positive PID, got %d", backend.PID())
	}

	output := readBackendUntil(t, backend, "hello", 2*time.Second)

	if !strings.Contains(output, "hello") {
		t.Errorf("expected 'hello' in output, got %q", output)
	}

	_ = backend.Resize(30, 100)
}

func readBackendUntil(t *testing.T, backend Backend, match string, timeout time.Duration) string {
	t.Helper()
	chunks := make(chan string, 8)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := backend.Read(buf)
			if n > 0 {
				chunks <- string(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	var output strings.Builder
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		select {
		case chunk := <-chunks:
			output.WriteString(chunk)
			if strings.Contains(output.String(), match) {
				return output.String()
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for %q; output=%q", match, output.String())
		}
	}
}

func TestPrepareInteractiveCommandNormalizesDumbTerminal(t *testing.T) {
	cmd := testEchoCommand("ok")
	cmd.Env = []string{"TERM=dumb", "PATH=/bin"}

	prepareInteractiveCommand(cmd)

	if cmd.Env[0] != "TERM=xterm-256color" {
		t.Fatalf("expected TERM to be normalized, got %q", cmd.Env[0])
	}
}

func TestPrepareInteractiveCommandDoesNotMutateNormalTerminal(t *testing.T) {
	cmd := testEchoCommand("ok")
	cmd.Env = []string{"TERM=screen-256color"}

	prepareInteractiveCommand(cmd)

	if len(cmd.Env) != 1 || cmd.Env[0] != "TERM=screen-256color" {
		t.Fatalf("normal TERM changed: %+v", cmd.Env)
	}
}

func TestTerminalBackendStdin(t *testing.T) {
	backend := NewBackend()
	cmd := testCatCommand()

	if err := backend.Start(cmd, 24, 80); err != nil {
		t.Fatalf("failed to start terminal backend: %v", err)
	}
	defer backend.Close()

	testInput := "streaming test line\n"
	_, err := backend.Write([]byte(testInput))
	if err != nil {
		t.Fatalf("failed to write to backend: %v", err)
	}

	buf := make([]byte, 1024)
	n, err := backend.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("failed to read from backend: %v", err)
	}

	out := string(buf[:n])
	if !strings.Contains(out, "streaming test line") {
		t.Errorf("expected echo of input, got %q", out)
	}
}
