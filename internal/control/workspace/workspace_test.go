package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceStore_AddListRemove(t *testing.T) {
	tempDir := t.TempDir()
	storeFile := filepath.Join(tempDir, "projects.json")

	store := NewStore(storeFile)

	// Create test directories
	proj1 := filepath.Join(tempDir, "proj-alpha")
	proj2 := filepath.Join(tempDir, "proj-beta")
	_ = os.MkdirAll(proj1, 0755)
	_ = os.MkdirAll(proj2, 0755)

	p1, err := store.Add(proj1, "Alpha Project")
	if err != nil {
		t.Fatalf("failed to add proj1: %v", err)
	}
	if p1.Name != "Alpha Project" {
		t.Errorf("expected Alpha Project, got %s", p1.Name)
	}

	p2, err := store.Add(proj2, "")
	if err != nil {
		t.Fatalf("failed to add proj2: %v", err)
	}
	if p2.Name != "proj-beta" {
		t.Errorf("expected proj-beta default name, got %s", p2.Name)
	}

	list := store.List()
	if len(list) < 2 {
		t.Fatalf("expected at least 2 projects, got %d", len(list))
	}

	// Remove proj1
	if err := store.Remove(proj1); err != nil {
		t.Fatalf("failed to remove proj1: %v", err)
	}

	// Verify reload from disk
	storeReloaded := NewStore(storeFile)
	list2 := storeReloaded.List()
	for _, p := range list2 {
		if p.Path == proj1 {
			t.Errorf("proj1 should have been removed from disk persistence")
		}
	}
}

func TestWorkspaceIDDistinctByPath(t *testing.T) {
	if makeWorkspaceID("/home/user/company/api") == makeWorkspaceID("/home/user/personal/api") {
		t.Error("workspace IDs must differ for different canonical paths sharing a basename")
	}
	if !strings.HasPrefix(makeWorkspaceID("/tmp/x/y"), "ws-") {
		t.Error("workspace ID must carry the ws- prefix")
	}
}

func TestWorkspaceListSortedByRecency(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "projects.json"))
	a := filepath.Join(t.TempDir(), "aaa")
	b := filepath.Join(t.TempDir(), "bbb")
	_ = os.MkdirAll(a, 0755)
	_ = os.MkdirAll(b, 0755)

	if _, err := store.Add(a, "a"); err != nil {
		t.Fatalf("failed to add a: %v", err)
	}
	if _, err := store.Add(b, "b"); err != nil {
		t.Fatalf("failed to add b: %v", err)
	}
	store.Touch(a)

	list := store.List()
	if len(list) < 2 {
		t.Fatalf("expected at least 2 projects, got %d", len(list))
	}
	if !workspacePathsEquivalent(list[0].Path, a) {
		t.Errorf("expected most recently used project first, got %s (want equivalent to %s)", list[0].Path, a)
	}
	// Deterministic ordering: two equal timestamps tie-break by ID.
	if list[0].LastUsedAt.Before(list[1].LastUsedAt) {
		t.Error("list must be ordered by LastUsedAt DESC")
	}
}

func TestWorkspaceStoreRemoveResolvesFilesystemAliases(t *testing.T) {
	root := t.TempDir()
	realPath := filepath.Join(root, "real")
	aliasPath := filepath.Join(root, "alias")
	if err := os.Mkdir(realPath, 0755); err != nil {
		t.Fatalf("create real workspace: %v", err)
	}
	if err := os.Symlink(realPath, aliasPath); err != nil {
		t.Fatalf("create workspace alias: %v", err)
	}

	store := NewStore(filepath.Join(root, "projects.json"))
	if _, err := store.Add(realPath, "real"); err != nil {
		t.Fatalf("add workspace: %v", err)
	}
	if err := store.Remove(aliasPath); err != nil {
		t.Fatalf("remove workspace through filesystem alias: %v", err)
	}
	if len(store.List()) != 1 {
		t.Fatalf("expected only the automatically discovered cwd workspace after removal, got %d", len(store.List()))
	}
}
