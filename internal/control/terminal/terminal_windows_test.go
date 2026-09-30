//go:build windows

package terminal

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
	"unicode/utf16"
)

func TestUTF16EnvBlockIsDoubleNullTerminated(t *testing.T) {
	block := utf16EnvBlock([]string{"NEXUS_A=one", "NEXUS_B=two"})
	decoded := string(utf16.Decode(block))
	if !strings.HasSuffix(decoded, "\x00\x00") {
		t.Fatalf("environment block must end with two NULs: %q", decoded)
	}
	if strings.Contains(decoded[:len(decoded)-2], "\x00\x00") {
		t.Fatalf("environment block contains an early double NUL: %q", decoded)
	}
	if !strings.Contains(decoded, "NEXUS_A=one\x00NEXUS_B=two\x00") {
		t.Fatalf("environment entries were not encoded contiguously: %q", decoded)
	}
}

func TestUTF16EnvBlockSeparatesEntriesAndTerminatesOnce(t *testing.T) {
	block := utf16EnvBlock([]string{"FIRST=one", "SECOND=two"})
	first := syscall.StringToUTF16("FIRST=one")
	second := syscall.StringToUTF16("SECOND=two")
	if len(block) != len(first)+len(second)+1 {
		t.Fatalf("expected one terminator per entry plus final block terminator, got %d UTF-16 units", len(block))
	}
	for i, value := range first {
		if block[i] != value {
			t.Fatalf("first entry mismatch at %d: got %d, want %d", i, block[i], value)
		}
	}
	for i, value := range second {
		if block[len(first)+i] != value {
			t.Fatalf("second entry mismatch at %d: got %d, want %d", i, block[len(first)+i], value)
		}
	}
	if block[len(block)-1] != 0 {
		t.Fatal("environment block must end with an additional NUL")
	}
}

func TestUTF16EnvBlockEmptyIsDoubleNullTerminated(t *testing.T) {
	block := utf16EnvBlock(nil)
	if len(block) != 2 || block[0] != 0 || block[1] != 0 {
		t.Fatalf("empty environment block = %#v, want two NULs", block)
	}
}

func TestConPTYUsesOnlyPseudoConsoleProcessAttribute(t *testing.T) {
	if numAttributesForConPTY != 1 {
		t.Fatalf("ConPTY attribute count = %d, want 1; inherited pipe handles make CreateProcessW reject bInheritHandles=false", numAttributesForConPTY)
	}
}

func TestCreateProcessSpecWrapsWindowsShellShim(t *testing.T) {
	cmd := exec.Command(`C:\Users\fbolz\AppData\Roaming\npm\codex.cmd`, "--help")
	app, line := createProcessSpec(cmd)

	if app != os.Getenv("ComSpec") {
		t.Fatalf("expected ComSpec application, got %q", app)
	}
	if !strings.Contains(line, " /d /s /c ") {
		t.Fatalf("expected cmd.exe shell flags, got %q", line)
	}
	if !strings.Contains(line, `codex.cmd`) || !strings.Contains(line, `--help`) {
		t.Fatalf("expected shim and arguments in command line, got %q", line)
	}
}

func TestCreateProcessSpecLeavesRealExecutablesDirect(t *testing.T) {
	cmd := exec.Command(`C:\Windows\System32\cmd.exe`, "/C", "echo ok")
	app, line := createProcessSpec(cmd)
	if app != cmd.Path {
		t.Fatalf("expected direct application path, got %q", app)
	}
	if !strings.Contains(line, `cmd.exe`) || !strings.Contains(line, "echo ok") {
		t.Fatalf("expected executable and args in command line, got %q", line)
	}
	if strings.Contains(line, " /d /s /c ") {
		t.Fatalf("real executables must not be wrapped in ComSpec: %q", line)
	}
}

// TestWindowsBackendEcho verifies the Windows backend end-to-end.
func TestWindowsBackendInterruptWritesETXToProviderInput(t *testing.T) {
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer readPipe.Close()
	defer writePipe.Close()

	backend := &windowsBackend{inPipe: writePipe}
	if err := backend.Signal(os.Interrupt); err != nil {
		t.Fatalf("Signal(os.Interrupt) failed: %v", err)
	}

	buf := make([]byte, 1)
	if _, err := io.ReadFull(readPipe, buf); err != nil {
		t.Fatalf("reading interrupt byte failed: %v", err)
	}
	if buf[0] != 0x03 {
		t.Fatalf("interrupt byte = %#x, want ETX 0x03", buf[0])
	}
}

