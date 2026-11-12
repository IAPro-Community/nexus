package web

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/executil"
	"github.com/IAPro-Community/nexus/internal/nexus"
)

// maxCollisionScanWorktrees caps git processes per request. Projects beyond
// this are reported as partial coverage instead of unbounded scanning.
const maxCollisionScanWorktrees = 16

// handleProjectChangeCollisions GET /api/v1/projects/{projectID}/change-collisions
//
// Read-only capability (change-collision-detector): compares the uncommitted
// changes of every git worktree in the project and reports files touched by
// more than one worktree. Never mutates git state and never blocks merges;
// the response always carries explicit uncertainty so the owner reviews
// before merging.
func (h *NexusHandler) handleProjectChangeCollisions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	proj, err := st.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	worktrees, err := listProjectGitWorktrees(r.Context(), proj.CanonicalPath)
	if err != nil {
		writeError(w, http.StatusConflict, "change collision scan requires a git repository: "+err.Error())
		return
	}

	report := nexus.ChangeCollisionReport{
		ProjectID:  projectID,
		ScannedAt:  time.Now().UTC(),
		Worktrees:  make([]nexus.ChangeCollisionWorktree, 0, len(worktrees)),
		Collisions: make([]nexus.ChangeCollision, 0),
	}

	scanLimit := len(worktrees)
	truncated := false
	if scanLimit > maxCollisionScanWorktrees {
		scanLimit = maxCollisionScanWorktrees
		truncated = true
	}

	changes := make([]nexus.WorktreeChanges, 0, scanLimit)
	for _, worktree := range worktrees[:scanLimit] {
		diff, err := worktreeUncommittedDiff(r.Context(), worktree.Path)
		if err != nil {
			report.ScanErrors = append(report.ScanErrors, nexus.ChangeCollisionScanError{
				Path:   worktree.Path,
				Reason: err.Error(),
			})
			continue
		}
		hunks := nexus.ParseDiffHunks(diff)
		files := make([]string, 0, len(hunks))
		for file := range hunks {
			files = append(files, file)
		}
		sort.Strings(files)
		if len(files) == 0 {
			continue
		}
		report.Worktrees = append(report.Worktrees, nexus.ChangeCollisionWorktree{
			Path:         worktree.Path,
			Branch:       worktree.Branch,
			Head:         worktree.Head,
			ChangedFiles: files,
		})
		changes = append(changes, nexus.WorktreeChanges{Path: worktree.Path, Hunks: hunks})
	}
	if truncated {
		report.ScanErrors = append(report.ScanErrors, nexus.ChangeCollisionScanError{
			Path:   "*",
			Reason: "project has more than 16 worktrees; scan is partial",
		})
	}
	sort.Slice(report.Worktrees, func(i, j int) bool {
		return report.Worktrees[i].Path < report.Worktrees[j].Path
	})

	report.Collisions = nexus.DetectChangeCollisions(changes)
	for _, collision := range report.Collisions {
		if collision.Uncertain {
			report.UncertainCount++
		}
	}
	writeJSON(w, http.StatusOK, report)
}

// worktreeUncommittedDiff returns `git diff HEAD -U0 --no-color` for the
// worktree (staged + unstaged relative to HEAD), with the same output cap and
// timeout discipline as the read-only worktree listing.
func worktreeUncommittedDiff(parent context.Context, dir string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	output, err := executil.CommandContext(ctx, "git", "-C", dir,
		"diff", "HEAD", "-U0", "--no-color").Output()
	if err != nil {
		return nil, normalizeCollisionScanError(err)
	}
	if len(output) > 2*1024*1024 {
		return nil, errDiffOutputTooLarge
	}
	return output, nil
}

var errDiffOutputTooLarge = collisionScanError("git diff output exceeds allowed limit")

type collisionScanError string

func (e collisionScanError) Error() string { return string(e) }

func normalizeCollisionScanError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "ambiguous argument 'HEAD'"):
		return collisionScanError("worktree has no commits yet; nothing to compare against HEAD")
	case strings.Contains(msg, "not a git repository"):
		return collisionScanError("path is not a git repository")
	default:
		return collisionScanError(strings.TrimSpace(msg))
	}
}
