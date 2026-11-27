package doctor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPruneProfileStorageStrictAllowlistAndSymlinkSafety(t *testing.T) {
	dataDir := t.TempDir()
	hostSecretDir := t.TempDir()
	hostSecretFile := filepath.Join(hostSecretDir, "auth.json")
	if err := os.WriteFile(hostSecretFile, []byte(`{"token":"keep-me"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	// 1. Valid profile with symlinked .codex + allowlisted + non-allowlisted caches + temp leak.
	profRoot := filepath.Join(dataDir, "profiles", "agy", "work")
	profHome := filepath.Join(profRoot, "home")
	goBuildCache := filepath.Join(profHome, ".cache", "go-build")
	agyCache := filepath.Join(profHome, ".cache", "antigravity")
	yarnShare := filepath.Join(profHome, ".local", "share", "yarn")
	for _, d := range []string{goBuildCache, agyCache, yarnShare} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(goBuildCache, "obj.a"), make([]byte, 4096), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agyCache, "state.json"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(yarnShare, "pkg.tgz"), make([]byte, 2048), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(hostSecretDir, filepath.Join(profHome, ".codex")); err != nil {
		t.Fatal(err)
	}

	tempScope := filepath.Join(profRoot, "account-scope.tmp.12345")
	if err := os.WriteFile(tempScope, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	_ = os.Chtimes(tempScope, old, old)

	// 2. Unreferenced ghost profile failing ValidateProfileName.
	ghostDir := filepath.Join(dataDir, "profiles", "cursor", "--with")
	if err := os.MkdirAll(filepath.Join(ghostDir, "home"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ghostDir, "home", "dummy"), make([]byte, 1024), 0o600); err != nil {
		t.Fatal(err)
	}

	// 3. Stale dead-PID listen file in web-auth.
	webAuth := filepath.Join(dataDir, "web-auth")
	if err := os.MkdirAll(webAuth, 0o700); err != nil {
		t.Fatal(err)
	}
	staleListen := filepath.Join(webAuth, "listen-999999.json")
	if err := os.WriteFile(staleListen, []byte(`{"pid":999999}`), 0o600); err != nil {
		t.Fatal(err)
	}

	dry, err := InspectProfileStorage(dataDir)
	if err != nil {
		t.Fatalf("InspectProfileStorage: %v", err)
	}
	if dry.ReclaimableBytes < 4096+2048+1024 {
		t.Fatalf("ReclaimableBytes = %d, expected >= %d", dry.ReclaimableBytes, 4096+2048+1024)
	}
	if _, err := os.Stat(goBuildCache); err != nil {
		t.Fatalf("dry-run must not delete go-build: %v", err)
	}

	applied, err := PruneProfileStorage(dataDir, PruneOptions{
		DryRun:               false,
		IncludeGhostProfiles: true,
		MinTempAge:           time.Hour,
	})
	if err != nil {
		t.Fatalf("PruneProfileStorage: %v", err)
	}
	if applied.FreedBytes == 0 || applied.StaleWebAuthRemoved != 1 {
		t.Fatalf("unexpected applied stats: %+v", applied)
	}

	// Allowlisted caches, temp leak, ghost dir, and dead listen file must be gone.
	for _, gone := range []string{goBuildCache, yarnShare, tempScope, ghostDir, staleListen} {
		if _, err := os.Lstat(gone); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be pruned, lstat err=%v", gone, err)
		}
	}

	// Non-allowlisted caches and symlink targets MUST remain completely intact!
	if _, err := os.Stat(filepath.Join(agyCache, "state.json")); err != nil {
		t.Fatalf("non-allowlisted .cache/antigravity must never be deleted: %v", err)
	}
	if _, err := os.Stat(hostSecretFile); err != nil {
		t.Fatalf("host symlink target must NEVER be modified or deleted: %v", err)
	}
}
