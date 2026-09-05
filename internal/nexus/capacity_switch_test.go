package nexus

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

func TestCapacitySwitchRequiresCurrentFingerprintAndConfirmation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nexus_switch_test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	n := &Nexus{st: st}
	ctx := context.Background()

	// 1. Stale fingerprint or unconfirmed switch must fail
	_, err = n.ConfirmCapacitySwitch(ctx, "agent-1", "generation-2", "", "gemini", "work")
	if err == nil {
		t.Fatal("empty or unconfirmed fingerprint must fail")
	}

	// 2. Non-existent agent must fail
	_, err = n.ConfirmCapacitySwitch(ctx, "non-existent-agent", "gen-1", "fp-123", "gemini", "work")
	if err == nil {
		t.Fatal("non-existent agent must fail")
	}
}
