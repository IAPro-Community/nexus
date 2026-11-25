//go:build !windows

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/creack/pty"
)

func TestRunInteractiveProvidesTTYAndForwardsInput(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	originalStdin, originalStdout := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = slave, slave
	t.Cleanup(func() {
		os.Stdin, os.Stdout = originalStdin, originalStdout
		_ = slave.Close()
		_ = master.Close()
	})

	bin := filepath.Join(t.TempDir(), "interactive-provider")
	script := "#!/bin/sh\n" +
		"if [ -t 0 ] && [ -t 1 ] && [ -t 2 ]; then printf 'TTY_READY\\n'; else printf 'NO_TTY\\n'; exit 42; fi\n" +
		"IFS= read -r input\n" +
		"printf 'INPUT:%s\\n' \"$input\"\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}

	chunks := make(chan string, 16)
	go func() {
		buffer := make([]byte, 1024)
		for {
			n, readErr := master.Read(buffer)
			if n > 0 {
				chunks <- string(buffer[:n])
			}
			if readErr != nil {
				return
			}
		}
	}()

	type runResult struct {
		failure model.Failure
		err     error
	}
	done := make(chan runResult, 1)
	cwd := t.TempDir()
	go func() {
		failure, runErr := RunInteractive(bin, nil, os.Environ(), cwd, "agy")
		done <- runResult{failure: failure, err: runErr}
	}()

	var output strings.Builder
	waitFor := func(needle string) {
		t.Helper()
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		for !strings.Contains(output.String(), needle) {
			select {
			case chunk := <-chunks:
				output.WriteString(chunk)
			case result := <-done:
				t.Fatalf("interactive command exited before %q: failure=%+v err=%v output=%q", needle, result.failure, result.err, output.String())
			case <-timer.C:
				t.Fatalf("timed out waiting for %q; output=%q", needle, output.String())
			}
		}
	}

	waitFor("TTY_READY")
	if _, err := master.Write([]byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	waitFor("INPUT:hello")
	select {
	case result := <-done:
		if result.err != nil || result.failure.Kind != model.FailureNone {
			t.Fatalf("interactive command failed: failure=%+v err=%v", result.failure, result.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("interactive command did not exit after input")
	}
}
