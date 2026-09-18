package spec

import (
	"github.com/IAPro-Community/nexus/internal/runtime"
)

var cursorSpec = Spec{
	ClearLine:        "\x15",
	ID:               "cursor",
	BinaryCandidates: []string{"cursor-agent", "agent"},
	EnvKeys: func(profileHome, binDir, profileName string) map[string]string {
		return map[string]string{
			"HOME":              profileHome,
			"USERPROFILE":       profileHome,
			"CURSOR_HOME":       profileHome,
			"CURSOR_CONFIG_DIR": profileHome,
			"AI_PROFILE":        profileName,
			"AI_PROVIDER":       "cursor",
			"PATH":              runtime.EnhancedPATH(binDir),
		}
	},
	Kickoff: func(prompt string) []string {
		if prompt == "" {
			return nil
		}
		return []string{"-p", prompt, "--output-format", "text"}
	},
	Resume: func(sessionID string) []string {
		return []string{"--resume", sessionID}
	},
	AutoApproval:  nil,
	ReviewMode:    []string{"--mode=ask", "--output-format", "text"},
	Quota:         QuotaUnsupported,
	KnownVersions: []string{},
}
