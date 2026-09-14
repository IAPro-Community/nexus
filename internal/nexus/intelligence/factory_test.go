package intelligence

import (
	"os"
	"path/filepath"
	"testing"

	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
)

func TestProviderFromConfigOffReturnsNil(t *testing.T) {
	p, err := ProviderFromConfig(coreconfig.IntelligenceConfig{Mode: coreconfig.IntelligenceOff})
	if err != nil || p != nil {
		t.Fatalf("expected nil provider without error, p=%v err=%v", p, err)
	}
}

func TestResolveAPIKeyReadsFileWithoutPersistingSecret(t *testing.T) {
	path := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(path, []byte("secret-value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := resolveAPIKey(coreconfig.IntelligenceConfig{APIKeyFile: path})
	if err != nil {
		t.Fatal(err)
	}
	if got != "secret-value" {
		t.Fatalf("unexpected secret resolution %q", got)
	}
}
