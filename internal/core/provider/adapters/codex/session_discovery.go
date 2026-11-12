package codex

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// errSessionDiscoveryHomeRequired guards against discovering sessions without
// a resolved profile home.
var errSessionDiscoveryHomeRequired = errors.New("codex session discovery requires a profile home")

// discoveredRollout is one candidate rollout file for live session discovery.
type discoveredRollout struct {
	path     string
	modTime  time.Time
	cwdMatch bool
}

// FindActiveSessionID returns the provider session id of the rollout this
// profile home owns that was written at/after `since`. It is the discovery
// half of supervised session continuity: PTY Codex runtimes cannot report a
// session id over a structured channel, so the host polls the profile's own
// session store for the rollout the live TUI is writing.
//
// Semantics:
//   - Only rollout files under this profile home are considered
//     (<home>/sessions and <home>/.codex/sessions).
//   - Files modified before `since` are ignored: session seeding for
//     CrossAccountResume finishes before the provider process starts, so the
//     live TUI's rollout is strictly newer.
//   - Cross-account resume imports are excluded even when their mtime is
//     recent: they are copies of other profiles' threads, never this session.
//   - When `workspace` is non-empty and at least one candidate records a
//     matching session_meta cwd, cwd-matching candidates win; otherwise the
//     newest candidate wins.
//
// An empty id means "not observed yet" (the TUI has not created its rollout
// yet); callers are expected to keep polling. The walk is strictly read-only
// and never contends with the interactive TUI.
func FindActiveSessionID(home string, since time.Time, workspace string) (string, error) {
	home = strings.TrimSpace(home)
	if home == "" {
		return "", errSessionDiscoveryHomeRequired
	}

	roots := profileSessionRoots(home)
	if len(roots) == 0 {
		return "", nil
	}

	var candidates []discoveredRollout
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			name := d.Name()
			if !strings.HasPrefix(name, "rollout-") || !strings.HasSuffix(name, ".jsonl") {
				return nil
			}
			info, infoErr := d.Info()
			if infoErr != nil {
				return nil
			}
			modTime := info.ModTime()
			if modTime.Before(since) {
				return nil
			}
			// Imported sibling/host rollouts are copies of other threads and
			// must never be adopted as this runtime's session id.
			if isCrossAccountResumeSession(home, path) {
				return nil
			}
			candidates = append(candidates, discoveredRollout{
				path:     path,
				modTime:  modTime,
				cwdMatch: workspace != "" && rolloutCWDMatches(path, workspace),
			})
			return nil
		})
	}

	if len(candidates) == 0 {
		return "", nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].cwdMatch != candidates[j].cwdMatch {
			return candidates[i].cwdMatch
		}
		return candidates[i].modTime.After(candidates[j].modTime)
	})
	best := candidates[0]

	sessionID := extractSessionIDFromRolloutName(filepath.Base(best.path))
	if sessionID == "" {
		// Filename session id is authoritative; only fall back to the
		// session_meta payload id when the filename carried none.
		entry := synthesizeIndexEntry(best.path, "", best.modTime)
		sessionID = strings.TrimSpace(entry.ID)
	}
	return sessionID, nil
}

// profileSessionRoots returns this profile's own rollout trees.
func profileSessionRoots(home string) []string {
	roots := []string{
		filepath.Join(home, "sessions"),
		filepath.Join(home, ".codex", "sessions"),
	}
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		if fi, err := os.Stat(root); err == nil && fi.IsDir() {
			out = append(out, root)
		}
	}
	return out
}

// rolloutCWDMatches reports whether the rollout's session_meta cwd matches the
// runtime workspace. Read failures are treated as "no match" so a truncated
// first line can never fabricate a match.
func rolloutCWDMatches(rolloutPath, workspace string) bool {
	entry := synthesizeIndexEntry(rolloutPath, "", time.Time{})
	cwd := strings.TrimSpace(entry.CWD)
	if cwd == "" {
		return false
	}
	workspace = strings.TrimSpace(workspace)
	return cwd == workspace
}
