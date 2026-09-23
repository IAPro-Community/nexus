package spec

import (
	"path/filepath"

	"github.com/IAPro-Community/nexus/internal/runtime"
)

var codexSpec = Spec{
	ClearLine:        "\x15",
	ID:               "codex",
	BinaryCandidates: []string{"codex"},
	EnvKeys: func(profileHome, binDir, profileName string) map[string]string {
		return map[string]string{
			"HOME":                                   profileHome,
			"USERPROFILE":                            profileHome,
			"CODEX_HOME":                             profileHome,
			"CODEX_CONFIG_DIR":                       profileHome,
			"XDG_CONFIG_HOME":                        filepath.Join(profileHome, ".config"),
			"XDG_CACHE_HOME":                         filepath.Join(profileHome, ".cache"),
			"XDG_DATA_HOME":                          filepath.Join(profileHome, ".local", "share"),
			"XDG_STATE_HOME":                         filepath.Join(profileHome, ".local", "state"),
			"CODEX_TUI_DISABLE_KEYBOARD_ENHANCEMENT": "1",
			"AI_PROFILE":                             profileName,
			"AI_PROVIDER":                            "codex",
			"PATH":                                   runtime.EnhancedPATH(binDir),
		}
	},
	Kickoff: func(prompt string) []string {
		return []string{"exec", prompt}
	},
	Resume: func(sessionID string) []string {
		return []string{"resume", sessionID}
	},
	AutoApproval: []string{"--ask-for-approval", "never", "--sandbox", "workspace-write", "-c", "sandbox_workspace_write.network_access=false"},
	// Codex review turns use the headless exec contract in a read-only
	// sandbox. The executor places these root options before `exec` so the
	// installed CLI accepts them and cannot mutate the implementation worktree.
	ReviewMode:          []string{"--sandbox", "read-only", "--ask-for-approval", "never"},
	AutonomousIsolation: IsolationWorkspaceWriteNoNetwork,
	ReviewIsolation:     IsolationReadOnlyNoNetwork,
	Quota:               QuotaLiveRPC,
	KnownVersions:       []string{"0.50.0"},
}
