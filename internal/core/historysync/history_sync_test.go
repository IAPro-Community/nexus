package historysync

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sdkprovider "github.com/IAPro-Community/IAPro-SDK/providers/spec"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
)

func TestSyncProviderHistoryMergesLocalDataAndIsolatesCredentials(t *testing.T) {
	hostHome := t.TempDir()
	dataDir := t.TempDir()
	t.Setenv("NEXUS_REAL_HOME", hostHome)
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())

	// Seed existing shared AGY data in host home.
	hostBrain := filepath.Join(hostHome, ".gemini", "antigravity-cli", "brain", "conv-1")
	if err := os.MkdirAll(hostBrain, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hostBrain, "notes.md"), []byte("from-host"), 0600); err != nil {
		t.Fatal(err)
	}

	// Seed profile omega-edu with local unlinked brain/conv-2, annotations, IDE conversations, stale cache, and its own OAuth token.
	profileHome := filepath.Join(dataDir, "profiles", "agy", "omega-edu", "home")
	localBrain := filepath.Join(profileHome, ".gemini", "antigravity-cli", "brain", "conv-2")
	if err := os.MkdirAll(localBrain, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localBrain, "notes.md"), []byte("from-omega"), 0600); err != nil {
		t.Fatal(err)
	}

	localAnnoDir := filepath.Join(profileHome, ".gemini", "antigravity-cli", "annotations")
	if err := os.MkdirAll(localAnnoDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localAnnoDir, "conv-1.pbtxt"), []byte(`title:"Shared Title"  plan_execute:true`), 0600); err != nil {
		t.Fatal(err)
	}

	hostIDESess := filepath.Join(hostHome, ".gemini", "antigravity", "conversations")
	if err := os.MkdirAll(hostIDESess, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hostIDESess, "ide-1.pb"), []byte("ide-data"), 0600); err != nil {
		t.Fatal(err)
	}

	jetboxCache := filepath.Join(profileHome, ".gemini", "antigravity-cli", "jetbox_summaries_proto.pb")
	if err := os.WriteFile(jetboxCache, []byte("stale-cache"), 0600); err != nil {
		t.Fatal(err)
	}
	staleTime := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(jetboxCache, staleTime, staleTime); err != nil {
		t.Fatal(err)
	}

	profileToken := filepath.Join(profileHome, ".gemini", "antigravity-cli", "antigravity-oauth-token")
	if err := os.WriteFile(profileToken, []byte(`{"token":"omega-secret"}`), 0600); err != nil {
		t.Fatal(err)
	}
	jetskiState := filepath.Join(profileHome, ".gemini", "antigravity-cli", "jetski_state.pbtxt")
	if err := os.WriteFile(jetskiState, []byte(`email:"omega@edu"`), 0600); err != nil {
		t.Fatal(err)
	}
	hostToken := filepath.Join(hostHome, ".gemini", "antigravity-cli", "antigravity-oauth-token")
	if err := os.WriteFile(hostToken, []byte(`{"token":"host-secret"}`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := SyncProviderHistory("agy", "omega-edu", profileHome); err != nil {
		t.Fatalf("SyncProviderHistory failed: %v", err)
	}

	// Both conv-1 (host) and conv-2 (merged from omega-edu) must exist through profile brain link.
	for _, conv := range []string{"conv-1", "conv-2"} {
		if _, err := os.Stat(filepath.Join(profileHome, ".gemini", "antigravity-cli", "brain", conv, "notes.md")); err != nil {
			t.Fatalf("missing synced conversation %s: %v", conv, err)
		}
	}
	// CLI annotations & IDE ("Other") conversations must both be synced.
	if _, err := os.Stat(filepath.Join(profileHome, ".gemini", "antigravity-cli", "annotations", "conv-1.pbtxt")); err != nil {
		t.Fatalf("missing synced annotations: %v", err)
	}
	if _, err := os.Stat(filepath.Join(profileHome, ".gemini", "antigravity", "conversations", "ide-1.pb")); err != nil {
		t.Fatalf("missing synced antigravity IDE conversations: %v", err)
	}
	// Stale jetbox_summaries_proto.pb must be invalidated when older than shared updates.
	if _, err := os.Stat(jetboxCache); !os.IsNotExist(err) {
		t.Fatalf("expected stale jetbox_summaries_proto.pb to be removed, stat err=%v", err)
	}
	// Credentials and jetski_state must stay strictly isolated per profile.
	gotToken, err := os.ReadFile(profileToken)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotToken) != `{"token":"omega-secret"}` {
		t.Fatalf("profile credential was overwritten: %q", string(gotToken))
	}
	gotJetski, err := os.ReadFile(jetskiState)
	if err != nil || string(gotJetski) != `email:"omega@edu"` {
		t.Fatalf("profile jetski_state was overwritten: %q (%v)", string(gotJetski), err)
	}
}

