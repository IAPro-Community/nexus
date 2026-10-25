package authcopy

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyValidatedFileKeepsSourceAndUsesPrivatePermissions(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "host", "auth.json")
	destination := filepath.Join(root, "profiles", "codex", "home", "auth.json")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	contents := []byte(`{"tokens":{"id_token":"signed","access_token":"access"}}`)
	if err := os.WriteFile(source, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	copied, err := CopyValidatedFile(source, destination, func(data []byte) bool {
		return string(data) == string(contents)
	})
	if err != nil || !copied {
		t.Fatalf("copy = %v, %v; want true, nil", copied, err)
	}
	got, err := os.ReadFile(destination)
	if err != nil || string(got) != string(contents) {
		t.Fatalf("destination content = %q, %v", got, err)
	}
	if got, err := os.ReadFile(source); err != nil || string(got) != string(contents) {
		t.Fatalf("source changed during import: %q, %v", got, err)
	}
	info, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("destination mode = %04o, want 0600", info.Mode().Perm())
	}
}

func TestCopyValidatedFileRejectsInvalidOrExistingCredentials(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "host.json")
	destination := filepath.Join(root, "profile", "auth.json")
	if err := os.WriteFile(source, []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	validate := func(data []byte) bool { return string(data) == "valid" }
	if copied, err := CopyValidatedFile(source, destination, validate); copied || err != nil {
		t.Fatalf("invalid import = %v, %v; want false, nil", copied, err)
	}
	if err := os.WriteFile(source, []byte("valid"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if copied, err := CopyValidatedFile(source, destination, validate); copied || !errors.Is(err, ErrDestinationExists) {
		t.Fatalf("existing import = %v, %v; want false, destination exists", copied, err)
	}
	got, err := os.ReadFile(destination)
	if err != nil || string(got) != "existing" {
		t.Fatalf("existing destination was changed: %q, %v", got, err)
	}
}

func TestCopyValidatedFileRejectsSymlinkSourceAndParent(t *testing.T) {
	root := t.TempDir()
	actual := filepath.Join(root, "actual.json")
	sourceLink := filepath.Join(root, "source.json")
	if err := os.WriteFile(actual, []byte("valid"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(actual, sourceLink); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	validate := func(data []byte) bool { return string(data) == "valid" }
	if copied, err := CopyValidatedFile(sourceLink, filepath.Join(root, "profile", "auth.json"), validate); copied || err == nil {
		t.Fatalf("symlink source import = %v, %v; want false and error", copied, err)
	}

	parent := filepath.Join(root, "linked")
	if err := os.Symlink(root, parent); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if copied, err := CopyValidatedFile(actual, filepath.Join(parent, "auth.json"), validate); copied || err == nil {
		t.Fatalf("symlink parent import = %v, %v; want false and error", copied, err)
	}
}
