package promotion

import (
	"strings"
)

// IsImmediateSubmitLine reports prompts that must receive Enter without waiting
// for Mission promotion. Provider CLIs (Codex, Agy, Claude, …) use leading "/"
// or ":" for in-app commands such as /status; holding Enter while DecideIntent
// / Project Intelligence runs makes every supervised CLI feel frozen.
//
// Empty lines also submit immediately. Nexus reserved prefixes are included:
// the interactive Lead attach path does not intercept /nexus on CmdInput, so
// delaying Enter only freezes the TUI.
func IsImmediateSubmitLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	return strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, ":")
}
