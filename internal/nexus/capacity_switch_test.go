package nexus

import (
	"context"
	"path/filepath"
	"testing"
	"time"

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

func TestCapacitySwitchRequiresPersistedMatchingRecommendation(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "nexus_switch_recommendation.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	project, err := st.CreateProject(store.Project{ID: "project-1", Name: "test", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	agent, err := st.CreateAgent(store.Agent{ID: "agent-1", ProjectID: project.ID, Name: "agent"})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := st.PutCapacityRecommendation(store.CapacityRecommendationRecord{
		Fingerprint: "fp-current", Provider: "gemini", Profile: "work", CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("put recommendation: %v", err)
	}
	n := &Nexus{st: st}
	ctx := context.Background()

	if _, err := n.ConfirmCapacitySwitch(ctx, agent.ID, "", "fp-missing", "gemini", "work"); err == nil {
		t.Fatal("missing persisted recommendation must fail")
	}
	if _, err := n.ConfirmCapacitySwitch(ctx, agent.ID, "", "fp-current", "anthropic", "work"); err == nil {
		t.Fatal("provider mismatch must fail")
	}
	if _, err := n.ConfirmCapacitySwitch(ctx, agent.ID, "", "fp-current", "gemini", "other"); err == nil {
		t.Fatal("profile mismatch must fail")
	}
}

func TestCapacitySwitchRejectsGenerationLookupErrors(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "nexus_switch_generation.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	project, err := st.CreateProject(store.Project{ID: "project-1", Name: "test", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	agent, err := st.CreateAgent(store.Agent{ID: "agent-1", ProjectID: project.ID, Name: "agent"})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := st.PutCapacityRecommendation(store.CapacityRecommendationRecord{
		Fingerprint: "fp-current", Provider: "gemini", Profile: "work", CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("put recommendation: %v", err)
	}
	n := &Nexus{st: st}
	if _, err := n.ConfirmCapacitySwitch(context.Background(), agent.ID, "stale-generation", "fp-current", "gemini", "work"); err == nil {
		t.Fatal("missing current generation must not be ignored")
	}
}