func TestMergeAnnotationPBTXTUnionsTitleAndFlags(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "host.pbtxt")
	src := filepath.Join(dir, "profile.pbtxt")

	if err := os.WriteFile(dst, []byte(`title:"Plan Evaluation And Continuation"`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte(`plan_execute:true`), 0600); err != nil {
		t.Fatal(err)
	}
	// Make profile.pbtxt newer; merge must STILL retain dst's title!
	older := time.Now().Add(-5 * time.Minute)
	newer := time.Now()
	if err := os.Chtimes(dst, older, older); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(src, newer, newer); err != nil {
		t.Fatal(err)
	}

	if err := mergePathInto(src, dst); err != nil {
		t.Fatalf("mergePathInto failed: %v", err)
	}
	raw, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	if !strings.Contains(got, `title:"Plan Evaluation And Continuation"`) {
		t.Fatalf("title lost after merge: %q", got)
	}
	if !strings.Contains(got, `plan_execute:true`) {
		t.Fatalf("plan_execute lost after merge: %q", got)
	}
}

func TestMergeAndLinkDefersActiveSQLiteWAL(t *testing.T) {
	root := t.TempDir()
	sharedDir := filepath.Join(root, "shared-conversations")
	profileDir := filepath.Join(root, "profile-conversations")
	if err := os.MkdirAll(sharedDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(profileDir, 0700); err != nil {
		t.Fatal(err)
	}
	dbFile := filepath.Join(profileDir, "active.db")
	walFile := filepath.Join(profileDir, "active.db-wal")
	if err := os.WriteFile(dbFile, []byte("sqlite-main"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(walFile, []byte("uncheckpointed-wal"), 0600); err != nil {
		t.Fatal(err)
	}

	err := MergeAndLinkSharedPath(sharedDir, profileDir)
	if !errors.Is(err, ErrActiveSQLiteSession) {
		t.Fatalf("expected ErrActiveSQLiteSession, got %v", err)
	}
	if _, statErr := os.Stat(dbFile); statErr != nil {
		t.Fatalf("local active.db must remain untouched when WAL is active: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(sharedDir, "active.db")); !os.IsNotExist(statErr) {
		t.Fatalf("active.db must not be migrated while WAL exists")
	}
}

func TestMergeClaudeProjectsInvalidatesConflictingSessionsIndex(t *testing.T) {
	dir := t.TempDir()
	sharedProj := filepath.Join(dir, "shared", "-projetos-nexus")
	profProj := filepath.Join(dir, "profile", "-projetos-nexus")
	if err := os.MkdirAll(sharedProj, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(profProj, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sharedProj, "sess-1.jsonl"), []byte(`{"id":"sess-1"}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sharedProj, "sessions-index.json"), []byte(`{"entries":[{"sessionId":"sess-1"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profProj, "sess-2.jsonl"), []byte(`{"id":"sess-2"}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profProj, "sessions-index.json"), []byte(`{"entries":[{"sessionId":"sess-2"}]}`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := MergeAndLinkSharedPath(sharedProj, profProj); err != nil {
		t.Fatalf("MergeAndLinkSharedPath failed: %v", err)
	}
	for _, id := range []string{"sess-1.jsonl", "sess-2.jsonl"} {
		if _, err := os.Stat(filepath.Join(sharedProj, id)); err != nil {
			t.Fatalf("missing merged transcript %s: %v", id, err)
		}
	}
	if _, err := os.Stat(filepath.Join(sharedProj, "sessions-index.json")); !os.IsNotExist(err) {
		t.Fatalf("expected conflicting sessions-index.json to be removed for rebuild, err=%v", err)
	}
}

func TestOpenCodeDBBlockedAsCredentialPath(t *testing.T) {
	oc, ok := spec.Get("opencode")
	if !ok {
		t.Fatal("opencode spec missing")
	}
	if oc.HistorySyncPolicy.Mode != spec.HistorySyncNone {
		t.Fatalf("opencode HistorySyncPolicy.Mode = %q, want %q", oc.HistorySyncPolicy.Mode, spec.HistorySyncNone)
	}
	if !isForbiddenCredentialPath("opencode.db", oc.HistorySyncPolicy.CredentialPaths) {
		t.Fatal("expected opencode.db to be protected as forbidden credential path")
	}
}

func TestSyncProviderHistoryBlocksCredentialOverlap(t *testing.T) {
	const testID = "bad-history-sync-test"
	defer spec.Unregister(testID)
	_ = sdkprovider.Register(sdkprovider.Spec{
		ID: testID,
		HistorySyncPolicy: sdkprovider.HistorySyncPolicy{
			Mode:            sdkprovider.HistorySyncLink,
			RootSubdir:      ".bad",
			SharedPaths:     []string{"auth.json"},
			CredentialPaths: []string{"auth.json"},
		},
	})
	if !isForbiddenCredentialPath("antigravity-cli/antigravity-oauth-token", []string{"antigravity-cli/antigravity-oauth-token"}) {
		t.Fatal("expected credential path guard to reject exact credential path")
	}
	if !isForbiddenCredentialPath("antigravity-cli", []string{"antigravity-cli/antigravity-oauth-token"}) {
		t.Fatal("expected credential path guard to reject parent dir containing credential")
	}
}
