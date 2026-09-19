package device

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInstallationIDIsStableThenResettable(t *testing.T) {
	dir := t.TempDir()
	first, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || first == "" {
		t.Fatalf("id must persist across loads: %q vs %q", first, second)
	}
	info, err := os.Stat(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	// Unix permission bits are not enforced by the Windows ACL model: Go
	// reports 0666 for 0600-created files there. The restrictive intent on
	// Windows comes from the per-user profile directory ACLs, so exact-bit
	// assertion is Unix-only.
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("identity file perms=%o, want 600", info.Mode().Perm())
	}
	if !info.Mode().IsRegular() {
		t.Fatal("identity file must be a regular file")
	}
	if err := Reset(dir); err != nil {
		t.Fatal(err)
	}
	third, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if third == first {
		t.Fatal("reset must mint an unrelated identity")
	}
}

func TestInstallationIDRejectsEmptyDir(t *testing.T) {
	if _, err := LoadOrCreate(""); err == nil {
		t.Fatal("empty data dir must fail")
	}
}
