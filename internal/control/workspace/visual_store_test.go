package workspace

import (
	"testing"
	"time"
)

func TestVisualStoreSaveAndGet(t *testing.T) {
	dir := t.TempDir()
	store := NewVisualStore(dir + "/visual.json")

	state := &WorkspaceVisualState{
		ProjectID:  "project-1",
		ActiveView: "view:agent-backend",
		Views: []ViewGeometry{
			{ViewID: "view:agent-backend", X: 0, Y: 0, Width: 800, Height: 600},
			{ViewID: "view:agent-frontend", X: 800, Y: 0, Width: 800, Height: 600},
		},
	}

	err := store.Save(state)
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	retrieved := store.Get("project-1")
	if retrieved == nil {
		t.Fatal("expected state to be retrieved")
	}
	if retrieved.ActiveView != "view:agent-backend" {
		t.Fatalf("expected active view view:agent-backend, got %s", retrieved.ActiveView)
	}
	if len(retrieved.Views) != 2 {
		t.Fatalf("expected 2 views, got %d", len(retrieved.Views))
	}
}

func TestVisualStoreDelete(t *testing.T) {
	dir := t.TempDir()
	store := NewVisualStore(dir + "/visual.json")

	state := &WorkspaceVisualState{
		ProjectID: "project-1",
		Views: []ViewGeometry{
			{ViewID: "view:1", X: 0, Y: 0, Width: 800, Height: 600},
		},
	}

	store.Save(state)
	store.Delete("project-1")

	retrieved := store.Get("project-1")
	if retrieved != nil {
		t.Fatal("expected state to be deleted")
	}
}

func TestVisualStoreList(t *testing.T) {
	dir := t.TempDir()
	store := NewVisualStore(dir + "/visual.json")

	store.Save(&WorkspaceVisualState{ProjectID: "project-1"})
	store.Save(&WorkspaceVisualState{ProjectID: "project-2"})

	list := store.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 states, got %d", len(list))
	}
}

func TestVisualStorePersistence(t *testing.T) {
	dir := t.TempDir()
	filePath := dir + "/visual.json"

	store1 := NewVisualStore(filePath)
	store1.Save(&WorkspaceVisualState{
		ProjectID:  "project-1",
		ActiveView: "view:1",
		Views:      []ViewGeometry{{ViewID: "view:1", X: 10, Y: 20, Width: 800, Height: 600}},
	})

	// Create new store from same file
	store2 := NewVisualStore(filePath)
	retrieved := store2.Get("project-1")
	if retrieved == nil {
		t.Fatal("expected state to persist across store instances")
	}
	if retrieved.ActiveView != "view:1" {
		t.Fatalf("expected active view view:1, got %s", retrieved.ActiveView)
	}
	if len(retrieved.Views) != 1 || retrieved.Views[0].X != 10 {
		t.Fatalf("expected view geometry to persist, got %v", retrieved.Views)
	}
}

func TestVisualStoreUpdatedAt(t *testing.T) {
	dir := t.TempDir()
	store := NewVisualStore(dir + "/visual.json")

	before := time.Now()
	store.Save(&WorkspaceVisualState{ProjectID: "project-1"})
	after := time.Now()

	state := store.Get("project-1")
	if state.UpdatedAt.Before(before) || state.UpdatedAt.After(after) {
		t.Fatalf("expected UpdatedAt between %v and %v, got %v", before, after, state.UpdatedAt)
	}
}
