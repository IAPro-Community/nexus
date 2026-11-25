package nexus

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestModelCatalogWatchDiffsPersistenceAndFreshness(t *testing.T) {
	watchPath := filepath.Join(t.TempDir(), "nex-model-catalog-watch.json")
	svc := NewModelCatalogWatchService(watchPath)
	t0 := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

	initialSnap := svc.Snapshot(t0)
	if initialSnap.Freshness != ModelWatchFreshnessUnknown {
		t.Fatalf("expected UNKNOWN before first observation, got %s", initialSnap.Freshness)
	}

	entries1 := []ModelWatchEntry{
		{Provider: "claude", Profile: "work", ModelID: "claude-sonnet-4", IsDefault: true, Source: "provider_snapshot"},
		{Provider: "claude", Profile: "work", ModelID: "claude-opus-4", Source: "provider_snapshot"},
	}
	changes1 := svc.ObserveEntries(entries1, t0)
	if len(changes1) != 2 || changes1[0].Kind != ModelChangeAdded || changes1[1].Kind != ModelChangeAdded {
		t.Fatalf("expected 2 ADDED changes, got %+v", changes1)
	}

	// Re-observing identical entries must emit zero changes.
	dup := svc.ObserveEntries(entries1, t0.Add(time.Minute))
	if len(dup) != 0 {
		t.Fatalf("expected 0 duplicate changes on identical observation, got %+v", dup)
	}

	// Deprecate opus, remove sonnet, add haiku as new default.
	entries2 := []ModelWatchEntry{
		{Provider: "claude", Profile: "work", ModelID: "claude-opus-4", Deprecated: true, Source: "provider_snapshot"},
		{Provider: "claude", Profile: "work", ModelID: "claude-haiku-4", IsDefault: true, Source: "provider_snapshot"},
	}
	changes2 := svc.ObserveEntries(entries2, t0.Add(2*time.Minute))
	kinds := map[string]bool{}
	for _, ch := range changes2 {
		kinds[ch.Kind] = true
	}
	for _, expected := range []string{ModelChangeAdded, ModelChangeRemoved, ModelChangeDeprecated, ModelChangeDefaultChanged} {
		if !kinds[expected] {
			t.Fatalf("expected change kind %s in %+v", expected, changes2)
		}
	}

	// Fresh within 10m, STALE after 10m.
	if got := svc.Snapshot(t0.Add(5 * time.Minute)).Freshness; got != ModelWatchFreshnessFresh {
		t.Fatalf("expected FRESH snapshot, got %s", got)
	}
	if got := svc.Snapshot(t0.Add(15 * time.Minute)).Freshness; got != ModelWatchFreshnessStale {
		t.Fatalf("expected STALE snapshot after 10m, got %s", got)
	}

	// Reload from disk preserves entries and change history.
	reloaded := NewModelCatalogWatchService(watchPath)
	snap := reloaded.Snapshot(t0.Add(3 * time.Minute))
	if snap.Freshness != ModelWatchFreshnessFresh || len(snap.Entries) != 2 || len(snap.Changes) != 6 {
		t.Fatalf("unexpected reloaded snapshot: %+v", snap)
	}

	// Concurrent reads/observations stay race-free.
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = reloaded.ObserveEntries(entries2, t0.Add(time.Duration(idx+3)*time.Minute))
			_ = reloaded.Snapshot(t0.Add(time.Duration(idx+4) * time.Minute))
		}(i)
	}
	wg.Wait()
}
