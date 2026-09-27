package conversation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListRecentConversations(t *testing.T) {
	hostHome := t.TempDir()
	t.Setenv("NEXUS_REAL_HOME", hostHome)
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)
	t.Setenv("USERPROFILE", hostHome)
	t.Setenv("APPDATA", filepath.Join(hostHome, "AppData", "Roaming"))
	t.Setenv("LOCALAPPDATA", filepath.Join(hostHome, "AppData", "Local"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(hostHome, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(hostHome, ".local", "share"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(hostHome, ".cache"))
	t.Setenv("AI_CLI_DATA_DIR", filepath.Join(hostHome, "nexus-data"))
	t.Setenv("AI_MANAGER_DATA_DIR", filepath.Join(hostHome, "nexus-data"))
	t.Setenv("NEXUS_DATA_DIR", filepath.Join(hostHome, "nexus-data"))

	// Create simulated AGY history.jsonl
	agyCliDir := filepath.Join(hostHome, ".gemini", "antigravity-cli")
	_ = os.MkdirAll(agyCliDir, 0755)
	histData := `{"display":"Refactor auth system","timestamp":1787839787154,"workspace":"/projetos/my-app","conversationId":"conv-agy-1"}
{"display":"Fix lint error","timestamp":1787849787154,"workspace":"/projetos/my-app","conversationId":"conv-agy-2"}
`
	_ = os.WriteFile(filepath.Join(agyCliDir, "history.jsonl"), []byte(histData), 0644)

	// Create simulated Codex session_index.jsonl
	codexDir := filepath.Join(hostHome, ".codex")
	_ = os.MkdirAll(codexDir, 0755)
	indexData := `{"id":"conv-codex-1","thread_name":"Codex Thread Test","updated_at":"2026-08-27T10:00:00Z"}` + "\n"
	_ = os.WriteFile(filepath.Join(codexDir, "session_index.jsonl"), []byte(indexData), 0644)

	convs := ListRecent(10, "/projetos/my-app")
	if len(convs) < 3 {
		t.Fatalf("expected at least 3 conversations, got %d", len(convs))
	}

	foundAgy2 := false
	foundCodex := false
	for _, c := range convs {
		if c.ID == "conv-agy-2" && c.Provider == "agy" {
			foundAgy2 = true
		}
		if c.ID == "conv-codex-1" && c.Title == "Codex Thread Test" && c.Provider == "codex" {
			foundCodex = true
		}
	}

	if !foundAgy2 {
		t.Errorf("conv-agy-2 not found: %+v", convs)
	}
	if !foundCodex {
		t.Errorf("conv-codex-1 not found: %+v", convs)
	}
}
