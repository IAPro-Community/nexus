package nexus

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNexPersonalMemoryCRUDRedactionAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nex-personal-memory.json")
	st := NewNexPersonalMemoryStore(path)
	now := time.Date(2026, 11, 25, 14, 0, 0, 0, time.UTC)

	if _, err := st.Upsert("", "value", "user_explicit", now); err == nil {
		t.Fatal("expected empty key rejection")
	}

	secret := "sk-proj-1234567890abcdefghijklmnopqrstuvwxyz"
	entry, err := st.Upsert("preferred_lang", "pt-BR token="+secret, "user_explicit", now)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	if entry.Scope != "@personal" || strings.Contains(entry.Value, secret) || !strings.Contains(entry.Value, "REDACTED") {
		t.Fatalf("expected @personal scope and redacted secret, got %+v", entry)
	}

	updated, err := st.Upsert("preferred_lang", "en-US concise", "assistant_approved", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("Upsert update failed: %v", err)
	}
	if updated.ID != entry.ID || updated.Value != "en-US concise" || updated.Provenance != "assistant_approved" {
		t.Fatalf("expected in-place key update, got %+v", updated)
	}

	// Reload from disk.
	reloaded := NewNexPersonalMemoryStore(path)
	list := reloaded.List()
	if len(list) != 1 || list[0].Value != "en-US concise" {
		t.Fatalf("unexpected reloaded entries: %+v", list)
	}
	if sum := reloaded.BoundedSummary(32); !strings.Contains(sum, "preferred_lang") {
		t.Fatalf("unexpected summary: %q", sum)
	}

	if !reloaded.Delete(entry.ID) {
		t.Fatal("expected Delete to remove existing entry")
	}
	if len(reloaded.List()) != 0 {
		t.Fatal("expected empty list after delete")
	}
}
