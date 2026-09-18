package session

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestSessionAggregationAndSearch(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("AI_CLI_STATE_DIR", stateDir)

	store := NewStore()

	now := time.Now()
	sessions := []model.Session{
		{
			ProviderID: "codex",
			ProfileID:  "work",
			ID:         "sess-1",
			Title:      "Fix Kubernetes DNS issue",
			Workspace:  "/projects/infra",
			UpdatedAt:  now.Add(-10 * time.Minute),
		},
		{
			ProviderID: "agy",
			ProfileID:  "personal",
			ID:         "sess-2",
			Title:      "Refactor Auth Router",
			Workspace:  "/projects/web",
			UpdatedAt:  now.Add(-5 * time.Minute),
		},
	}

	// Test Pinning
	if err := store.PinSession("codex", "sess-1", true); err != nil {
		t.Fatal(err)
	}

	aggregated := store.Aggregate(sessions, "/projects/web")
	if len(aggregated) != 2 {
		t.Fatalf("expected 2 aggregated sessions, got %d", len(aggregated))
	}
	// Pinned session should come first
	if aggregated[0].ID != "sess-1" || !aggregated[0].Pinned {
		t.Fatalf("expected pinned sess-1 to be first, got %+v", aggregated[0])
	}

	// Test Search
	matches := store.Search(aggregated, "Kubernetes")
	if len(matches) != 1 || matches[0].ID != "sess-1" {
		t.Fatalf("expected search match for Kubernetes, got %+v", matches)
	}

	// Test Workspace Grouping
	cfg := config.NewDefaultConfig()
	cfg.Bindings["/projects/infra"] = map[string]string{"codex": "work"}

	wsList := store.GroupByWorkspace(aggregated, cfg)
	if len(wsList) != 2 {
		t.Fatalf("expected 2 workspace groups, got %d", len(wsList))
	}
}
