package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// openTestServiceNexus builds a Nexus whose store lives in a temp data dir and
// whose process-global instance is released on cleanup.
func openTestServiceNexus(t *testing.T) *nexus.Nexus {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dir)
	t.Setenv("NEXUS_REAL_HOME", dir)
	st, err := store.Open(filepath.Join(dir, "nexus.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return nexus.NewWithStoreAndExecutor(st, nil)
}

// `nexus <harness>` runs from any directory. Registering that directory as a
// project implicitly is what accumulated stale project rows, so the supervised
// launch path may only reuse an existing registration.
func TestPrepareInteractiveLeadRefusesUnregisteredWorkspace(t *testing.T) {
	n := openTestServiceNexus(t)
	workspace := t.TempDir()

	_, err := prepareInteractiveLead(n, workspace, "codex", "")
	if err == nil {
		t.Fatal("an unregistered working directory must not become a project")
	}
	if !strings.Contains(err.Error(), "not a registered Nexus project") {
		t.Fatalf("error must explain how to register the project, got %v", err)
	}
	if !strings.Contains(err.Error(), "nexus projects add") {
		t.Fatalf("error must name the explicit registration command, got %v", err)
	}

	st, openErr := n.OpenProject()
	if openErr != nil {
		t.Fatal(openErr)
	}
	projects, listErr := st.ListProjects()
	if listErr != nil {
		t.Fatal(listErr)
	}
	for _, project := range projects {
		if strings.HasPrefix(project.CanonicalPath, workspace) {
			t.Fatalf("launch registered %s implicitly", project.CanonicalPath)
		}
	}
}

func TestProjectsPruneCommandRefusesUnknownFlags(t *testing.T) {
	err := projectsPruneCmd([]string{"--delete-everything"})
	if err == nil || !strings.Contains(err.Error(), "--apply") {
		t.Fatalf("unknown flags must be rejected with the supported options, got %v", err)
	}
}
