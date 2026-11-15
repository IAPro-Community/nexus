package claude

import (
	"os"
	"path/filepath"
)

// readClaudeIdentityJSON reads Claude Code's identity state from both known
// locations. Recent Claude Code builds keep profile state under
// `.claude/.claude.json`; legacy builds wrote `$HOME/.claude.json`. The newer
// location wins and the legacy file stays as a fallback for older profiles.
// Without this, a live quota probe can be silently discarded by the account
// attribution gate because the profile's authenticated email is invisible.
func readClaudeIdentityJSON(home string) ([]byte, error) {
	candidates := []string{
		filepath.Join(home, ".claude", ".claude.json"),
		filepath.Join(home, ".claude.json"),
	}
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err == nil {
			return data, nil
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return nil, os.ErrNotExist
}
