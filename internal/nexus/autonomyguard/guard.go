package autonomyguard

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/IAPro-Community/nexus/internal/executil"
)

func AllowedPath(name string, patterns []string) bool {
	name = strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"), "./")
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		pattern = strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(pattern), "\\", "/"), "./")
		if pattern == "" {
			continue
		}
		if ok, _ := path.Match(pattern, name); ok {
			return true
		}
		if strings.Contains(pattern, "**") {
			re := regexp.QuoteMeta(pattern)
			re = strings.ReplaceAll(re, `\*\*`, `.*`)
			re = strings.ReplaceAll(re, `\*`, `[^/]*`)
			re = "^" + re + "$"
			if matched, _ := regexp.MatchString(re, name); matched {
				return true
			}
		}
	}
	return false
}

func ForbiddenGitCommand(args []string, disallowDestructive, allowPush bool) bool {
	cmd, rest := gitSubcommand(args)
	if cmd == "" {
		return false
	}
	if cmd == "push" && !allowPush {
		return true
	}
	if !disallowDestructive {
		return false
	}
	switch cmd {
	case "clean":
		return containsFlag(rest, "-f") || containsFlag(rest, "--force")
	case "reset":
		return containsFlag(rest, "--hard")
	case "branch":
		return containsFlag(rest, "-D") || containsFlag(rest, "--delete-force")
	case "checkout":
		return containsToken(rest, "--")
	case "restore", "merge", "rebase", "cherry-pick":
		return true
	}
	return false
}

func gitSubcommand(args []string) (string, []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-C" || a == "-c" || a == "--git-dir" || a == "--work-tree" || a == "--namespace" || a == "--config-env" {
			i++
			continue
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		return strings.ToLower(a), args[i+1:]
	}
	return "", nil
}

func containsToken(args []string, target string) bool {
	for _, a := range args {
		if a == target {
			return true
		}
	}
	return false
}

func containsFlag(args []string, target string) bool {
	for _, a := range args {
		if a == target {
			return true
		}
		if strings.HasPrefix(target, "-") && !strings.HasPrefix(target, "--") && strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.Contains(strings.TrimPrefix(a, "-"), strings.TrimPrefix(target, "-")) {
			return true
		}
	}
	return false
}

// Snapshot records signatures of paths that are already dirty in a git worktree.
// It allows Nexus to distinguish pre-existing user changes from mutations made
// during one autonomous provider turn.
type Snapshot map[string]string

func ChangedSince(before, after Snapshot) []string {
	seen := map[string]struct{}{}
	for p := range before {
		seen[p] = struct{}{}
	}
	for p := range after {
		seen[p] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		if before[p] != after[p] {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

func ValidateAllowedChanges(changed, patterns []string) error {
	if len(patterns) == 0 {
		return nil
	}
	var rejected []string
	for _, p := range changed {
		if !AllowedPath(p, patterns) {
			rejected = append(rejected, p)
		}
	}
	if len(rejected) > 0 {
		sort.Strings(rejected)
		return fmt.Errorf("autonomy contract rejected out-of-scope file changes: %s", strings.Join(rejected, ", "))
	}
	return nil
}

// GitChangedPaths returns every currently modified/staged/untracked path in a
// git worktree. Autonomous Agents use dedicated worktrees, so this represents
// the cumulative Mission mutation set rather than unrelated canonical changes.
func GitChangedPaths(ctx context.Context, workspace string) ([]string, error) {
	commands := [][]string{
		{"-C", workspace, "diff", "--name-only", "-z"},
		{"-C", workspace, "diff", "--cached", "--name-only", "-z"},
		{"-C", workspace, "ls-files", "--others", "--exclude-standard", "-z"},
	}
	seen := map[string]struct{}{}
	for _, args := range commands {
		out, err := executil.CommandContext(ctx, "git", args...).Output()
		if err != nil {
			return nil, fmt.Errorf("git changed paths: %w", err)
		}
		for _, raw := range bytes.Split(out, []byte{0}) {
			name := strings.TrimSpace(string(raw))
			if name != "" {
				seen[strings.ReplaceAll(name, "\\", "/")] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}
