package nexus

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Change collision detection (capability change-collision-detector).
//
// The detector is strictly read-only and best-effort: it compares the
// uncommitted changes of every git worktree in a project and reports files
// touched by more than one worktree. It never blocks a merge, never mutates
// state, and always expresses uncertainty explicitly — overlapping hunks are
// high-confidence evidence of a conflict, while "same file, disjoint hunks"
// is only a candidate the owner must review.

// Collision confidence levels.
const (
	// CollisionHunkOverlap: two worktrees edited intersecting line ranges of
	// the same file. Merge will very likely conflict.
	CollisionHunkOverlap = "hunk-overlap"
	// CollisionSameFile: two worktrees edited the same file in disjoint
	// regions. Semantically incompatible changes are still possible.
	CollisionSameFile = "same-file"
)

// Hunk is an inclusive changed line range on the new side of a diff.
type Hunk struct {
	Start int
	End   int
}

// ChangeCollisionWorktree describes one scanned worktree.
type ChangeCollisionWorktree struct {
	Path         string   `json:"path"`
	Branch       string   `json:"branch,omitempty"`
	Head         string   `json:"head,omitempty"`
	ChangedFiles []string `json:"changed_files"`
}

// ChangeCollision reports one file changed by more than one worktree.
type ChangeCollision struct {
	File       string   `json:"file"`
	Worktrees  []string `json:"worktrees"`
	Confidence string   `json:"confidence"`
	Uncertain  bool     `json:"uncertain"`
	Reason     string   `json:"reason"`
}

// ChangeCollisionReport is the full read-only scan result for a project.
type ChangeCollisionReport struct {
	ProjectID      string                     `json:"project_id"`
	ScannedAt      time.Time                  `json:"scanned_at"`
	Worktrees      []ChangeCollisionWorktree  `json:"worktrees"`
	Collisions     []ChangeCollision          `json:"collisions"`
	UncertainCount int                        `json:"uncertain_count"`
	ScanErrors     []ChangeCollisionScanError `json:"scan_errors"`
}

// ChangeCollisionScanError records a worktree that could not be scanned; the
// report stays honest that coverage is partial instead of silently ignoring.
type ChangeCollisionScanError struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// WorktreeChanges carries parsed diff hunks for one worktree.
type WorktreeChanges struct {
	Path  string
	Hunks map[string][]Hunk
}

// hunksIntersect reports whether two inclusive line ranges overlap.
func hunksIntersect(a, b Hunk) bool {
	return a.Start <= b.End && b.Start <= a.End
}

// ParseDiffHunks extracts per-file changed line ranges from a `git diff -U0`
// output. The new-file side (+start,len from the @@ header) is what matters
// for merge conflicts. Paths are read from `+++ b/<path>` lines; deleted
// files (`+++ /dev/null`) contribute nothing because their content disappears
// regardless of who else edited them.
func ParseDiffHunks(diff []byte) map[string][]Hunk {
	hunks := make(map[string][]Hunk)
	currentFile := ""
	for _, line := range strings.Split(string(diff), "\n") {
		switch {
		case strings.HasPrefix(line, "+++ "):
			path := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
			path = strings.TrimPrefix(path, "b/")
			path = strings.Trim(path, "\"")
			if path == "/dev/null" {
				currentFile = ""
				continue
			}
			currentFile = path
		case strings.HasPrefix(line, "@@ ") && currentFile != "":
			newSide, ok := parseHunkNewSide(line)
			if !ok {
				continue
			}
			hunks[currentFile] = append(hunks[currentFile], newSide)
		}
	}
	return hunks
}

// parseHunkNewSide parses `@@ -old,count +new,count @@` and returns the
// inclusive new-side range. A zero-length range (pure insertion point) is
// anchored at its start line so an insertion adjacent to another worktree's
// edit is always surfaced for review instead of silently missed.
func parseHunkNewSide(header string) (Hunk, bool) {
	segments := strings.SplitN(header, " ", 4)
	if len(segments) < 3 {
		return Hunk{}, false
	}
	newSide := strings.TrimSpace(segments[2])
	if !strings.HasPrefix(newSide, "+") {
		return Hunk{}, false
	}
	body := strings.TrimPrefix(newSide, "+")
	if body == "" {
		return Hunk{}, false
	}
	parts := strings.SplitN(body, ",", 2)
	start, err := strconv.Atoi(parts[0])
	if err != nil || start < 0 {
		return Hunk{}, false
	}
	end := start
	if len(parts) == 2 {
		count, err := strconv.Atoi(parts[1])
		if err != nil || count < 0 {
			return Hunk{}, false
		}
		if count > 1 {
			end = start + count - 1
		}
	}
	return Hunk{Start: start, End: end}, true
}

// DetectChangeCollisions compares every pair of worktrees and returns the
// sorted collision list. Determinism: files sort alphabetically, worktree
// paths sort within each collision, so reports are stable for UI snapshots.
func DetectChangeCollisions(changes []WorktreeChanges) []ChangeCollision {
	byFile := make(map[string][]WorktreeChanges)
	for _, change := range changes {
		for file := range change.Hunks {
			byFile[file] = append(byFile[file], change)
		}
	}

	files := make([]string, 0, len(byFile))
	for file := range byFile {
		files = append(files, file)
	}
	sort.Strings(files)

	collisions := make([]ChangeCollision, 0)
	for _, file := range files {
		participants := byFile[file]
		// De-duplicate worktree paths: a worktree appears once per file even
		// if several hunks matched.
		seen := make(map[string]bool, len(participants))
		unique := make([]WorktreeChanges, 0, len(participants))
		for _, participant := range participants {
			if seen[participant.Path] {
				continue
			}
			seen[participant.Path] = true
			unique = append(unique, participant)
		}
		if len(unique) < 2 {
			continue
		}
		paths := make([]string, 0, len(unique))
		for _, participant := range unique {
			paths = append(paths, participant.Path)
		}
		sort.Strings(paths)

		overlap := false
	collect:
		for i := 0; i < len(unique); i++ {
			for j := i + 1; j < len(unique); j++ {
				if rangesOverlap(unique[i].Hunks[file], unique[j].Hunks[file]) {
					overlap = true
					break collect
				}
			}
		}

		reason := fmt.Sprintf(
			"%d worktrees changed %s with disjoint hunks; semantic conflicts are still possible — review before merging",
			len(unique), file,
		)
		confidence := CollisionSameFile
		if overlap {
			confidence = CollisionHunkOverlap
			reason = fmt.Sprintf(
				"%d worktrees changed intersecting line ranges of %s; merge will very likely conflict",
				len(unique), file,
			)
		}
		collisions = append(collisions, ChangeCollision{
			File:       file,
			Worktrees:  paths,
			Confidence: confidence,
			Uncertain:  !overlap,
			Reason:     reason,
		})
	}
	return collisions
}

// rangesOverlap reports whether any pair of ranges intersects.
func rangesOverlap(a, b []Hunk) bool {
	for _, ra := range a {
		for _, rb := range b {
			if hunksIntersect(ra, rb) {
				return true
			}
		}
	}
	return false
}
