package driver

import (
	"context"
	"os"
	"path/filepath"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
	"github.com/IAPro-Community/nexus/internal/core/security"
)

// bootstrapProfile prepares a profile with non-credential conversation
// artifacts so a new supervised runtime can see prior local context.
func bootstrapProfile(provider string, p model.Profile) (string, error) {
	home, err := config.ProfileHome(provider, p.Name)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		return "", err
	}
	cfgObj, _ := config.LoadConfig()
	if err := security.ApplyIsolation(home, security.GetPolicy(cfgObj.IsolationPreset)); err != nil {
		return "", err
	}
	hostHome := security.FindHostHome()
	if hostHome == "" {
		if provider == "codex" {
			// Still run Prepare for CrossAccountResume seed across Nexus profiles.
			_ = codex.New().Prepare(context.Background(), p)
			home, _ = config.ProfileHome(provider, p.Name)
		}
		return home, nil
	}
	switch provider {
	case "codex":
		linkConversationArtifacts(home, filepath.Join(hostHome, ".codex"), []string{
			"rules", "skills", "customizations",
		})
		// Ensure isolated session dirs exist (do not symlink host sessions).
		_ = os.MkdirAll(filepath.Join(home, "sessions"), 0700)
		// Supervised launches previously skipped adapter.Prepare, so
		// CrossAccountResume never seeded into CODEX_HOME. Mirror the direct path.
		if err := codex.New().Prepare(context.Background(), p); err != nil {
			return "", err
		}
		home, err = config.ProfileHome(provider, p.Name)
		if err != nil {
			return "", err
		}
	case "agy":
		linkConversationArtifacts(filepath.Join(home, ".gemini"), filepath.Join(hostHome, ".gemini"), []string{
			"antigravity-cli/history.jsonl", "antigravity-cli/conversation_summaries.db",
		})
	case "opencode":
		linkConversationArtifacts(filepath.Join(home, ".local", "share", "opencode"), filepath.Join(hostHome, ".local", "share", "opencode"), []string{
			"session", "sessions", "storage/session", "storage/message",
		})
	}
	return home, nil
}

func linkConversationArtifacts(destinationRoot, sourceRoot string, items []string) {
	for _, item := range items {
		src := filepath.Join(sourceRoot, item)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		if err := security.SafeLinkOrCopy(src, filepath.Join(destinationRoot, item)); err != nil {
			// Best-effort continuity; missing history must not block bootstrap.
			continue
		}
	}
}
