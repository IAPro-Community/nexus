package spec

import (
	"path/filepath"

	"github.com/IAPro-Community/nexus/internal/runtime"
)

var geminiSpec = Spec{
	ClearLine:        "\x15",
	ID:               "gemini",
	BinaryCandidates: []string{"gemini"},
	EnvKeys: func(profileHome, binDir, profileName string) map[string]string {
		return map[string]string{
			"HOME":            profileHome,
			"GEMINI_CLI_HOME": profileHome,
			"XDG_CONFIG_HOME": filepath.Join(profileHome, ".config"),
			"AI_PROFILE":      profileName,
			"AI_PROVIDER":     "gemini",
			"PATH":            runtime.EnhancedPATH(binDir),
		}
	},
	Kickoff: func(prompt string) []string {
		return []string{"-p", prompt}
	},
	Resume: func(sessionID string) []string {
		return []string{"-r", sessionID}
	},
	AutoApproval:  []string{"--approval-mode=yolo", "--output-format", "text"},
	ReviewMode:    []string{"--approval-mode=plan", "--output-format", "text"},
	Quota:         QuotaLiveHTTP,
	KnownVersions: []string{"0.28.2"},
}
