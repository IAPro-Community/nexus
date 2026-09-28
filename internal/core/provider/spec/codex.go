package spec

import (
	"path/filepath"
	stdruntime "runtime"
	"strings"

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

const codexUnixSocketSafePathBytes = 100

var codexNonInteractiveSubcommands = map[string]struct{}{
	"agents": {}, "app-server": {}, "apply": {}, "archive": {}, "cloud": {},
	"completion": {}, "debug": {}, "delete": {}, "doctor": {}, "exec": {},
	"execpolicy": {}, "features": {}, "login": {}, "logout": {}, "mcp": {},
	"migrate-rollouts": {}, "queue": {}, "remote-control": {}, "review": {},
	"sandbox": {}, "stdio-to-uds": {}, "tcp-tunnel": {}, "unarchive": {}, "update": {},
}

// CodexInteractiveArgs disables the managed background app-server only when
// the profile's derived Unix-domain control socket would be unsafe for
// SUN_LEN. Isolated Nexus profiles are intentionally deep paths, so recent
// Codex releases can otherwise fail before the TUI starts. Headless/provider
// service subcommands keep their native daemon semantics.
func CodexInteractiveArgs(profileHome string, args []string) []string {
	out := append([]string(nil), args...)
	if stdruntime.GOOS == "windows" || strings.TrimSpace(profileHome) == "" {
		return out
	}
	for _, arg := range out {
		if strings.EqualFold(strings.TrimSpace(arg), "--no-daemon") {
			return out
		}
	}
	controlSocket := filepath.Join(profileHome, "app-server-control", "app-server-control.sock")
	if len([]byte(controlSocket)) < codexUnixSocketSafePathBytes || !codexUsesInteractiveTUI(out) {
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
