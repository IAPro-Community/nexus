package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IAPro-Community/nexus/internal/testutil"
)

func TestCanonicalWorkspacePath(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "real_sub")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subDir: %v", err)
	}

	symlinkDir := filepath.Join(tempDir, "symlink_sub")
	testutil.SymlinkOrSkip(t, subDir, symlinkDir)

	// CanonicalWorkspacePath should resolve symlink to real_sub
	canonicalFromReal, err := CanonicalWorkspacePath(subDir)
	if err != nil {
		t.Fatalf("CanonicalWorkspacePath(subDir) failed: %v", err)
	}

	canonicalFromSymlink, err := CanonicalWorkspacePath(symlinkDir)
	if err != nil {
		t.Fatalf("CanonicalWorkspacePath(symlinkDir) failed: %v", err)
	}

	if canonicalFromReal != canonicalFromSymlink {
		t.Errorf("expected canonical paths to be identical, got %q vs %q", canonicalFromReal, canonicalFromSymlink)
	}

	// Test CanonicalExistingWorkspaceDir
	checkedDir, err := CanonicalExistingWorkspaceDir(symlinkDir)
	if err != nil {
		t.Fatalf("CanonicalExistingWorkspaceDir failed: %v", err)
	}
	if checkedDir != canonicalFromReal {
		t.Errorf("expected checked dir %q, got %q", canonicalFromReal, checkedDir)
	}
}

func TestResolvePathRefUsesDisplayCanonicalAndFilesystemIdentity(t *testing.T) {
	dir := t.TempDir()
	ref, err := ResolvePathRef(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ref.DisplayPath != dir || ref.CanonicalPath == "" {
		t.Fatalf("unexpected path ref: %+v", ref)
	}
	if !ref.Identity.Available || ref.Identity.StableKey == "" {
		t.Fatalf("existing directory must have filesystem identity: %+v", ref.Identity)
	}
}
