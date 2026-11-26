package historysync

import (
	"os"
	"path/filepath"
	"testing"

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

	// Seed profile omega-edu with local unlinked brain/conv-2 and its own OAuth token.
	profileHome := filepath.Join(dataDir, "profiles", "agy", "omega-edu", "home")
	localBrain := filepath.Join(profileHome, ".gemini", "antigravity-cli", "brain", "conv-2")
	if err := os.MkdirAll(localBrain, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localBrain, "notes.md"), []byte("from-omega"), 0600); err != nil {
		t.Fatal(err)
	}
	profileToken := filepath.Join(profileHome, ".gemini", "antigravity-cli", "antigravity-oauth-token")
	if err := os.WriteFile(profileToken, []byte(`{"token":"omega-secret"}`), 0600); err != nil {
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
	// Credentials must stay strictly isolated per profile.
	gotToken, err := os.ReadFile(profileToken)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotToken) != `{"token":"omega-secret"}` {
		t.Fatalf("profile credential was overwritten: %q", string(gotToken))
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
	// Note: sdkprovider.Register itself validates CredentialPaths overlap;
	// verify guard directly as well.
	if !isForbiddenCredentialPath("antigravity-cli/antigravity-oauth-token", []string{"antigravity-cli/antigravity-oauth-token"}) {
		t.Fatal("expected credential path guard to reject exact credential path")
	}
	if !isForbiddenCredentialPath("antigravity-cli", []string{"antigravity-cli/antigravity-oauth-token"}) {
		t.Fatal("expected credential path guard to reject parent dir containing credential")
	}
}
