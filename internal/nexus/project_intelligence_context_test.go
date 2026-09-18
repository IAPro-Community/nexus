package nexus

import (
	"context"
	"os"
	"path/filepath"
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
	if envelope["scanner_version"] == "" || envelope["identity"] == nil {
		t.Fatalf("missing snapshot identity envelope: %+v", envelope)
	}
	facts, ok := envelope["facts"].([]map[string]any)
	if !ok || len(facts) == 0 {
		t.Fatalf("expected bounded observed facts: %#v", envelope["facts"])
	}
	for _, fact := range facts {
		if fact["basis"] == "" || fact["confidence"] == "" || fact["source"] == "" {
			t.Fatalf("fact lacks evidence metadata: %+v", fact)
		}
	}
}