func TestWindowsBackendDefaultsToStandardPipes(t *testing.T) {
	t.Setenv("AI_CLI_ENABLE_CONPTY", "")
	t.Setenv("AI_CLI_DISABLE_CONPTY", "")
	backend := NewBackend()
	cmd := exec.Command("cmd.exe")
	if err := backend.Start(cmd, 24, 80); err != nil {
		t.Fatalf("failed to start Windows backend: %v", err)
	}
	defer backend.Close()
	if _, err := backend.Write([]byte("echo hello-conpty\r\nexit\r\n")); err != nil {
		t.Fatalf("failed to write command: %v", err)
	}

	if strings.HasPrefix(backend.Mechanism(), "ConPTY") {
		t.Fatalf("ConPTY must remain opt-in, got mechanism %q", backend.Mechanism())
	}
	if backend.SupportsResize() || backend.SupportsRawMode() {
		t.Fatalf("standard-pipe backend must report unsupported resize/raw mode")
	}

	buf := make([]byte, 4096)
	deadline := time.Now().Add(10 * time.Second)
	var out strings.Builder
	for time.Now().Before(deadline) {
		n, err := backend.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
		}
		if strings.Contains(out.String(), "hello-conpty") {
			break
		}
		if err == io.EOF {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !strings.Contains(out.String(), "hello-conpty") {
		t.Fatalf("expected Windows terminal output to contain hello-conpty, got %q", out.String())
	}

	// Resize round-trip must not error.
	if err := backend.Resize(30, 100); err != nil {
		t.Fatalf("Resize failed: %v", err)
	}

	if err := backend.Wait(); err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
}

// TestWindowsBackendInteractive verifies input is delivered to the child.
func TestWindowsBackendInteractiveConPTYOptIn(t *testing.T) {
	t.Setenv("AI_CLI_ENABLE_CONPTY", "1")
	t.Setenv("AI_CLI_DISABLE_CONPTY", "")
	backend := NewBackend()
	cmd := exec.Command("cmd.exe", "/C", "set /p X=prompt: && echo GOT-%X%")
	if err := backend.Start(cmd, 24, 80); err != nil {
		t.Fatalf("failed to start Windows backend: %v", err)
	}
	defer backend.Close()
	if !strings.HasPrefix(backend.Mechanism(), "ConPTY") {
		t.Skip("interactive line discipline requires ConPTY; standard pipes are validated by the echo test")
	}
	if _, err := backend.Write([]byte("ABC\r\n")); err != nil {
		t.Fatalf("failed to write interactive input: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	buf := make([]byte, 4096)
	var out strings.Builder
	for time.Now().Before(deadline) {
		n, err := backend.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
		}
		if strings.Contains(out.String(), "GOT-ABC") {
			break
		}
		if err == io.EOF {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !strings.Contains(out.String(), "GOT-ABC") {
		t.Fatalf("expected interactive echo GOT-ABC, got %q", out.String())
	}

	_ = backend.Signal(os.Interrupt)
	_ = backend.Kill()
}

func TestWindowsBackendKillUsesSupervisedJobObject(t *testing.T) {
	t.Setenv("AI_CLI_ENABLE_CONPTY", "")
	t.Setenv("AI_CLI_DISABLE_CONPTY", "")
	backend := NewBackend()
	cmd := exec.Command("cmd.exe", "/D", "/Q", "/C", "ping -n 30 127.0.0.1 >nul")
	if err := backend.Start(cmd, 24, 80); err != nil {
		t.Fatalf("failed to start Windows backend: %v", err)
	}
	defer backend.Close()
	if err := backend.Supervise(); err != nil {
		t.Fatalf("failed to supervise Windows backend: %v", err)
	}

	windows, ok := backend.(*windowsBackend)
	if !ok {
		t.Fatalf("backend type = %T, want *windowsBackend", backend)
	}
	windows.mu.Lock()
	job := windows.hJob
	windows.mu.Unlock()
	if job == 0 {
		t.Fatal("Supervise must attach the provider to a Job Object")
	}

	if err := backend.Kill(); err != nil {
		t.Fatalf("Kill must terminate the supervised Job Object: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := backend.WaitContext(ctx); err != nil {
		t.Fatalf("provider did not terminate after Job Object kill: %v", err)
	}
}
