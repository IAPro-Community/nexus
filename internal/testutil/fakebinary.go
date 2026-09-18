// Package testutil provides shared test helpers.
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// WriteFakeBinary writes a fake executable to dir/name. On Unix it creates a
// shell script; on Windows it compiles a minimal Go stub so the file is a
// real PE executable that Windows can find via LookPath.
//
// body is shell script content on Unix (ignored on Windows when Go source is
// used instead). Pass an empty string if you only need the binary to exist.
func WriteFakeBinary(t *testing.T, dir, name, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return writeWindowsStub(t, dir, name)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// WriteFakeBinaryWithScript writes a shell-script fake binary. On Windows the
// body is embedded into a Go stub that prints it as a comment and exits 0,
// so tests that only need the binary on PATH get a working executable.
func WriteFakeBinaryWithScript(t *testing.T, dir, name, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return writeWindowsStub(t, dir, name)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// writeWindowsStub compiles a tiny Go program that exits 0, producing a real
// .exe on Windows. This is necessary because shell scripts are not executable
// on Windows and tests that use os/exec.LookPath need a valid PE binary.
func writeWindowsStub(t *testing.T, dir, name string) string {
	t.Helper()
	src := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(src, []byte(`package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	out := os.Getenv("AI_TEST_OUT")
	if out == "" {
		return
	}
	provider := strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0]))
	homeKey := "HOME"
	if strings.EqualFold(provider, "codex") {
		homeKey = "CODEX_HOME"
	}
	cwd, _ := os.Getwd()
	f, err := os.Create(out)
	if err != nil {
		os.Exit(1)
	}
	defer f.Close()
	fmt.Fprintf(f, "provider=%s\nhome=%s\ncwd=%s\n", provider, os.Getenv(homeKey), cwd)
	for _, arg := range os.Args[1:] {
		fmt.Fprintf(f, "arg=%s\n", arg)
	}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, name+".exe")
	cmd := exec.Command("go", "build", "-o", exe, src)
	cmd.Dir = t.TempDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compile windows stub %s: %v\n%s", name, err, out)
	}
	return exe
}
