//go:build !windows

package protocol

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// longRuntimeID mimics the real runtime identifier: "runtime_" + ULID (26
// chars), which is what pushed the macOS socket path past sun_path.
const longRuntimeID = "runtime_01JCX9Z3M4N5P6Q7R8S9T0V1W2"

func TestEndpointPathFitsKernelLimit(t *testing.T) {
	path := EndpointPath(longRuntimeID)
	if len(path) > maxUnixSocketPathLen {
		t.Fatalf("socket path is %d bytes (limit %d): %s", len(path), maxUnixSocketPathLen, path)
	}
	if got := EndpointPath(longRuntimeID); got != path {
		t.Fatalf("EndpointPath is not deterministic: %q vs %q", path, got)
	}
}

func TestEndpointPathShortensLongTempDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix sockets only")
	}
	// Simulate the macOS per-user TMPDIR that broke Listen with EINVAL.
	longDir := filepath.Join(t.TempDir(), strings.Repeat("very-long-temp-segment-", 3))
	if err := os.MkdirAll(longDir, 0700); err != nil {
		t.Fatalf("failed to create long temp dir: %v", err)
	}
	t.Setenv("TMPDIR", longDir)
	t.Setenv("XDG_RUNTIME_DIR", "")

	path := EndpointPath(longRuntimeID)
	if len(path) > maxUnixSocketPathLen {
		t.Fatalf("socket path is %d bytes (limit %d): %s", len(path), maxUnixSocketPathLen, path)
	}
	if strings.HasPrefix(path, longDir) {
		t.Fatalf("expected fallback directory, got %q inside %q", path, longDir)
	}

	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("socket directory %q does not exist: %v", dir, err)
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Fatalf("socket directory %q has mode %o, want 0700", dir, perm)
	}
	if !isOwnedBy(info, os.Getuid()) {
		t.Fatalf("socket directory %q is not owned by the current user", dir)
	}
}

// TestListenDialWithLongTempDir reproduces the macOS bind EINVAL regression.
func TestListenDialWithLongTempDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix sockets only")
	}
	longDir := filepath.Join(t.TempDir(), strings.Repeat("very-long-temp-segment-", 3))
	if err := os.MkdirAll(longDir, 0700); err != nil {
		t.Fatalf("failed to create long temp dir: %v", err)
	}
	t.Setenv("TMPDIR", longDir)
	t.Setenv("XDG_RUNTIME_DIR", "")

	listener, err := Listen(longRuntimeID)
	if err != nil {
		t.Fatalf("Listen failed with long temp dir: %v", err)
	}
	defer listener.Close()

	conn, err := Dial(longRuntimeID)
	if err != nil {
		t.Fatalf("Dial failed with long temp dir: %v", err)
	}
	defer conn.Close()
}

// TestPrivateSocketDirectoryRefusesUnsafePaths ensures a pre-existing
// permissive directory or symlink cannot host the local control socket.
func TestPrivateSocketDirectoryRefusesUnsafePaths(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "socket")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if !ensurePrivateSocketDir(dir, os.Getuid()) {
		t.Fatal("expected private owned directory to be accepted")
	}
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if ensurePrivateSocketDir(dir, os.Getuid()) {
		t.Fatal("world-readable directory must not be accepted")
	}
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if ensurePrivateSocketDir(link, os.Getuid()) {
		t.Fatal("symlinked socket directory must not be accepted")
	}
}

func TestEndpointPathRejectsInsecureXDGDirectory(t *testing.T) {
	xdg := t.TempDir()
	dir := filepath.Join(xdg, "nexus")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_RUNTIME_DIR", xdg)
	path := EndpointPath(longRuntimeID)
	if strings.HasPrefix(path, dir+string(os.PathSeparator)) {
		t.Fatalf("socket must not be placed in insecure XDG directory: %q", path)
	}
	if got := shortSocketDir(os.Getuid()); got != shortSocketDir(os.Getuid()) {
		t.Fatalf("short fallback must be stable across processes: %q", got)
	}
}
