package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
)

// FakeCommandResponse describes one exact argv response for WriteFakeCommand.
type FakeCommandResponse struct {
	Args      []string
	Stdout    string
	Stderr    string
	ExitCode  int
	ReadStdin bool
}

type fakeCommandConfig struct {
	Responses          []FakeCommandResponse
	DefaultExitCode    int
	MarkerOnInvocation string
}

// WriteFakeCommand builds a tiny stdlib-only executable that matches argv
// exactly and returns deterministic stdout/stderr/exit codes on every OS.
// It is intended for tests whose behavior cannot be represented by the older
// PATH-only fake binary on Windows.
func WriteFakeCommand(t *testing.T, dir, name string, responses []FakeCommandResponse, defaultExitCode int, markerOnInvocation string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(t.TempDir(), "fake-command.json")
	raw, err := json.Marshal(fakeCommandConfig{
		Responses:          responses,
		DefaultExitCode:    defaultExitCode,
		MarkerOnInvocation: markerOnInvocation,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(t.TempDir(), "main.go")
	program := fmt.Sprintf(`package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type response struct {
	Args []string
	Stdout string
	Stderr string
	ExitCode int
	ReadStdin bool
}
type config struct {
	Responses []response
	DefaultExitCode int
	MarkerOnInvocation string
}
func equal(a, b []string) bool {
	if len(a) != len(b) { return false }
	for i := range a { if a[i] != b[i] { return false } }
	return true
}
func main() {
	raw, err := os.ReadFile(%s)
	if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(97) }
	var cfg config
	if err := json.Unmarshal(raw, &cfg); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(98) }
	if cfg.MarkerOnInvocation != "" {
		_ = os.WriteFile(cfg.MarkerOnInvocation, []byte("invoked"), 0600)
	}
	args := os.Args[1:]
	for _, item := range cfg.Responses {
		if !equal(args, item.Args) { continue }
		if item.ReadStdin { _, _ = io.Copy(io.Discard, os.Stdin) }
		if item.Stdout != "" { _, _ = fmt.Fprint(os.Stdout, item.Stdout) }
		if item.Stderr != "" { _, _ = fmt.Fprint(os.Stderr, item.Stderr) }
		os.Exit(item.ExitCode)
	}
	os.Exit(cfg.DefaultExitCode)
}
`, strconv.Quote(configPath))
	if err := os.WriteFile(src, []byte(program), 0o600); err != nil {
		t.Fatal(err)
	}

	outputName := name
	if runtime.GOOS == "windows" {
		outputName += ".exe"
	}
	outputPath := filepath.Join(dir, outputName)
	cmd := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-o", outputPath, src)
	cmd.Dir = t.TempDir()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile fake command %s: %v\n%s", name, err, out)
	}
	return outputPath
}
