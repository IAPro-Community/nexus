package profile

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/IAPro-Community/nexus/internal/core/security"
)

// KnownCLISignature defines an AI CLI tool signature to look for on the host.
type KnownCLISignature struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Binaries    []string `json:"binaries"`
	ConfigPaths []string `json:"config_paths"`
	Tier        int      `json:"tier"` // 1 = native in Nexus, 2 = broader ecosystem
}

// KnownCLICatalog references terminal coding agents and harnesses curated from
// canonical references such as bradAGI/awesome-cli-coding-agents and MustCodeAl/awesome-ai-cli.
var KnownCLICatalog = []KnownCLISignature{
	{
		ID:          "claude",
		Name:        "Claude Code",
		Binaries:    []string{"claude"},
		ConfigPaths: []string{".claude.json", ".claude"},
		Tier:        1,
	},
	{
		ID:          "codex",
		Name:        "OpenAI Codex",
		Binaries:    []string{"codex"},
		ConfigPaths: []string{".codex"},
		Tier:        1,
	},
	{
		ID:          "gemini",
		Name:        "Google Gemini CLI",
		Binaries:    []string{"gemini"},
		ConfigPaths: []string{".gemini"},
		Tier:        1,
	},
	{
		ID:          "agy",
		Name:        "Antigravity CLI",
		Binaries:    []string{"agy"},
		ConfigPaths: []string{".gemini/antigravity-cli"},
		Tier:        1,
	},
	{
		ID:          "cursor",
		Name:        "Cursor CLI",
		Binaries:    []string{"cursor-agent", "cursor", "agent"},
		ConfigPaths: []string{".cursor"},
		Tier:        1,
	},
	{
		ID:          "opencode",
		Name:        "OpenCode",
		Binaries:    []string{"opencode"},
		ConfigPaths: []string{".local/share/opencode", ".config/opencode"},
		Tier:        1,
	},
	{
		ID:          "zcode",
		Name:        "ZCode",
		Binaries:    []string{"zcode"},
		ConfigPaths: []string{".zcode"},
		Tier:        1,
	},
	// Tier 2 - Broader ecosystem references
	{
		ID:          "aider",
		Name:        "Aider",
		Binaries:    []string{"aider"},
		ConfigPaths: []string{".aidercfg", ".aider.conf.yml"},
		Tier:        2,
	},
	{
		ID:          "goose",
		Name:        "Goose",
		Binaries:    []string{"goose"},
		ConfigPaths: []string{".config/goose"},
		Tier:        2,
	},
	{
		ID:          "gh-copilot",
		Name:        "GitHub Copilot CLI",
		Binaries:    []string{"copilot", "gh"},
		ConfigPaths: []string{".config/gh/hosts.yml"},
		Tier:        2,
	},
	{
		ID:          "openhands",
		Name:        "OpenHands CLI",
		Binaries:    []string{"openhands"},
		ConfigPaths: []string{".openhands"},
		Tier:        2,
	},
	{
		ID:          "plandex",
		Name:        "Plandex",
		Binaries:    []string{"plandex"},
		ConfigPaths: []string{".plandex"},
		Tier:        2,
	},
}

// HostCLIDetection represents a discovered CLI on the host system.
type HostCLIDetection struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	BinaryPath string `json:"binary_path,omitempty"`
	ConfigPath string `json:"config_path,omitempty"`
	Tier       int    `json:"tier"`
	Installed  bool   `json:"installed"`
	HasConfig  bool   `json:"has_config"`
}

// ScanHostCLIs inspects the host environment for all known AI CLI tools.
func ScanHostCLIs(ctx context.Context) []HostCLIDetection {
	hostHome := security.FindHostHome()
	var results []HostCLIDetection

	for _, sig := range KnownCLICatalog {
		var binPath string
		for _, b := range sig.Binaries {
			if path, err := exec.LookPath(b); err == nil {
				binPath = path
				break
			}
		}

		var cfgFound string
		if hostHome != "" {
			for _, cp := range sig.ConfigPaths {
				target := filepath.Join(hostHome, cp)
				if _, err := os.Stat(target); err == nil {
					cfgFound = target
					break
				}
			}
		}

		if binPath != "" || cfgFound != "" {
			results = append(results, HostCLIDetection{
				ID:         sig.ID,
				Name:       sig.Name,
				BinaryPath: binPath,
				ConfigPath: cfgFound,
				Tier:       sig.Tier,
				Installed:  binPath != "",
				HasConfig:  cfgFound != "",
			})
		}
	}
	return results
}
