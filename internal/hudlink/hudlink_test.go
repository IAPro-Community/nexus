package hudlink

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnabled(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{"", true},
		{"1", true},
		{"true", true},
		{"0", false},
		{"false", false},
		{"off", false},
		{" NO ", false},
		{"disabled", false},
	}
	for _, tc := range cases {
		t.Setenv(SpawnEnv, tc.value)
		if got := Enabled(); got != tc.want {
			t.Fatalf("Enabled with %q = %v, want %v", tc.value, got, tc.want)
		}
	}
}

func TestLinked(t *testing.T) {
	getenv := func(string) string { return "" }
	if Linked(nil, getenv) {
		t.Fatal("Linked with no args and empty env must be false")
	}
	if !Linked([]string{"--linked"}, getenv) {
		t.Fatal("Linked must detect the --linked flag")
	}
	if !Linked(nil, func(string) string { return "1" }) {
		t.Fatal("Linked must detect NEXUS_HUD_LINKED=1")
	}
	if !Linked(nil, func(string) string { return "TRUE" }) {
		t.Fatal("Linked must detect NEXUS_HUD_LINKED=true")
	}
	if Linked(nil, func(string) string { return "0" }) {
		t.Fatal("Linked must ignore NEXUS_HUD_LINKED=0")
	}
}

func TestBinaryPathIn(t *testing.T) {
	if _, ok := binaryPathIn(""); ok {
		t.Fatal("binaryPathIn must reject an empty directory")
	}
	if _, ok := binaryPathIn(t.TempDir()); ok {
		t.Fatal("binaryPathIn must reject a directory without nexus-hud")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, binaryName())
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, "nexus-hud.exe")
	}
	if err := os.WriteFile(path, []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, ok := binaryPathIn(dir)
	if !ok || got != path {
		t.Fatalf("binaryPathIn = (%q, %v), want (%q, true)", got, ok, path)
	}

	dirAsFile := t.TempDir()
	if err := os.Mkdir(filepath.Join(dirAsFile, binaryName()), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok := binaryPathIn(dirAsFile); ok {
		t.Fatal("binaryPathIn must reject a directory named like the binary")
	}
}

// TestBinaryPathForExeFindsSiblingMacOSBundle covers the shipped macOS layout,
// where the HUD lives in its own bundle next to the desktop app.
func TestBinaryPathForExeFindsSiblingMacOSBundle(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS bundle layout only")
	}
	parent := t.TempDir()
	desktopMacOS := filepath.Join(parent, "IAPro Nexus.app", "Contents", "MacOS")
	hudMacOS := filepath.Join(parent, hudBundleName, "Contents", "MacOS")
	for _, dir := range []string{desktopMacOS, hudMacOS} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	desktopExe := filepath.Join(desktopMacOS, "nexus-desktop")
	if err := os.WriteFile(desktopExe, []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}
	hudExe := filepath.Join(hudMacOS, binaryName())
	if err := os.WriteFile(hudExe, []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, ok := binaryPathForExe(desktopExe)
	if !ok || got != hudExe {
		t.Fatalf("binaryPathForExe = (%q, %v), want (%q, true)", got, ok, hudExe)
	}

	// A sibling flat layout still wins when both are present.
	flat := filepath.Join(desktopMacOS, binaryName())
	if err := os.WriteFile(flat, []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, ok = binaryPathForExe(desktopExe)
	if !ok || got != flat {
		t.Fatalf("binaryPathForExe with flat companion = (%q, %v), want (%q, true)", got, ok, flat)
	}

	// Without the companion bundle nothing is reported.
	if err := os.Remove(flat); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(hudExe); err != nil {
		t.Fatal(err)
	}
	if _, ok := binaryPathForExe(desktopExe); ok {
		t.Fatal("binaryPathForExe must not report a missing HUD")
	}
	if _, ok := binaryPathForExe(""); ok {
		t.Fatal("binaryPathForExe must reject an empty executable path")
	}
}

func TestLinkedCommand(t *testing.T) {
	cmd := linkedCommand(`C:\tools\nexus-hud.exe`)
	if len(cmd.Args) != 2 || cmd.Args[1] != "--linked" {
		t.Fatalf("linkedCommand args = %v, want [path --linked]", cmd.Args)
	}
	found := false
	for _, env := range cmd.Env {
		if env == LinkedEnv+"=1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("linkedCommand env must include %s=1", LinkedEnv)
	}
}

func TestSpawnDisabled(t *testing.T) {
	t.Setenv(SpawnEnv, "0")
	proc, err := Spawn()
	if err != nil {
		t.Fatalf("Spawn with companion disabled returned error: %v", err)
	}
	if proc != nil {
		proc.Close()
		t.Fatal("Spawn with companion disabled must return a nil process")
	}
}

func TestSpawnMissingBinary(t *testing.T) {
	if _, ok := BinaryPath(); ok && filepath.Base(mustExecutable(t)) == "nexus-hud.exe" {
		t.Skip("test runs from the HUD binary itself")
	}
	// The test binary never ships a companion next to it.
	if _, ok := BinaryPath(); ok {
		t.Skip("a nexus-hud binary exists next to the test executable")
	}
	proc, err := Spawn()
	if err != nil {
		t.Fatalf("Spawn without a companion binary returned error: %v", err)
	}
	if proc != nil {
		proc.Close()
		t.Fatal("Spawn without a companion binary must return a nil process")
	}
}

func mustExecutable(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(exe)
}
