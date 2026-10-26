package nexus

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestBoundedProjectIntelligenceContextGroundsComposerWithObservedFacts(t *testing.T) {
	n := openTestNexus(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Grounded", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := n.BoundedProjectIntelligenceContext(context.Background(), project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if envelope["scanner_version"] == "" || envelope["identity_digest"] == "" {
		t.Fatalf("missing snapshot identity envelope: %+v", envelope)
	}
	facts, ok := envelope["facts"].([]map[string]any)
	if !ok || len(facts) == 0 {
		t.Fatalf("expected bounded observed facts: %#v", envelope["facts"])
	}
	for _, fact := range facts {
		if fact["basis"] == "" || fact["confidence"] == "" || fact["source"] == "" || fact["observed_at"] == "" {
			t.Fatalf("fact lacks evidence metadata: %+v", fact)
		}
	}
	if _, leaksLocalPath := envelope["canonical_path"]; leaksLocalPath {
		t.Fatal("model context must not expose the local project path")
	}
}

func TestBoundedProjectIntelligenceContextCapsAndRedactsLargeFacts(t *testing.T) {
	n := openTestNexus(t)
	root := t.TempDir()
	packageJSON := `{"name":"fixture","scripts":{"test":"vitest ` + strings.Repeat("é", 40*1024) + ` OPENAI_API_KEY=sk-proj-1234567890abcdef1234567890abcdef"},"devDependencies":{"vitest":"3.0.0","react":"19.0.0"}}`
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(packageJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Bounded", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}

	envelope, err := n.BoundedProjectIntelligenceContext(context.Background(), project.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	const contextLimitBytes = 16 * 1024
	if len(raw) > contextLimitBytes {
		t.Fatalf("grounded context = %d bytes, limit = %d", len(raw), contextLimitBytes)
	}
	if strings.Contains(string(raw), "sk-proj-") {
		t.Fatalf("project intelligence leaked an API key: %s", raw)
	}
	if envelope["truncated"] != true {
		t.Fatalf("oversized project evidence must be marked truncated: %#v", envelope["truncated"])
	}
	if strings.Contains(string(raw), strings.Repeat("é", 257)) {
		t.Fatal("long manifest value was not truncated before entering model context")
	}
}

func TestBoundedProjectIntelligenceContextInvalidatesChangedRepositoryIdentity(t *testing.T) {
	n := openTestNexus(t)
	root := t.TempDir()
	packagePath := filepath.Join(root, "package.json")
	if err := os.WriteFile(packagePath, []byte(`{"scripts":{"test":"vitest run"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Freshness", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	before, err := n.BoundedProjectIntelligenceContext(context.Background(), project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(packagePath, []byte(`{"scripts":{"test":"bun run test:unit"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := n.BoundedProjectIntelligenceContext(context.Background(), project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if before["identity_digest"] == after["identity_digest"] {
		t.Fatalf("repository change did not invalidate grounded evidence: before=%#v after=%#v", before["identity_digest"], after["identity_digest"])
	}
	if before["identity_digest"] == "" || after["identity_digest"] == "" {
		t.Fatalf("grounded evidence must expose its freshness identity: before=%#v after=%#v", before["identity_digest"], after["identity_digest"])
	}
}
