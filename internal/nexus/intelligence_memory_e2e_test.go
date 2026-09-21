package nexus

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// E2E proof without OverMemory: producer writes a durable PROJECT fact,
// persistence survives reload, consumer receives/uses it. PROJECT wins over
// INSTALL on the same key; secret-like values are rejected.
func TestIntelligenceMemoryProjectE2EReloadPrecedenceAndSecret(t *testing.T) {
	root := t.TempDir()
	cfgDir := filepath.Join(root, "config")
	dataDir := filepath.Join(root, "data")
	stateDir := filepath.Join(root, "state")
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_STATE_DIR", stateDir)
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(dataDir, "nexus.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	producer := &Nexus{st: st, launcher: newMockLauncher()}
	ctx := context.Background()

	project, err := st.CreateProject(store.Project{Name: "Memory E2E", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	// INSTALL baseline (same key, different value).
	if _, err := producer.UpsertIntelligenceMemory(ctx, MemoryScopeInstall, "", "quality.verification", "make install-check", MemorySourceExplicit); err != nil {
		t.Fatalf("upsert install: %v", err)
	}
	// PROJECT producer writes durable fact (forced promotion path).
	if _, err := producer.UpsertIntelligenceMemory(ctx, MemoryScopeProject, project.ID, "quality.verification", "make quality", MemorySourceComposer); err != nil {
		t.Fatalf("upsert project: %v", err)
	}

	// Secret-like must be rejected, never persisted.
	if _, err := producer.UpsertIntelligenceMemory(ctx, MemoryScopeProject, project.ID, "quality.verification", "api_key=sk-secret-value", MemorySourceExplicit); err == nil {
		t.Fatalf("secret-like PROJECT value must be rejected")
	}
	if err := producer.PromoteHabitMemory(ctx, project.ID, "quality.testing", "token abc123", MemorySourceComposer, true); err != nil {
		t.Fatalf("promote secret should not error: %v", err)
	}

	// Persistence: close and reload from the same files.
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	st2, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	t.Cleanup(func() { _ = st2.Close() })
	consumer := &Nexus{st: st2, launcher: newMockLauncher()}

	facts, err := consumer.MemoryFactsMap(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Precedence PROJECT > INSTALL on the bare key.
	if facts["quality.verification"] != "make quality" {
		t.Fatalf("PROJECT must win over INSTALL, got %q facts=%#v", facts["quality.verification"], facts)
	}
	if facts["memory.project.quality.verification"] != "make quality" {
		t.Fatalf("project-scoped fact missing: %#v", facts)
	}
	if facts["memory.install.quality.verification"] != "make install-check" {
		t.Fatalf("install-scoped fact missing: %#v", facts)
	}
	// Secret-like must never appear in facts.
	for k, v := range facts {
		if LooksLikeSecret(v) {
			t.Fatalf("secret leaked into facts %q=%q", k, v)
		}
	}
	if v := facts["quality.testing"]; v == "token abc123" {
		t.Fatalf("secret promote leaked into facts")
	}

	// Recall is bounded and deterministic; PROJECT wins precedence without
	// hiding the INSTALL source from reflection/conflict diagnostics.
	recall, err := consumer.RecallIntelligenceMemories(ctx, MemoryRecallRequest{
		ProjectID: project.ID,
		Keys:      []string{"quality.verification"},
		Limit:     8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(recall.Items) != 2 {
		t.Fatalf("expected project + install recall rows, got %+v", recall.Items)
	}
	if recall.Items[0].Memory.Scope != MemoryScopeProject || recall.Items[0].Memory.Value != "make quality" {
		t.Fatalf("PROJECT recall must rank first, got %+v", recall.Items[0])
	}
	reflection, err := consumer.ReflectIntelligenceMemories(ctx, MemoryRecallRequest{
		ProjectID: project.ID,
		Keys:      []string{"quality.verification"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if reflection.Mode != "DETERMINISTIC" || len(reflection.Effective) != 1 {
		t.Fatalf("unexpected reflection: %+v", reflection)
	}
	if reflection.Effective[0].Memory.Value != "make quality" {
		t.Fatalf("reflection effective value=%q, want project value", reflection.Effective[0].Memory.Value)
	}
	if len(reflection.Conflicts) != 1 || reflection.Conflicts[0].Effective != MemoryScopeProject {
		t.Fatalf("expected explicit INSTALL/PROJECT conflict, got %+v", reflection.Conflicts)
	}
	beforeReflectFacts, err := consumer.MemoryFactsMap(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := consumer.ReflectIntelligenceMemories(ctx, MemoryRecallRequest{ProjectID: project.ID}); err != nil {
		t.Fatal(err)
	}
	afterReflectFacts, err := consumer.MemoryFactsMap(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(beforeReflectFacts) != len(afterReflectFacts) {
		t.Fatalf("reflection must not persist memory: before=%#v after=%#v", beforeReflectFacts, afterReflectFacts)
	}
	for key, value := range beforeReflectFacts {
		if afterReflectFacts[key] != value {
			t.Fatalf("reflection mutated memory fact %q: before=%q after=%q", key, value, afterReflectFacts[key])
		}
	}

	// Consumer uses the fact: Composer brief must not re-ask verification.
	brief := newComposerBrief("melhorar o dashboard", "")
	ApplyMemoriesToBrief(&brief, facts)
	refreshComposerBrief(&brief)
	for _, u := range brief.Unknowns {
		if u.Field == "quality.verification" && u.Status == PromptUnknownOpen {
			t.Fatalf("consumer must not re-ask verification covered by durable memory: %+v", u)
		}
	}
	found := false
	for _, v := range brief.Quality.Verification {
		if v == "make quality" {
			found = true
		}
	}
	if !found {
		t.Fatalf("consumer brief must carry project memory value, got %+v", brief.Quality.Verification)
	}
}
