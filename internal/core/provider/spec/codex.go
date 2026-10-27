package spec

import (
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/runtime"
)

var codexSpec = Spec{
	ClearLine:        "\x15",
	ID:               "codex",
	BinaryCandidates: []string{"codex"},
	EnvKeys: func(profileHome, binDir, profileName string) map[string]string {
		return map[string]string{
			"HOME":             profileHome,
			"USERPROFILE":      profileHome,
			"CODEX_HOME":       profileHome,
			"CODEX_CONFIG_DIR": profileHome,
			"XDG_CONFIG_HOME":  filepath.Join(profileHome, ".config"),
			"XDG_CACHE_HOME":   filepath.Join(profileHome, ".cache"),
			"XDG_DATA_HOME":    filepath.Join(profileHome, ".local", "share"),
			"XDG_STATE_HOME":   filepath.Join(profileHome, ".local", "state"),
			runtime.EnvCodexDisableKeyboardEnhancement: "1",
			"AI_PROFILE":  profileName,
			"AI_PROVIDER": "codex",
			"PATH":        runtime.EnhancedPATH(binDir),
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

const codexAppServerSocketPathBudgetBytes = 100

var codexNonInteractiveSubcommands = map[string]struct{}{
	"agents": {}, "app-server": {}, "apply": {}, "archive": {}, "cloud": {},
	"completion": {}, "debug": {}, "delete": {}, "doctor": {}, "exec": {},
	"execpolicy": {}, "features": {}, "login": {}, "logout": {}, "mcp": {},
	"migrate-rollouts": {}, "queue": {}, "remote-control": {}, "review": {},
	"sandbox": {}, "stdio-to-uds": {}, "tcp-tunnel": {}, "unarchive": {}, "update": {},
}

// CodexInteractiveArgs disables the managed background app-server when the
// profile's derived control-socket path approaches the platform's
// Unix-domain-socket limit. The conservative 100-byte budget leaves room below
// common platform limits. Headless/provider service subcommands keep their
// native daemon semantics.
func CodexInteractiveArgs(profileHome string, args []string) []string {
	out := append([]string(nil), args...)
	if strings.TrimSpace(profileHome) == "" {
		return out
	}
	for _, arg := range out {
		if strings.EqualFold(strings.TrimSpace(arg), "--no-daemon") {
			return out
		}
	}
	controlSocket := filepath.Join(profileHome, "app-server-control", "app-server-control.sock")
	if len([]byte(controlSocket)) < codexAppServerSocketPathBudgetBytes || !codexUsesInteractiveTUI(out) {
		return out
	}
	return append([]string{"--no-daemon"}, out...)
}

func codexUsesInteractiveTUI(args []string) bool {
	for _, arg := range args {
		token := strings.ToLower(strings.TrimSpace(arg))
		if token == "resume" || token == "fork" {
			return true
		}
		if _, nonInteractive := codexNonInteractiveSubcommands[token]; nonInteractive {
			return false
		}
	}
	return true
}
