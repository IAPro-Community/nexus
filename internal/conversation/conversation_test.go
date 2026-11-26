package conversation

import (
	"context"
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

func TestListRecentIncludesCursorProfileChats(t *testing.T) {
	hostHome, dataDir := t.TempDir(), t.TempDir()
	t.Setenv("NEXUS_REAL_HOME", hostHome)
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)
	t.Setenv("USERPROFILE", hostHome)
	t.Setenv("APPDATA", filepath.Join(hostHome, "AppData", "Roaming"))
	t.Setenv("LOCALAPPDATA", filepath.Join(hostHome, "AppData", "Local"))
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("NEXUS_DATA_DIR", dataDir)

	profileRoot := filepath.Join(dataDir, "profiles", "cursor", "cursor2")
	if err := os.MkdirAll(profileRoot, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileRoot, "profile.json"), []byte(`{"provider":"cursor","name":"cursor2"}`), 0600); err != nil {
		t.Fatal(err)
	}
	chatID := "f8ebf955-eda8-482d-bbda-cefee4bfdb32"
	chatDir := filepath.Join(profileRoot, "home", ".cursor", "chats", "workspace-hash", chatID)
	if err := os.MkdirAll(chatDir, 0700); err != nil {
		t.Fatal(err)
	}
	meta := `{"hasConversation":true,"title":"Cursor profile chat","cwd":"/workspace/cursor","createdAtMs":1790000000000,"updatedAtMs":1790000001000}`
	if err := os.WriteFile(filepath.Join(chatDir, "meta.json"), []byte(meta), 0600); err != nil {
		t.Fatal(err)
	}

	conversations := ListRecent(10, "/workspace/cursor")
	for _, item := range conversations {
		if item.Provider == "cursor" && item.ID == chatID && item.Profile == "cursor2" {
			return
		}
	}
	t.Fatalf("Cursor profile conversation missing from universal history: %+v", conversations)
}

func TestListOtherReadOnlyKeepsLegacyProfilesUntouched(t *testing.T) {
	data, host := t.TempDir(), t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("NEXUS_REAL_HOME", host)
	t.Setenv("AI_REAL_HOME", host)
	t.Setenv("HOME", host)
	t.Setenv("USERPROFILE", host)
	t.Setenv("APPDATA", filepath.Join(host, "AppData", "Roaming"))
	t.Setenv("LOCALAPPDATA", filepath.Join(host, "AppData", "Local"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(host, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(host, ".local", "share"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(host, ".cache"))
	legacy := filepath.Join(data, "profiles", "agy", "legacy", "home")
	if err := os.MkdirAll(legacy, 0700); err != nil {
		t.Fatal(err)
	}
	history := filepath.Join(host, ".gemini", "antigravity-cli", "history.jsonl")
	if err := os.MkdirAll(filepath.Dir(history), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(history, []byte(`{"display":"Other provider","timestamp":1787839787154,"workspace":"/elsewhere","conversationId":"other-1"}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cursorProfile := filepath.Join(data, "profiles", "cursor", "cursor2")
	if err := os.MkdirAll(cursorProfile, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cursorProfile, "profile.json"), []byte(`{"provider":"cursor","name":"cursor2"}`), 0600); err != nil {
		t.Fatal(err)
	}
	cursorChat := filepath.Join(cursorProfile, "home", ".cursor", "chats", "workspace-hash", "cursor-chat")
	if err := os.MkdirAll(cursorChat, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cursorChat, "meta.json"), []byte(`{"hasConversation":true,"title":"Cursor history","cwd":"/workspace/cursor"}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := ListOtherReadOnly(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("other history missing: %+v", got)
	}
	foundCursor := false
	for _, conversation := range got {
		if conversation.Provider == "cursor" && conversation.ID == "cursor-chat" && conversation.Profile == "cursor2" {
			foundCursor = true
		}
	}
	if !foundCursor {
		t.Fatalf("Cursor session missing from read-only history picker: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(legacy), "profile.json")); !os.IsNotExist(err) {
		t.Fatal("discovery changed legacy profile")
	}
}
