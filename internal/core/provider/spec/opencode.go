package spec

import (
	"path/filepath"

	"github.com/IAPro-Community/nexus/internal/runtime"
)

var opencodeSpec = Spec{
	ClearLine:        "\x15",
	ID:               "opencode",
	BinaryCandidates: []string{"opencode"},
	EnvKeys: func(profileHome, binDir, profileName string) map[string]string {
		return map[string]string{
			"HOME":                profileHome,
			"OPENCODE_HOME":       profileHome,
			"OPENCODE_CONFIG_DIR": filepath.Join(profileHome, ".config", "opencode"),
			"XDG_DATA_HOME":       filepath.Join(profileHome, ".local", "share"),
			"XDG_CACHE_HOME":      filepath.Join(profileHome, ".cache"),
			"AI_PROFILE":          profileName,
			"AI_PROVIDER":         "opencode",
			"PATH":                runtime.EnhancedPATH(binDir),
		}
	},
	Kickoff: func(prompt string) []string {
		return []string{"run", prompt}
	},
	Resume: func(sessionID string) []string {
		return []string{"-s", sessionID}
	},
	AutoApproval:  []string{"--auto"},
	ReviewMode:    []string{"--permission-mode", "plan"},
	Quota:         QuotaLiveHTTP,
	KnownVersions: []string{"1.18.30"},
}
