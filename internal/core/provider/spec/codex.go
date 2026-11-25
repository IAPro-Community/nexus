package spec

import (
	"path/filepath"
	"strings"
)

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
