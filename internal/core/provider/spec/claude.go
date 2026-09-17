package spec

import (
	"path/filepath"

	"github.com/IAPro-Community/nexus/internal/runtime"
)

var claudeSpec = Spec{
	ClearLine:        "\x15",
	ID:               "claude",
	BinaryCandidates: []string{"claude"},
	EnvKeys: func(profileHome, binDir, profileName string) map[string]string {
		return map[string]string{
			"HOME":              profileHome,
			"USERPROFILE":       profileHome,
			"CLAUDE_CONFIG_DIR": filepath.Join(profileHome, ".claude"),
			"XDG_CONFIG_HOME":   filepath.Join(profileHome, ".config"),
			"AI_PROFILE":        profileName,
			"AI_PROVIDER":       "claude",
			"PATH":              runtime.EnhancedPATH(binDir),
		}
	},
	Kickoff: func(prompt string) []string {
		return []string{"-p", prompt}
	},
	Resume: func(sessionID string) []string {
		return []string{"--resume", sessionID}
	},
	AutoApproval:  []string{"--dangerously-skip-permissions", "--output-format", "text"},
	ReviewMode:    []string{"--permission-mode", "plan", "--output-format", "text"},
	Quota:         QuotaLiveHTTP,
	KnownVersions: []string{"2.1.250"},
}
